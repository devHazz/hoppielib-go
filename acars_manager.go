package hoppielibgo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	AcarsRequestUrl = "http://www.hoppie.nl/acars/system/connect.html"
	// Poll Interval when polling new messages (Seconds)
	DefaultPollInterval = 60
)

type ACARSManager struct {
	// Hoppie's ACARS Logon Code
	logon string
	// Your callsign, to logon to CPDLC or make any ACARS request
	callsign *string
	// Channel for receiving parsed ACARS messages
	//
	// Parsed into up/downlink station callsign, message type & message data
	messages chan ACARSMessage
	// ACARSConnection struct for handling ongoing CPDLC connection with a station
	Connection *ACARSConnection
	// Error group for handling goroutines and any forthcoming panics/errors
	ErrGroup *errgroup.Group
	// Options for the ACARSManager
	//
	// Allows certain 'features' to be set like ADS-C Reporting and custom timeouts/intervals
	opts   ACARSManagerOptions
	Ctx    context.Context
	cancel context.CancelFunc
}

type ACARSManagerOptions struct {
	// Enable ADS-C Reporting to ACARS. To be used by downlink stations, reporting altitude, speed, heading etc
	adscReporting bool
	// Amount of time to wait out before timing out a logon with a station. If time exceeds the value set/default value, state will be set to Disconnected
	//
	// If nil, will wait an indefinite amount of time before state change or otherwise
	cpdlcLogonTimeout *int
	// Set a custom ACARS polling interval (Default is 60 seconds)
	pollingInterval int
}

func (o *ACARSManagerOptions) AdsCReporting(enable bool) {
	o.adscReporting = enable
}

func (o *ACARSManagerOptions) LogonTimeout(time int) {
	o.cpdlcLogonTimeout = &time
}

// Set polling interval for ACARS listen
//
// Time value needs to be in seconds, so for example SetPollInterval(30) would be 30 seconds
func (o *ACARSManagerOptions) PollInterval(time int) {
	if !(time <= 0) {
		o.pollingInterval = time
	}
}

func NewACARSManager(logon string, callsign string, opts ...ACARSManagerOptions) *ACARSManager {
	ctx, cancel := context.WithCancel(context.Background())
	group, _ := errgroup.WithContext(ctx)

	options := ACARSManagerOptions{
		adscReporting:     false,
		cpdlcLogonTimeout: nil,
		pollingInterval:   DefaultPollInterval,
	}

	if len(opts) > 0 {
		options = opts[0]
		if options.adscReporting {
			log.Info().
				Bool("ADS-C Reporting", true).
				Msg("Manager Option Added")
		}

		if options.pollingInterval == 0 {
			options.pollingInterval = DefaultPollInterval
		}
	}

	return &ACARSManager{
		logon:    logon,
		callsign: &callsign,
		messages: make(chan ACARSMessage, 1),
		Connection: &ACARSConnection{
			state:   Disconnected,
			rx:      make(chan ConnectionState, 1),
			lastMin: 1,
		},
		opts:     options,
		ErrGroup: group,
		Ctx:      ctx,
		cancel:   cancel,
	}
}

type ConnectionState int

const (
	Connected ConnectionState = iota
	Waiting
	Disconnected
)

// Structure for an ACARS connection, set as disconnected by default
//
// Once logon has been accepted by a downlink station, the state changes to Connected & station field value is no longer nil
type ACARSConnection struct {
	mutex   sync.Mutex
	state   ConnectionState
	rx      chan ConnectionState
	station *string
	lastMin int
}

func (c *ACARSConnection) SetStation(station string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.station = &station
}

func (c *ACARSConnection) Station() *string {
	return c.station
}

func (c *ACARSConnection) IncrementMin() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastMin++
}

func (c *ACARSConnection) PushState(state ConnectionState) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.state = state
	c.rx <- state
}

func (m *ACARSManager) Connect(station string) error {
	if (m.callsign == nil || *m.callsign == "") && station == "" {
		m.cancel()
		return errors.New("invalid fields for logon (no up/downlink stations provided)")
	}

	m.Connection.SetStation(station)

	_, err := MakeRawRequest(m.logon, *m.callsign, station, CpdlcMessageType, MakeCPDLCPacket(
		m.Connection.lastMin,
		nil,
		RespondRequired,
		"REQUEST LOGON",
	))

	if err != nil {
		m.cancel()
		return err
	}

	m.ErrGroup.Go(m.Listen)
	m.Connection.PushState(Waiting)

	return nil
}

func (m *ACARSManager) Close() {
	m.cancel()

	m.ErrGroup.Wait()

	close(m.messages)
	close(m.Connection.rx)
}

func (m *ACARSManager) OnConnected(f func() error) error {
	for {
		select {
		case state := <-m.RecvState():
			switch state {
			case Connected:
				f()
			case Waiting:
				log.Info().
					Str("Station", *m.Connection.Station()).
					Msg("Waiting for Logon")
			default:
				log.Info().
					Str("Station", *m.Connection.Station()).
					Msg("Connection Disconnected")
			}
		case <-m.Ctx.Done():
			return errors.New("manager context done/cancelled")
		}
	}
}

func (m *ACARSManager) ConnectionState() ConnectionState {
	return m.Connection.state
}

func (m *ACARSManager) RecvState() chan ConnectionState {
	return m.Connection.rx
}

func (m *ACARSManager) Listen() error {
	if m.callsign == nil || m.Connection.Station() == nil {
		return errors.New("acars listen: invalid value provided")
	}

	// Create a ticker with a certain interval to make Hoppie happy
	log.Debug().
		Int("Interval", m.opts.pollingInterval).
		Msg("Polling Started")

	// elapsedTime := 0
	ticker := time.NewTicker(time.Duration(m.opts.pollingInterval) * time.Second)
	var timeout <-chan time.Time

	if m.opts.cpdlcLogonTimeout != nil {
		timeout = time.After(time.Duration(*m.opts.cpdlcLogonTimeout) * time.Second)
	}

	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			m.Connection.PushState(Disconnected)
			return errors.New("CPDLC logon timeout reached, pushed Disconnected state")
		case <-ticker.C:
			// elapsedTime = elapsedTime + m.opts.pollingInterval

			// if m.opts.cpdlcLogonTimeout != nil && elapsedTime >= *m.opts.cpdlcLogonTimeout {
			// 	// CPDLC Logon timeout, push state to disconnected
			// }

			data, e := MakeRawRequest(
				m.logon,
				*m.callsign,
				*m.Connection.Station(),
				PollMessageType,
				"",
			)
			if e != nil {
				m.cancel()
				return e
			}

			// We parse any ACARS messages within the data array & check for valid logon accepted CPDLC messages to push to a connected state
			for _, v := range ParseACARSMessage(data) {
				if m.ConnectionState() == Waiting && v.Type == CpdlcMessageType {
					message, e := ParseCPDLCMessage(v.Data)
					if e != nil {
						return e
					}

					if message.Data == "LOGON ACCEPTED" &&
						message.Mrn != nil && *message.Mrn == m.Connection.lastMin &&
						v.Sender == *m.Connection.Station() {

						m.Connection.PushState(Connected)

						log.Info().
							Str("Station", v.Sender).
							Msg("Logon Successful")

					} else {
						log.Debug().
							Str("Stored Station", *m.Connection.Station()).
							Int("Last Recorded MIN", m.Connection.lastMin).
							Dict("Message", zerolog.Dict().
								Str("Sender", v.Sender).
								Str("MRN", NilCheck(message.Mrn)).
								Str("Data", message.Data),
							).
							Msg("Received reply to logon but failed match")
					}
				}

				m.messages <- v
			}
		case <-m.Ctx.Done():
			return m.Ctx.Err()
		}
	}
}

// Sends a CPDLC Request to connected station
//
// For example, providing LOGON ACCEPTED in the data field & OperationalResponse as a RRK
//
// Will result in data being sent with an output like `/data2/min/mrn/NE/LOGON ACCEPTED`
func (m *ACARSManager) CPDLCRequest(data string, rrk ResponseRequirements) error {
	if m.ConnectionState() != Connected || m.Connection.Station() == nil {
		return errors.New("no cpdlc connection made or invalid station")
	}

	m.Connection.IncrementMin()

	packet := MakeCPDLCPacket(
		m.Connection.lastMin,
		nil,
		rrk,
		data,
	)

	_, err := MakeRawRequest(m.logon, *m.callsign, *m.Connection.Station(), CpdlcMessageType, packet)
	if err != nil {
		return err
	}

	return nil
}

func (m *ACARSManager) WeatherRequest(icao string, dataType WeatherRequestType) error {
	if len(icao) < 4 || len(icao) > 4 {
		return errors.New("weather request: invalid icao")
	}

	data, err := MakeRawRequest(m.logon, *m.callsign, "SERVER", InfoRequestMessageType, string(dataType)+" "+icao)
	if err != nil {
		return err
	}

	messages := ParseACARSMessage(data)
	if len(messages) < 1 || len(messages) > 1 {
		return errors.New("weather request: invalid messages len")
	}

	for _, message := range messages {
		m.messages <- message
	}

	return nil
}

// Sends a telex message with the provided body & station to send to
func (m *ACARSManager) Telex(data string, station string) error {
	if data == "" || station == "" {
		return fmt.Errorf("telex request from %s: invalid data", *m.callsign)
	}

	_, err := MakeRawRequest(m.logon, *m.callsign, station, TelexMessageType, data)
	if err != nil {
		return err
	}

	return nil
}

// Basic implementation of how the queue system would look when receiving a message from
func (m *ACARSManager) Recv() chan ACARSMessage {
	return m.messages
}

func MakeCPDLCPacket(
	min int,
	mrn *int,
	rrk ResponseRequirements,
	data string,
) string {
	packet := make([]string, 5)

	// Write protocol type
	packet[0] = "/data2"

	// Write MIN (Message Identification Number
	packet[1] = strconv.Itoa(min)

	// Write MRN (Message Reference Number) if applicable
	if mrn != nil {
		packet[2] = strconv.Itoa(*mrn)
	} else {
		packet[2] = ""
	}

	// Write RRK (Response Requirement Key)
	packet[3] = string(rrk)

	// Finally, write the packet content
	packet[4] = data

	return strings.Join(packet, "/")
}

func MakeRawRequest(
	logon string,
	callsign string,
	station string,
	messageType MessageType,
	content string,
) (string, error) {
	requestParams := url.Values{
		"logon":  {logon},
		"from":   {callsign},
		"to":     {station},
		"type":   {string(messageType)},
		"packet": {content},
	}
	constructedUrl := AcarsRequestUrl + "?" + requestParams.Encode()
	r, e := http.Get(constructedUrl)
	if e != nil {
		return "", fmt.Errorf("failed to send raw request: %w", e)
	}

	defer r.Body.Close()
	data, e := io.ReadAll(r.Body)
	if e != nil {
		return "", fmt.Errorf("could not read response body via io reader: %w", e)
	}

	if strings.HasPrefix(string(data), "ok") {
		// Good response
		return string(data), nil
	} else if strings.HasPrefix(string(data), "error") {
		// Bad response
		errorMessage := string(data)[strings.IndexRune(string(data), '{')+1 : strings.IndexRune(string(data), '}')]
		return "", fmt.Errorf("hoppie acars returned an error from request: %s", errorMessage)
	}

	return "", nil
}

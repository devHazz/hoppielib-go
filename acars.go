package hoppielibgo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	AcarsRequestUrl = "http://www.hoppie.nl/acars/system/connect.html"
	// Poll Interval when polling new messages (Seconds)
	AcarsPollInterval = 20
)

type ACARSManager struct {
	logon               string
	callsign            *string
	messages            chan ACARSMessage
	Connection          *ACARSConnection
	ErrGroup            *errgroup.Group
	inboundPollInterval int
	Ctx                 context.Context
	cancel              context.CancelFunc
}

type ConnectionState int

const (
	Connected ConnectionState = iota
	Waiting
	Disconnected
)

type ACARSConnection struct {
	mutex       sync.Mutex
	state       ConnectionState
	stateChange chan ConnectionState
	station     *string
	lastMin     int
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

func (c *ACARSConnection) SetConnectionState(state ConnectionState) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.state = state
}

func NewACARSManager(logon string, callsign string) *ACARSManager {
	ctx, cancel := context.WithCancel(context.Background())
	group, _ := errgroup.WithContext(ctx)

	return &ACARSManager{
		logon:    logon,
		callsign: &callsign,
		messages: make(chan ACARSMessage, 1),
		Connection: &ACARSConnection{
			state:       Disconnected,
			stateChange: make(chan ConnectionState, 1),
			lastMin:     1,
		},
		inboundPollInterval: AcarsPollInterval, // 20 Second Polling Rate
		ErrGroup:            group,
		Ctx:                 ctx,
		cancel:              cancel,
	}
}

func (m *ACARSManager) Connect(station string) error {
	m.Connection.SetStation(station)

	_, err := MakeRawRequest(m.logon, *m.callsign, station, CpdlcMessage, MakeCPDLCPacket(
		m.Connection.lastMin,
		nil,
		RespondRequired,
		"REQUEST LOGON",
	))

	if err != nil {
		m.cancel()
		return err
	}

	m.Connection.SetConnectionState(Waiting)
	m.Connection.stateChange <- Waiting

	m.ErrGroup.Go(m.Listen)

	return nil
}

func (m *ACARSManager) Close() {
	m.cancel()

	m.ErrGroup.Wait()

	close(m.messages)
	close(m.Connection.stateChange)
}

func (m *ACARSManager) OnConnected(f func() error) error {
	for {
		select {
		case state := <-m.RecvState():
			switch state {
			case Connected:
				f()
			case Waiting:
				fmt.Printf("Waiting for logon with station: %s\n", *m.Connection.Station())
			default:
				fmt.Printf("Connection state change, no longer connected with station: %s\n", *m.Connection.Station())
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
	return m.Connection.stateChange
}

func (m *ACARSManager) Listen() error {
	if m.callsign == nil && m.Connection.Station() == nil {
		return errors.New("both fields for acars connection invalid")
	}

	// Create a ticker with a certain interval to make Hoppie happy
	fmt.Printf("Setup polling with interval of %ds..\n", m.inboundPollInterval)

	ticker := time.NewTicker(time.Duration(m.inboundPollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data, e := MakeRawRequest(
				m.logon,
				*m.callsign,
				*m.Connection.station,
				PollMessage,
				"",
			)
			if e != nil {
				m.cancel()
				return e
			}

			for _, v := range ParseACARSMessage(data) {
				if m.ConnectionState() == Waiting && v.Type == CpdlcMessage {
					message, e := ParseCPDLCMessage(v.Data)
					if e != nil {
						return e
					}

					if message.Data == "LOGON ACCEPTED" && message.Mrn != nil && *message.Mrn == m.Connection.lastMin {
						m.Connection.SetConnectionState(Connected)
						m.Connection.stateChange <- Connected

						fmt.Printf("Received successful logon from station: %s, pushing connected to current state\n", v.Sender)
					} else {
						fmt.Printf("Received reply to logon request, with invalid MRN from station: %s\n", v.Sender)
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

	_, e := MakeRawRequest(m.logon, *m.callsign, *m.Connection.Station(), CpdlcMessage, packet)
	if e != nil {
		return e
	}

	return nil
}

func (m *ACARSManager) Telex(data string) error {
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
	packet := strings.Builder{}

	// Write protocol type
	packet.WriteString("/data2/")

	// Write MIN (Message Identification Number)
	packet.WriteString(fmt.Sprintf("%d/", min))

	// Write MRN (Message Reference Number) if applicable
	if mrn != nil {
		packet.WriteString(fmt.Sprintf("%d/", *mrn))
	} else {
		packet.WriteRune('/')
	}

	// Write RRK (Response Requirement Key)
	packet.WriteString(fmt.Sprintf("%s/", rrk))

	// Finally, write the packet content
	packet.WriteString(data)

	return packet.String()
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

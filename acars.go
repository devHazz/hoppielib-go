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
	connection          *ACARSConnection
	inboundPollInterval int
	ctx                 context.Context
	cancel              context.CancelFunc
}

type ConnectionState int

const (
	Connected ConnectionState = iota
	Waiting
	Disconnected
)

type ACARSConnection struct {
	mutex   sync.Mutex
	state   ConnectionState
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

func (c *ACARSConnection) SetConnectionState(state ConnectionState) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.state = state
}

func (c *ACARSConnection) State() ConnectionState {
	return c.state
}

func NewACARSManager(logon string, callsign string) *ACARSManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ACARSManager{
		logon:    logon,
		callsign: &callsign,
		messages: make(chan ACARSMessage),
		connection: &ACARSConnection{
			state:   Disconnected,
			lastMin: 1,
		},
		inboundPollInterval: AcarsPollInterval, // 20 Second Polling Rate
		ctx:                 ctx,
		cancel:              cancel,
	}
}

func (m *ACARSManager) Connect(station string) error {
	m.connection.SetStation(station)
	m.connection.SetConnectionState(Waiting)

	_, err := MakeRawRequest(m.logon, *m.callsign, station, CpdlcMessage, MakeCPDLCPacket(
		*m.callsign,
		m.connection.lastMin,
		nil,
		Required,
		"REQUEST LOGON",
	))

	fmt.Printf("Sending logon request to station: %s\n", station)

	if err != nil {
		return err
	}

	g, _ := errgroup.WithContext(m.ctx)
	g.Go(m.Listen)

	if err = g.Wait(); err != nil {
		return err
	}

	return nil
}

func (m *ACARSManager) Listen() error {
	if m.callsign == nil && m.connection.station == nil {
		return errors.New("both fields for acars connection invalid")
	}

	go m.listenMessageQueue()

	// Create a ticker with a certain interval to make Hoppie happy
	fmt.Printf("Setup polling with interval of %ds..\n", m.inboundPollInterval)

	for range time.Tick(time.Duration(m.inboundPollInterval) * time.Second) {
		data, e := MakeRawRequest(
			m.logon,
			*m.callsign,
			*m.connection.station,
			PollMessage,
			"",
		)
		if e != nil {
			return e
		}

		if strings.HasPrefix(data, "ok") {
			for _, v := range ParseACARSMessage(data) {
				if m.connection.State() == Waiting && v.Type == CpdlcMessage {
					message, e := ParseCPDLCMessage(v.Data)
					if e != nil {
						return e
					}

					if message.Data == "LOGON ACCEPTED" && *message.Mrn == m.connection.lastMin {
						m.connection.SetConnectionState(Connected)
						fmt.Printf("Received successful logon from station: %s, pushing connected to current state\n", v.Sender)
					}
				}

				m.messages <- v
			}
		}
	}

	return nil
}

// Basic implementation of how the queue system would look when receiving a message from
func (m *ACARSManager) listenMessageQueue() {
	fmt.Println("Setup listening for message queue..")
	for {
		message := <-m.messages
		fmt.Printf("Got message: From=%s, Type=%s (%s), Content=%s\n",
			message.Sender,
			message.Type,
			message.Type.Description(),
			message.Data,
		)
	}
}

func MakeCPDLCPacket(
	callsign string,
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

package hoppielibgo

import (
	"errors"
	"log"
	"regexp"
	"slices"
	"strconv"
	"time"
)

var (
	// ADS Contract Message Errors
	ErrInvalidPrefix       = errors.New("Invalid ADS-C message prefix")
	ErrInvalidFieldCount   = errors.New("Invalid ADS-C field count")
	ErrInvalidADSCFormat   = errors.New("Invalid ADS-C format")
	ErrInvalidHeading      = errors.New("Invalid heading value")
	ErrInvalidPositionData = errors.New("Invalid position data (latitude or longitude)")
	//
	ErrInvalidAdsCInterval = errors.New("Invalid interval provided for ADS Periodic Contract")
	ErrOriginatorMismatch  = errors.New("Assigned originator mismatch with attempted request")
)

type AdsContractManager struct {
	inner      AdsContractInner
	contracts  []*AdsContract
	originator AdsContractOriginator
}

type AdsContractKind int

const (
	Periodic AdsContractKind = iota
	Demand
	Events
)

// Determine the sending station type by originator, to handle specific messages based on the type
type AdsContractOriginator int

const (
	AtsuOriginator AdsContractOriginator = iota
	AircraftOriginator
)

type AdsContractInner struct {
	logon    *string
	Sender   string
	Receiver string
}

type AdsContractMessage struct {
	Inner AdsContractInner
	Data  string
}

// Follows a similar structure to ACARSManager, in terms of listeners & mutex
type AdsContract struct {
	kind     AdsContractKind
	inner    AdsContractInner
	interval int
	Exchange chan AdsContractMessage
}

func NewAdsContractManager(
	logon string,
	callsign string,
	receiver string,
	originator AdsContractOriginator,
) *AdsContractManager {
	return &AdsContractManager{
		inner: AdsContractInner{
			logon:    &logon,
			Sender:   callsign,
			Receiver: receiver,
		},
		originator: originator,
	}
}

func (m *AdsContractManager) CreatePeriodicContract(interval int) (*AdsContract, error) {
	if m.originator != AtsuOriginator {
		return &AdsContract{}, ErrOriginatorMismatch
	}

	// We'll simulate the range and resolution of the time interval as per ICAO GOLD
	const IntervalAllowedStart, IntervalAllowedEnd = 1, 4096

	if interval < IntervalAllowedStart || interval > IntervalAllowedEnd {
		return &AdsContract{}, ErrInvalidAdsCInterval
	}

	_, err := MakeRawRequest(
		*m.inner.logon,
		m.inner.Sender,
		m.inner.Receiver,
		AdsCMessageType,
		"REQUEST PERIODIC "+strconv.Itoa(interval),
	)

	if err != nil {
		return &AdsContract{}, err
	}

	log.Printf("Sent periodic ADS contract request to %s", m.inner.Sender)

	exchange := make(chan AdsContractMessage)
	contract := &AdsContract{Periodic, m.inner, interval, exchange}

	go func() {
		log.Println("Started goroutine & ticker init")
		ticker := time.NewTicker(time.Second * 20)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if data, err := MakeRawRequest(*m.inner.logon,
					m.inner.Sender,
					m.inner.Receiver,
					PollMessageType,
					"",
				); err == nil {
					// Parse raw messages sat in queue
					for _, message := range ParseACARSMessage(data) {
						// Check to make sure is correct type & no current contracts initialized for initial setup
						activeContract := slices.IndexFunc(m.contracts, func(c *AdsContract) bool { return c.kind == Periodic }) != -1
						if message.Type == AdsCMessageType &&
							(len(m.contracts) == 0 || !activeContract) &&
							message.Sender == m.inner.Receiver {

							exp := regexp.MustCompile(`ACK\s+PERIODIC\s+(\d+)`)
							hit := exp.FindStringSubmatch(message.Data)

							if len(hit) > 1 {
								interval := hit[1]
								if i, err := strconv.Atoi(interval); err == nil && i == contract.interval {
									m.contracts = append(m.contracts, contract)
									log.Printf("Periodic contract acknowledged with receiver: %s  Interval: %ss", message.Sender, interval)
								}
							}

							// log.Printf("Received ADS-C Message (No Current Contract)  Sender: %s  Data: %s", message.Sender, message.Data)
						} else if message.Type == AdsCMessageType && len(m.contracts) >= 1 {
							message := AdsContractMessage{m.inner, message.Data}
							contract.Exchange <- message
						}
					}
				} else {
					// TODO: Handle errors with an ErrGroup or similar
					panic(err)
				}
			}
		}
	}()

	return contract, nil
}

func (m *AdsContractManager) GetContractByKind(kind AdsContractKind) (*AdsContract, int) {
	contractIdx := slices.IndexFunc(m.contracts, func(c *AdsContract) bool { return c.kind == Periodic })
	if contractIdx != -1 {
		return m.contracts[contractIdx], contractIdx
	}
	return nil, 0
}

func (m *AdsContractManager) CancelContract(kind AdsContractKind) {
	if contract, idx := m.GetContractByKind(kind); contract != nil {
		close(contract.Exchange)
		m.contracts = slices.Delete(m.contracts, idx, idx+1)
	}
}

type AdsContractEvent string

const (
	WaypointChangeEvent      AdsContractEvent = "WCE"
	LevelRangeDeviationEvent AdsContractEvent = "LRDE"
	LateralDeviationEvent    AdsContractEvent = "LDE"
	VerticalRateChangeEvent  AdsContractEvent = "VRE"
)

// TODO: Integrate ADS-C events and individual groups
func (c *AdsContract) CreateEventsContract(events ...AdsContractEvent) {}

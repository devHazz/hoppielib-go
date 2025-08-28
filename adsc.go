package hoppielibgo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidPrefix       = errors.New("Invalid ADS-C message prefix")
	ErrInvalidFieldCount   = errors.New("Invalid ADS-C field count")
	ErrInvalidADSCFormat   = errors.New("Invalid ADS-C format")
	ErrInvalidHeading      = errors.New("Invalid heading value")
	ErrInvalidPositionData = errors.New("Invalid position data (latitude or longitude)")
)

type ADSCMessage struct {
	Callsign  string
	Time      string
	Latitude  float32
	Longitude float32
	Altitude  int
	Heading   *int
}

func ParseAdsCMessage(message string) (*ADSCMessage, error) {
	if stripped, valid := strings.CutPrefix(message, "REPORT"); valid {
		parts := strings.Split(strings.TrimPrefix(stripped, " "), " ")

		if len(parts) < 5 {
			return nil, errors.Join(ErrInvalidFieldCount, errors.New(fmt.Sprintf("(got %d field values, expected 5 field values)", len(parts))))
		}

		callsign, time, lat, long, alt := parts[0], parts[1], parts[2], parts[3], parts[4]
		var heading *int

		if len(parts) > 5 {
			rawHeading, err := strconv.Atoi(parts[5])
			if err != nil {
				return nil, ErrInvalidHeading
			}

			if rawHeading > 360 || rawHeading < 0 {
				return nil, ErrInvalidHeading
			}

			heading = &rawHeading
		}

		latitude, err := strconv.ParseFloat(lat, 32)
		if err != nil {
			return nil, ErrInvalidPositionData
		}

		longitude, err := strconv.ParseFloat(long, 32)
		if err != nil {
			return nil, ErrInvalidPositionData
		}

		altitude, err := strconv.Atoi(alt)
		if err != nil {
			return nil, ErrInvalidADSCFormat
		}

		return &ADSCMessage{
			Callsign:  callsign,
			Time:      time,
			Latitude:  float32(latitude),
			Longitude: float32(longitude),
			Altitude:  altitude,
			Heading:   heading,
		}, nil
	}

	return nil, ErrInvalidPrefix
}

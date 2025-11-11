package hoppielibgo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type AdsContractReport struct {
	Callsign  string
	Time      string
	Latitude  float32
	Longitude float32
	Altitude  int
	Heading   *int
}

func ParseAdsCReport(message string) (*AdsContractReport, error) {
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

		return &AdsContractReport{
			Callsign:  callsign,
			Time:      time[len(time)-4:],
			Latitude:  float32(latitude),
			Longitude: float32(longitude),
			Altitude:  altitude,
			Heading:   heading,
		}, nil
	}

	return nil, ErrInvalidPrefix
}

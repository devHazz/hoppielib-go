package hoppielibgo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidCPDLCFormat = errors.New("Invalid CPDLC format")
)

type ResponseRequirements string

const (
	RespondWilcoUnable         ResponseRequirements = "WU"
	RespondAffirmNegative      ResponseRequirements = "AN"
	RespondRoger               ResponseRequirements = "R"
	RespondOperationalResponse ResponseRequirements = "NE"
	RespondRequired            ResponseRequirements = "Y"
	RespondNotRequired         ResponseRequirements = "N"
)

var responseRequirementMap = map[ResponseRequirements]string{
	RespondWilcoUnable:         "Wilco or Unable",
	RespondAffirmNegative:      "Affirm or Negative",
	RespondRoger:               "Roger",
	RespondOperationalResponse: "Operational Response Required",
	RespondRequired:            "Response Required",
	RespondNotRequired:         "Response Not Required",
}

func (rrk *ResponseRequirements) Description() string {
	return responseRequirementMap[*rrk]
}

func isValidResponseRequirement(rrk string) bool {
	return responseRequirementMap[ResponseRequirements(rrk)] != ""
}

type CPDLCMessage struct {
	Min  int
	Mrn  *int
	Rrk  ResponseRequirements
	Data string
}

func ParseCPDLCMessage(data string) (*CPDLCMessage, error) {
	if stripped, valid := strings.CutPrefix(data, "/data2/"); valid {
		parts := strings.Split(stripped, "/")

		if len(parts) != 4 {
			return nil, ErrInvalidCPDLCFormat
		}

		min, mrn, rrk, data := parts[0], parts[1], parts[2], parts[3]

		minCast, err := strconv.Atoi(min)
		if err != nil {
			return nil, fmt.Errorf("MIN: %s is not a valid identification number", min)
		}

		// Since the MRN is an optional, we prevent nil pointer deref and check if nil then cast
		var mrnOptional *int

		if mrn != "" {
			mrnCast, err := strconv.Atoi(mrn)
			if err != nil {
				return nil, fmt.Errorf("MRN: %v is not a valid message reference number", mrn)
			}

			mrnOptional = &mrnCast
		}

		if !isValidResponseRequirement(rrk) {
			return nil, fmt.Errorf("Key: %s is not a valid response requirement key", rrk)
		}

		return &CPDLCMessage{
			Min:  minCast,
			Mrn:  mrnOptional,
			Rrk:  ResponseRequirements(rrk),
			Data: data,
		}, nil
	}

	return nil, ErrInvalidCPDLCFormat
}

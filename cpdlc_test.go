package hoppielibgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCPDLCMessageValid(t *testing.T) {
	message := "/data2/102//Y/REQUEST LOGON"
	parsed, err := ParseCPDLCMessage(message)
	if err != nil {
		assert.Error(t, err)
	}

	assert.Equal(t, 102, parsed.Min)
	// Check MRN, as none provided
	assert.Nil(t, parsed.Mrn)
	// Check Response Requirement Key, should match as 'Y'
	assert.Equal(t, RespondRequired, parsed.Rrk)

	assert.Equal(t, "REQUEST LOGON", parsed.Data)
}

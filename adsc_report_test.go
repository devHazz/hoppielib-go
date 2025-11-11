package hoppielibgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAdsCReportValid(t *testing.T) {
	message := "REPORT BAW47C 91909 57.66224 12.28856 30006 247"
	report, err := ParseAdsCReport(message)
	if err != nil {
		assert.Error(t, err)
	}

	assert.Equal(t, "BAW47C", report.Callsign)

	assert.Equal(t, "1909", report.Time)

	// Assert position data
	assert.Equal(t, float32(57.66224), report.Latitude)
	assert.Equal(t, float32(12.28856), report.Longitude)

	assert.Equal(t, 30006, report.Altitude)

	assert.Equal(t, 247, *report.Heading)
}

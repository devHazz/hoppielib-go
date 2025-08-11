package hoppielibgo

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func GetStatusNotams(client *http.Client) ([]string, error) {
	r, e := client.Get(StatusRequestUrl)
	if e != nil {
		return nil, fmt.Errorf("failed to fetch hoppie status: %w", e)
	}

	defer r.Body.Close()

	var status Status
	if e = json.NewDecoder(r.Body).Decode(&status); e != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", e)
	}

	return status.Notams, nil
}

// Brief function to assist with nil pointer deref checking, particularly used for MRNs
func NilCheck[T any](v *T) string {
	if v != nil {
		return fmt.Sprintf("%v", *v)
	} else {
		return "nil"
	}
}

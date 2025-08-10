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

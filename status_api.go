package hoppielibgo

const StatusRequestUrl = "https://www.hoppie.nl/acars/system/status.html"

type Status struct {
	StatusCode     string      `json:"status_code"`
	SystemTime     string      `json:"system_time"`
	Message        string      `json:"message,omitempty"`
	LoadPercentage float32     `json:"system_load_percent"`
	UserCount      OnlineUsers `json:"online_users"`
	Notams         []string    `json:"notams"`
}

type OnlineUsers struct {
	IVAO   int
	None   int
	VATSIM int
}

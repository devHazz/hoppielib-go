package hoppielibgo

type MessageType string

type Message struct {
	Id       int
	Type     MessageType
	Network  string
	Sender   string
	Receiver string
	Data     string
}

const (
	InfoRequestMessage MessageType = "inforeq"
	DataRequestMessage MessageType = "datareq"
	AdsCMessage        MessageType = "ads-c"
	TelexMessage       MessageType = "telex"
	CpdlcMessage       MessageType = "cpdlc"
	ProgressMessage    MessageType = "progress"
	PeekMessage        MessageType = "peek"
	PollMessage        MessageType = "poll"
	PingMessage        MessageType = "ping"
)

var messageTypeDescription = map[MessageType]string{
	InfoRequestMessage: "Information Request",
	DataRequestMessage: "Data Request",
	AdsCMessage:        "ADS-C",
	TelexMessage:       "Telex",
	CpdlcMessage:       "CPDLC",
	ProgressMessage:    "Progress Report",
	PeekMessage:        "Message Peek",
	PollMessage:        "Communication Server Message Poll",
	PingMessage:        "Connection Ping",
}

func (t *MessageType) Description() string {
	return messageTypeDescription[*t]
}

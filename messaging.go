package hoppielibgo

// TODO: write up handling for general ACARS log messages
// This is currently placeholder code, ready for implementation

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
	InfoRequestMessageType MessageType = "inforeq"
	DataRequestMessageType MessageType = "datareq"
	AdsCMessageType        MessageType = "ads-c"
	TelexMessageType       MessageType = "telex"
	CpdlcMessageType       MessageType = "cpdlc"
	ProgressMessageType    MessageType = "progress"
	PeekMessageType        MessageType = "peek"
	PollMessageType        MessageType = "poll"
	PingMessageType        MessageType = "ping"
)

var messageTypeDescription = map[MessageType]string{
	InfoRequestMessageType: "Information Request",
	DataRequestMessageType: "Data Request",
	AdsCMessageType:        "ADS-C",
	TelexMessageType:       "Telex",
	CpdlcMessageType:       "CPDLC",
	ProgressMessageType:    "Progress Report",
	PeekMessageType:        "Message Peek",
	PollMessageType:        "Communication Server Message Poll",
	PingMessageType:        "Connection Ping",
}

func (t *MessageType) Description() string {
	return messageTypeDescription[*t]
}

type WeatherRequestType string

// Weather Request Types (Information Request Message)
const (
	Default       WeatherRequestType = "metar"
	TAF           WeatherRequestType = "taf"
	ShortTAF      WeatherRequestType = "shorttaf"
	VatsimATIS    WeatherRequestType = "vatatis"
	PilotEdgeATIS WeatherRequestType = "peatis"
	IvaoATIS      WeatherRequestType = "ivaoatis"
)

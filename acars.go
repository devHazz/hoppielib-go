package hoppielibgo

import (
	"regexp"
)

type ACARSMessage struct {
	Sender string
	Type   MessageType
	Data   string
}

func (m *ACARSMessage) Reply(logon string, callsign string, data string) error {
	data, err := MakeRawRequest(logon,
		callsign,
		m.Sender,
		m.Type,
		data,
	)

	if err != nil {
		return err
	}

	return nil
}

func ParseACARSMessage(data string) (messages []ACARSMessage) {
	expr := regexp.MustCompile(`\{([A-Za-z0-9_]+)\s+([A-Za-z0-9-]+)\s+(\{[^}]+\})\}`)
	matches := expr.FindAllStringSubmatch(data, -1)

	for _, m := range matches {
		messages = append(messages, ACARSMessage{
			Sender: m[1],
			Type:   MessageType(m[2]),
			Data:   m[3][1 : len(m[3])-1],
		})
	}

	return messages
}

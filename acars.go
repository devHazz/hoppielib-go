package hoppielibgo

import (
	"regexp"
)

type ACARSMessage struct {
	Sender string
	Type   MessageType
	Data   string
}

func ParseACARSMessage(data string) (messages []ACARSMessage) {
	expr := regexp.MustCompile(`\{([A-Za-z0-9]+)\s+([a-z]+)\s+(\{[^}]+\})\}`)
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

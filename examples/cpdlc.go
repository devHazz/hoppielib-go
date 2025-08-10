package main

import (
	"fmt"

	lib "github.com/devHazz/hoppielib-go"
)

const hoppieLogon = "XXXXXXXXXXXXXX"

func main() {
	sender, receiver := "WLS1", "WLS2"
	manager := lib.NewACARSManager(hoppieLogon, sender)

	// Setup CPDLC Connection with Receiving Station by sending a REQUEST LOGON message to WLS2
	e := manager.Connect(receiver)
	if e != nil {
		fmt.Println(e)
	}
}

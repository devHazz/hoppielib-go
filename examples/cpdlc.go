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
	manager.ErrGroup.Go(func() error {
		return manager.Connect(receiver)
	})

	go manager.HandleConnectedState(func() {
		// Make a generic request once connected to REQUEST CLIMB TO FL330
		// Might need to add this state handle to errgroup to catch any errors within this block
		manager.CPDLCRequest("REQUEST CLIMB TO FL330", lib.RespondRequired)
	})

	// Spin up goroutine for processing incoming messages
	manager.ErrGroup.Go(func() error {
		for {
			message := <-manager.Recv()
			fmt.Printf("Received ACARS Message from Station: %s | Type=%s, Data=%s\n", message.Sender, message.Type, message.Data)
			if message.Type == lib.CpdlcMessage {
				m, e := lib.ParseCPDLCMessage(message.Data)
				if e != nil {
					return e
				}

				// Basic view for CPDLC decode within command line
				fmt.Printf("===CPDLC===\nMIN: %d\nMRN: %s\nData: %s\n===============\n", m.Min, lib.NilCheck(m.Mrn), m.Data)
			}
		}
	})

	if err := manager.ErrGroup.Wait(); err != nil {
		// Generic error handling, TODO: write proper error handling system
		fmt.Println(err)
	}

}

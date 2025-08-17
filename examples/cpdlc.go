package main

import (
	"flag"
	"fmt"

	lib "github.com/devHazz/hoppielib-go"
)

// This example is structured around having 2 goroutines, which is handled by the ACARSManager ErrGroup to handle errors with concurrency
//
// You could purely do this synchronous if you'd like with a for select loop for handling message events
//
// The example provided will receive messages in the background whilst initiating a logon request with the station you provide.
// Then once successfully logged on, sends an immediate request to climb to flight level 330. Just a generic example
func main() {
	logon := flag.String("logon", "", "Hoppie Logon Code")
	sender := flag.String("tx", "", "Sender station (Your callsign)")
	receiver := flag.String("rx", "", "Receiving station")

	flag.Parse()

	manager := lib.NewACARSManager(*logon, *sender)

	// Setup CPDLC Connection with Receiving Station by sending a REQUEST LOGON message to WLS2
	if err := manager.Connect(*receiver); err != nil {
		fmt.Println(err)
	}

	manager.ErrGroup.Go(func() error {
		return manager.OnConnected(func() error {
			// Make a generic request once connected to REQUEST CLIMB TO FL330
			return manager.CPDLCRequest("REQUEST CLIMB TO FL330", lib.RespondRequired)
		})

	})

	// Spin up goroutine for processing incoming messages
	manager.ErrGroup.Go(func() error {
		for {
			select {
			case message := <-manager.Recv():
				fmt.Printf("Received ACARS Message from Station: %s | Type=%s, Data=%s\n", message.Sender, message.Type, message.Data)
				if message.Type == lib.CpdlcMessage && manager.ConnectionState() == lib.Connected {
					m, e := lib.ParseCPDLCMessage(message.Data)
					if e != nil {
						return e
					}

					// Basic view for CPDLC decode within command line
					fmt.Printf("===CPDLC===\nMIN: %d\nMRN: %s\nData: %s\n===============\n", m.Min, lib.NilCheck(m.Mrn), m.Data)
				}

			case <-manager.Ctx.Done():
				return manager.Ctx.Err()
			}

		}
	})

	if err := manager.ErrGroup.Wait(); err != nil {
		manager.Close()
		fmt.Println(err)
	}

}

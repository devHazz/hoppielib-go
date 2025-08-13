package main

import (
	"fmt"

	lib "github.com/devHazz/hoppielib-go"
)

const hoppieLogon = "pK95WpehdwX94Jd7"

func main() {
	sender, receiver := "WLS1", "WLS2"
	manager := lib.NewACARSManager(hoppieLogon, sender)

	// Setup CPDLC Connection with Receiving Station by sending a REQUEST LOGON message to WLS2
	if err := manager.Connect(receiver); err != nil {
		fmt.Println(err)
	}

	manager.ErrGroup.Go(func() error {
		err := manager.OnConnected(func() error {
			// Make a generic request once connected to REQUEST CLIMB TO FL330
			if err := manager.CPDLCRequest("REQUEST CLIMB TO FL330", lib.RespondRequired); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil
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

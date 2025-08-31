package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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

	// Setup our zerolog consts and default values
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	var opts lib.ACARSManagerOptions
	opts.PollInterval(30)
	// opts.LogonTimeout(10)

	manager := lib.NewACARSManager(*logon, *sender, opts)

	// Setup CPDLC Connection with Receiving Station by sending a REQUEST LOGON message to WLS2
	if err := manager.Connect(*receiver); err != nil {
		log.Error().Err(err).Msg("Manager Connect Error")
	}

	// manager.ErrGroup.Go(func() error {
	// 	return manager.OnConnected(func() error {
	// 		// Make a generic request once connected to REQUEST CLIMB TO FL330
	// 		return manager.CPDLCRequest("REQUEST CLIMB TO FL330", lib.RespondRequired)
	// 	})
	// })

	// Spin up goroutine for processing incoming messages
	manager.ErrGroup.Go(func() error {
		for {
			select {
			case message := <-manager.Recv():
				log.Info().
					Str("Station", message.Sender).
					Str("Type", fmt.Sprintf("%s (%s)", message.Type, message.Type.Description())).
					Str("Data", message.Data).
					Msg("ACARS Message")

				if message.Type == lib.CpdlcMessageType && manager.ConnectionState() == lib.Connected {
					m, e := lib.ParseCPDLCMessage(message.Data)
					if e != nil {
						return e
					}

					// Basic view for CPDLC decode within command line
					log.Info().
						Int("MIN", m.Min).
						Str("MRN", lib.NilCheck(m.Mrn)).
						Str("RRK", fmt.Sprintf("%s (%s)", m.Rrk, m.Rrk.Description())).
						Str("Data", m.Data).
						Msg("CPDLC Message")
				}

			case <-manager.Ctx.Done():
				return manager.Ctx.Err()
			}

		}
	})

	if err := manager.ErrGroup.Wait(); err != nil {
		manager.Close()
		log.Err(err).Msg("ACARS Manager Error")
	}

}

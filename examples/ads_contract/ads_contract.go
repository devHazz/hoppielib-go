package main

import (
	"flag"
	"fmt"
	"os"

	lib "github.com/devHazz/hoppielib-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	logon := flag.String("logon", "", "Hoppie Logon Code")
	sender := flag.String("tx", "", "Sender station (Your callsign)")
	receiver := flag.String("rx", "", "Receiving station")

	flag.Parse()

	// Setup our zerolog consts and default values
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// View debug logs via global DebugLevel
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Setup ADS-C Manager as an ATSU
	manager := lib.NewAdsContractManager(
		*logon,
		*sender,
		*receiver,
		lib.AtsuOriginator,
	)

	// Initiate a periodic contract for 300 seconds, with station set in contract manager
	// This method will also handle the 'handshake' that happens with the aircraft
	// As soon as the handshake is successful, any messages that come through on the contract are put through the exchange
	periodicContract, err := manager.CreatePeriodicContract(300)
	if err != nil {
		panic(err)
	}

	// TODO: expand on events contract, as soon as syntax and format is figured out

	for {
		recvMessage := <-periodicContract.Exchange
		report, err := lib.ParseAdsCReport(recvMessage.Data)
		if err != nil {
			// TODO: handle errors properly, refer to ACARSManager
			panic(err)
		}

		event := log.Info().
			Str("Sender", report.Callsign).
			Str("Datetime", report.Time).
			Float32("Latitude", report.Latitude).
			Float32("Longitude", report.Longitude).
			Str("Altitude", fmt.Sprintf("%dft", report.Altitude))

		if report.Heading != nil {
			event.Int("Heading", *report.Heading)
		}

		event.Msg("Periodic ADS-C Message")

		// log.Printf("Periodic ADS-C Message   Sender: %s, Time: %s, Position: (%f, %f), Altitude: %dft, Heading: %s°",
		// 	parsedMessage.Callsign, parsedMessage.Time, parsedMessage.Latitude, parsedMessage.Longitude, parsedMessage.Altitude, lib.NilCheck(parsedMessage.Heading),
		// )
	}
}

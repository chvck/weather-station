package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const readInterval = 5 * time.Second

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func main() {
	atmosProvider := NewBME280SensorProvider(DefaultBME280Addr)
	err := atmosProvider.Connect()
	if err != nil {
		log.WithError(err).Panic("failed to connect to atmospherics provider")
	}

	producer := NewSensorProducer(atmosProvider)
	go func() {
		producer.Run(readInterval)
	}()

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt)
	defer signal.Stop(stopSig)

	<-stopSig

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-producer.Stop()
		wg.Done()
	}()

	wg.Wait()
	atmosProvider.Disconnect()
	fmt.Println("Graceful shutdown completed")
}

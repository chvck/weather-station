package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/chvck/weatherstn"

	log "github.com/sirupsen/logrus"
)

const readInterval = 30 * time.Second

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
	atmosProvider := weatherstn.NewBME280SensorProvider(weatherstn.DefaultBME280Addr, weatherstn.DefaultI2CBusDevice)
	err := atmosProvider.Connect()
	if err != nil {
		log.WithError(err).Panic("failed to connect to atmospherics provider")
	}

	windProvider := weatherstn.NewSEN08942WindSensorProvider(weatherstn.SEN08942WindSensorProviderConfig{
		AnemPinNumber:     5,
		AnemInterval:      5 * time.Second,
		VaneCSPinNumber:   8,
		VaneDOutPinNumber: 9,
		VaneDInPinNumber:  10,
		VaneClkPinNumber:  11,
		VaneChannel:       0,
	})
	err = windProvider.Connect()
	if err != nil {
		log.WithError(err).Panic("failed to connect to wind provider")
	}

	rainProvider := weatherstn.NewSEN08942RainSensorProvider(weatherstn.SEN08942RainSensorProviderConfig{
		PinNumber: 6,
		Interval:  5 * time.Second,
	})
	err = rainProvider.Connect()
	if err != nil {
		log.WithError(err).Panic("failed to connect to rain provider")
	}

	producer := weatherstn.NewSensorProducer(atmosProvider, windProvider, rainProvider)
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
	windProvider.Disconnect()
	rainProvider.Disconnect()
	fmt.Println("Graceful shutdown completed")
}

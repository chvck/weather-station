package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// SensorProducer collects weather station readings from sensors.
type SensorProducer struct {
	atmosProvider AtmosphericSensorProvider
	stopCh        chan struct{}
	stoppedCh     chan struct{}
}

// NewSensorProducer creates and returns a SensorProducer.
func NewSensorProducer(atmosProvider AtmosphericSensorProvider) *SensorProducer {
	return &SensorProducer{
		atmosProvider: atmosProvider,
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}
}

func (sp *SensorProducer) poll() AtmoshphericReadings {
	atmosReadings, err := sp.atmosProvider.Readings()
	if err != nil {
		log.WithError(err).WithField("event", "atmospheric readings")
	}

	return *atmosReadings
}

// Run starts the collector for gathering and saving readings.
func (sp *SensorProducer) Run(interval time.Duration) {
	for {
		atmosReadings := sp.poll()

		fmt.Printf("Humidity: %f, Temperature: %f, Pressure: %f\n", atmosReadings.Humidity, atmosReadings.Temperature,
			atmosReadings.Pressure)

		select {
		case <-sp.stopCh:
			sp.stoppedCh <- struct{}{}
			return
		case <-time.After(interval):
		}
	}
}

// Stop causes the run loop to be halted, returning a channel that is written to when the loop has completed any
// work.
func (sp *SensorProducer) Stop() chan struct{} {
	sp.stopCh <- struct{}{}
	return sp.stoppedCh
}

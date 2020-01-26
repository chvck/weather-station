package weatherstn

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// SensorProducer collects weather station readings from sensors.
type SensorProducer struct {
	atmosProvider AtmosphericSensorProvider
	windProvider  WindSensorProvider
	rainProvider  RainSensorProvider
	stopCh        chan struct{}
	stoppedCh     chan struct{}
}

// NewSensorProducer creates and returns a SensorProducer.
func NewSensorProducer(atmosProvider AtmosphericSensorProvider, windProvider WindSensorProvider,
	rainProvider RainSensorProvider) *SensorProducer {
	return &SensorProducer{
		atmosProvider: atmosProvider,
		windProvider:  windProvider,
		rainProvider:  rainProvider,
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}
}

func (sp *SensorProducer) poll() (*AtmoshphericReadings, *WindReadings, *RainReadings) {
	atmosReadings, err := sp.atmosProvider.Readings()
	if err != nil {
		log.WithError(err).WithField("event", "atmospheric readings")
	}

	windReadings, err := sp.windProvider.Readings()
	if err != nil {
		log.WithError(err).WithField("event", "wind readings")
	}

	rainReadings, err := sp.rainProvider.Readings()
	if err != nil {
		log.WithError(err).WithField("event", "rain readings")
	}

	return atmosReadings, windReadings, rainReadings
}

// Run starts the collector for gathering and saving readings.
func (sp *SensorProducer) Run(interval time.Duration) {
	for {
		select {
		case <-sp.stopCh:
			sp.stoppedCh <- struct{}{}
			return
		case <-time.After(interval):
		}

		atmosReadings, windReadings, rainReadings := sp.poll()

		fmt.Printf("Humidity: %f, Temperature: %f, Pressure: %f\n", atmosReadings.Humidity, atmosReadings.Temperature,
			atmosReadings.Pressure)
		fmt.Printf("Wind speed: %f, Direction: %f, Gust: %f\n", windReadings.Speed, windReadings.Direction,
			windReadings.Gust)
		fmt.Printf("Railfall Total: %f\n", rainReadings.Rainfall)
	}
}

// Stop causes the run loop to be halted, returning a channel that is written to when the loop has completed any
// work.
func (sp *SensorProducer) Stop() chan struct{} {
	sp.stopCh <- struct{}{}
	return sp.stoppedCh
}

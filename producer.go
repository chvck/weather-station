package weatherstn

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// SensorProducer collects weather station readings from sensors.
type SensorProducer struct {
	atmosProvider AtmosphericSensorProvider
	windProvider  WindSensorProvider
	rainProvider  RainSensorProvider
	datastore     DataStore
	stopCh        chan struct{}
}

// NewSensorProducer creates and returns a SensorProducer.
func NewSensorProducer(atmosProvider AtmosphericSensorProvider, windProvider WindSensorProvider,
	rainProvider RainSensorProvider, store DataStore) *SensorProducer {
	return &SensorProducer{
		atmosProvider: atmosProvider,
		windProvider:  windProvider,
		rainProvider:  rainProvider,
		datastore:     store,
		stopCh:        make(chan struct{}),
	}
}

func (sp *SensorProducer) poll() (AtmoshphericReadings, WindReadings, RainReadings) {
	atmosReadings, err := sp.atmosProvider.Readings()
	if err != nil {
		atmosReadings = &AtmoshphericReadings{}
		log.WithError(err).WithField("event", "atmospheric readings")
	}

	windReadings, err := sp.windProvider.Readings()
	if err != nil {
		windReadings = &WindReadings{}
		log.WithError(err).WithField("event", "wind readings")
	}

	rainReadings, err := sp.rainProvider.Readings()
	if err != nil {
		rainReadings = &RainReadings{}
		log.WithError(err).WithField("event", "rain readings")
	}

	return *atmosReadings, *windReadings, *rainReadings
}

// Run starts the collector for gathering and saving readings.
func (sp *SensorProducer) Run(interval time.Duration) {
	for {
		select {
		case <-sp.stopCh:
			return
		case <-time.After(interval):
		}

		t := time.Now().Unix()
		atmosReadings, windReadings, rainReadings := sp.poll()

		err := sp.datastore.Write(WeatherDataRow{
			Timestamp:       t,
			AtmosReadings:   atmosReadings,
			WindReadings:    windReadings,
			RainReadings:    rainReadings,
			IntervalSeconds: int(interval.Seconds()),
		})
		if err != nil {
			log.WithError(err).
				WithField("component", "SensorProducer").
				WithField("event", "store").
				Error("failed to write sensor data to store")
		}
	}
}

// Stop causes the run loop to be halted, returning once the run loop has completed any work.
func (sp *SensorProducer) Stop() {
	sp.stopCh <- struct{}{}
	return
}

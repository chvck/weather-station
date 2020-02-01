package weatherstn

import (
	"github.com/maciej/bme280"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/io/i2c"
)

// AtmosphericSensorProvider provides a way to setup and collect atmospheric data readings.
type AtmosphericSensorProvider interface {
	SensorProvider
	Readings() (*AtmoshphericReadings, error)
}

// AtmoshphericReadings are the sensor readings about measurements such as air temperature.
type AtmoshphericReadings struct {
	Temperature float64 `json:"temperature"`
	Pressure    float64 `json:"pressure"`
	Humidity    float64 `json:"humidity"`
}

// BME280SensorProvider provides temperature, pressure, and humidity readings using the BME280 chip.
type BME280SensorProvider struct {
	i2cAddr int
	i2cBus  string

	driver *bme280.Driver
}

// BME280SensorProviderConfig is used for configuring the BME280.
type BME280SensorProviderConfig struct {
	I2cAddr      int    `json:"i2cAddr"`
	I2cBusDevice string `json:"i2cBusDevice"`
}

// NewBME280SensorProvider creates and returns a BME280SensorProvider.
func NewBME280SensorProvider(config BME280SensorProviderConfig) *BME280SensorProvider {
	return &BME280SensorProvider{
		i2cAddr: config.I2cAddr,
		i2cBus:  config.I2cBusDevice,
	}
}

// Connect initialises the BME280 connection and ensures that readings work correctly.
func (bme *BME280SensorProvider) Connect() error {
	device, err := i2c.Open(&i2c.Devfs{Dev: bme.i2cBus}, bme.i2cAddr)
	if err != nil {
		return err
	}

	driver := bme280.New(device)

	// IBM recommended settings for weather stations
	err = driver.InitWith(bme280.ModeForced, bme280.Settings{
		Filter:                  bme280.FilterOff,
		PressureOversampling:    bme280.Oversampling1x,
		TemperatureOversampling: bme280.Oversampling1x,
		HumidityOversampling:    bme280.Oversampling1x,
	})
	if err != nil {
		if deviceErr := device.Close(); deviceErr != nil {
			log.WithError(deviceErr).
				WithField("component", "atmospheric provider").
				Error("device failed to close")
		}

		return err
	}

	// Check that a read succeeds on the driver
	_, err = driver.Read()
	if err != nil {
		if driverErr := driver.Close(); driverErr != nil {
			log.WithError(driverErr).
				WithField("component", "atmospheric provider").
				Error("driver failed to close")
		}

		return err
	}

	bme.driver = driver

	return nil
}

// Readings returns the set of AtmoshphericReadings provided by the BME280.
func (bme *BME280SensorProvider) Readings() (*AtmoshphericReadings, error) {
	response, err := bme.driver.Read()
	if err != nil {
		return nil, err
	}

	return &AtmoshphericReadings{
		Temperature: response.Temperature,
		Humidity:    response.Humidity,
		Pressure:    response.Pressure,
	}, nil
}

// Disconnect closes the connection to the BME280.
func (bme *BME280SensorProvider) Disconnect() {
	if bme.driver == nil {
		log.WithField("component", "atmospheric provider").
			Debug("attempted to disconnect not connected provider")
		return
	}

	if driverErr := bme.driver.Close(); driverErr != nil {
		log.WithError(driverErr).
			WithField("component", "atmospheric provider").
			Error("driver failed to close")
	}
}

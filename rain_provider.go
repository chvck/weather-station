package main

// RainSensorProvider provides a way to setup and collect rain data readings.
type RainSensorProvider interface {
	SensorProvider
	Readings() (*RainReadings, error)
}

// RainReadings are the sensor readings about measurements such as rainfall.
type RainReadings struct {
	Rainfall float64 // mm
}

package main

// WindSensorProvider provides a way to setup and collect wind data readings.
type WindSensorProvider interface {
	SensorProvider
	Readings() (*WindReadings, error)
}

// WindReadings are the sensor readings about measurements such as wind speed.
type WindReadings struct {
	Speed     float64 // km/h
	Direction float64 // ??
	Gust      float64 // km/h
}

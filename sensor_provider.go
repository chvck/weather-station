package weatherstn

// SensorProvider is the base interface for sensor providers.
type SensorProvider interface {
	Connect() error
	Disconnect()
}

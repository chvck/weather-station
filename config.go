package weatherstn

import (
	"encoding/json"
	"io/ioutil"
)

// ProducerConfig is the set of configuration properties for setting up the Producer.
type ProducerConfig struct {
	PollIntervalSecs int                              `json:"intervalSecs"`
	Wind             SEN08942WindSensorProviderConfig `json:"wind"`
	Rain             SEN08942RainSensorProviderConfig `json:"rain"`
	Atmos            BME280SensorProviderConfig       `json:"atmos"`
}

// PublisherConfig is the set of configuration properties for setting up the Publisher.
type PublisherConfig struct {
	PushIntervalSecs int            `json:"intervalSecs"`
	EndpointConfig   EndpointConfig `json:"endpoints"`
}

// DatabaseConfig is the set of configuration properties for setting up the Database.
type DatabaseConfig struct {
	Path       string `json:"path"`
	Migrations string `json:"migrations"`
}

// AppConfig is the set of configuration properties for setting up the application.
type AppConfig struct {
	ProducerConfig  ProducerConfig  `json:"producer"`
	PublisherConfig PublisherConfig `json:"publisher"`
	DatabaseConfig  DatabaseConfig  `json:"database"`
	path            string
}

// NewAppConfig creates a new AppConfig.
func NewAppConfig(path string) *AppConfig {
	return &AppConfig{path: path}
}

// Parse reads and parses the config file.
func (ac *AppConfig) Parse() error {
	bytes, err := ioutil.ReadFile(ac.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &ac); err != nil {
		return err
	}

	return nil
}

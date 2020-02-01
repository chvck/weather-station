package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/chvck/weatherstn"

	log "github.com/sirupsen/logrus"
)

const (
	readInterval    = 30 * time.Second
	publishInterval = 60 * time.Second
)

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
	migrations := flag.Int("migrations", 0, "specifies to run n migrations (can be negative) and then exit")
	migrateAll := flag.Bool("migrateall", false, "specifies to run all migrations and then exit")
	configPath := flag.String("config", "config.json", "path to the config file")
	flag.Parse()

	config := weatherstn.NewAppConfig(*configPath)
	if err := config.Parse(); err != nil {
		log.WithError(err).Panic("failed to parse config")
	}

	log.Info(fmt.Sprintf("Running with config: %#v", config))

	if *migrations != 0 || *migrateAll {
		if *migrations != 0 && *migrateAll {
			panic("migrations and migrateall cannot be run together")
		}
		if err := doMigrate(config.DatabaseConfig.Path, config.DatabaseConfig.Migrations, *migrations,
			*migrateAll); err != nil {
			log.WithError(err).Panic("failed to perform migrations")
		}

		return
	}

	atmosProvider := weatherstn.NewBME280SensorProvider(weatherstn.BME280SensorProviderConfig{
		I2cAddr:      config.ProducerConfig.Atmos.I2cAddr,
		I2cBusDevice: config.ProducerConfig.Atmos.I2cBusDevice,
	})
	if err := atmosProvider.Connect(); err != nil {
		log.WithError(err).Panic("failed to connect to atmospherics provider")
	}

	windProvider := weatherstn.NewSEN08942WindSensorProvider(weatherstn.SEN08942WindSensorProviderConfig{
		AnemPinNumber:     config.ProducerConfig.Wind.AnemPinNumber,
		AnemInterval:      config.ProducerConfig.Wind.AnemInterval,
		VaneCSPinNumber:   config.ProducerConfig.Wind.VaneCSPinNumber,
		VaneDOutPinNumber: config.ProducerConfig.Wind.VaneDOutPinNumber,
		VaneDInPinNumber:  config.ProducerConfig.Wind.VaneDInPinNumber,
		VaneClkPinNumber:  config.ProducerConfig.Wind.VaneClkPinNumber,
		VaneChannel:       config.ProducerConfig.Wind.VaneChannel,
	})
	if err := windProvider.Connect(); err != nil {
		log.WithError(err).Panic("failed to connect to wind provider")
	}

	rainProvider := weatherstn.NewSEN08942RainSensorProvider(weatherstn.SEN08942RainSensorProviderConfig{
		PinNumber: config.ProducerConfig.Rain.PinNumber,
		Interval:  config.ProducerConfig.Rain.Interval,
	})
	if err := rainProvider.Connect(); err != nil {
		log.WithError(err).Panic("failed to connect to rain provider")
	}

	db, err := sqlx.Open("sqlite3", config.DatabaseConfig.Path)
	if err != nil {
		log.WithError(err).Panic("failed to connect to datastore")
	}

	datastore := weatherstn.NewSqliteDataStore(db)
	var wg sync.WaitGroup

	producer := weatherstn.NewSensorProducer(atmosProvider, windProvider, rainProvider, datastore)
	go func() {
		producer.Run(time.Duration(config.ProducerConfig.PollIntervalSecs) * time.Second)
	}()
	wg.Add(1)

	publisher := weatherstn.NewPublisher(datastore, config.PublisherConfig.EndpointConfig, &http.Client{})
	go func() {
		publisher.Run(time.Duration(config.PublisherConfig.PushIntervalSecs) * time.Second)
	}()
	wg.Add(1)

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt)
	defer signal.Stop(stopSig)

	<-stopSig

	go func() {
		producer.Stop()
		wg.Done()
	}()
	go func() {
		publisher.Stop()
		wg.Done()
	}()

	wg.Wait()
	atmosProvider.Disconnect()
	windProvider.Disconnect()
	rainProvider.Disconnect()
	fmt.Println("Graceful shutdown completed")
}

func doMigrate(dbPath, migrationsPath string, n int, all bool) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		"sqlite3://"+dbPath)
	if err != nil {
		return err
	}

	if all {
		err = m.Up()
		if err != nil {
			return err
		}
	}
	if n != 0 {
		err = m.Steps(n)
		if err != nil {
			return err
		}
	}

	return nil
}

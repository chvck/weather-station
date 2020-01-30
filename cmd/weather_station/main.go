package main

import (
	"flag"
	"fmt"
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
	migrations := flag.Int("migrations", 0, "specifies to run n migrations (can be negative) and then exit")
	migrateAll := flag.Bool("migrateall", false, "specifies to run all migrations and then exit")
	flag.Parse()

	if *migrations != 0 || *migrateAll {
		if *migrations != 0 && *migrateAll {
			panic("migrations and migrateall cannot be run together")
		}
		err := doMigrate("weather", "migrations", *migrations, *migrateAll)
		if err != nil {
			panic(err)
		}

		return
	}

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

	db, err := sqlx.Open("sqlite3", "weather")
	if err != nil {
		log.WithError(err).Panic("failed to connect to datastore")
	}

	datastore := weatherstn.NewSqliteDataStore(db)

	producer := weatherstn.NewSensorProducer(atmosProvider, windProvider, rainProvider, datastore)
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

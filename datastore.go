package weatherstn

import (
	log "github.com/sirupsen/logrus"

	"github.com/jmoiron/sqlx"
)

const (
	stmtInsertDataRow = "INSERT INTO observations (timestamp, wind_speed, wind_direction, wind_gust_speed," +
		"rainfall, temperature, humidity, pressure, interval_secs) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);"
	queryFetchUnpublishedDataRow = "SELECT timestamp, wind_speed, wind_direction, wind_gust_speed," +
		"rainfall, temperature, humidity, pressure, interval_secs FROM observations where published=false " +
		"ORDER BY timestamp ASC;"
	stmtUpdateDataRow = "UPDATE observations SET published=true WHERE timestamp BETWEEN ? AND ?;"
)

// WeatherDataRow is the structure for data passed to and from a DataStore.
type WeatherDataRow struct {
	Timestamp       int64                `json:"timestamp"`
	AtmosReadings   AtmoshphericReadings `json:"atmospherics"`
	WindReadings    WindReadings         `json:"wind"`
	RainReadings    RainReadings         `json:"rain"`
	IntervalSeconds int
}

type weatherDataRow struct {
	Timestamp       int64   `db:"timestamp"`
	Temperature     float64 `db:"temperature"`
	Pressure        float64 `db:"pressure"`
	Humidity        float64 `db:"humidity"`
	WindSpeed       float64 `db:"wind_speed"`
	WindDirection   float32 `db:"wind_direction"`
	WindGust        float64 `db:"wind_gust_speed"`
	Rainfall        float64 `db:"rainfall"`
	IntervalSeconds int     `db:"interval_secs"`
}

// DataStore is responsible for persisting and reading data from storage.
type DataStore interface {
	Write(WeatherDataRow) error
	ReadUnpublished() ([]WeatherDataRow, error)
	UpdatePublished(minTimestamp, maxTimestamp int64) error
}

// SqliteDataStore is an implementation of a DataStore that uses Sqlite statement syntax.
type SqliteDataStore struct {
	db *sqlx.DB
}

// NewSqliteDataStore creates a new SqliteDataStore.
func NewSqliteDataStore(db *sqlx.DB) *SqliteDataStore {
	return &SqliteDataStore{
		db: db,
	}
}

// Write persists the row to disk.
func (sds *SqliteDataStore) Write(row WeatherDataRow) error {
	_, err := sds.db.Exec(stmtInsertDataRow,
		row.Timestamp,
		row.WindReadings.Speed,
		row.WindReadings.Direction,
		row.WindReadings.Gust,
		row.RainReadings.Rainfall,
		row.AtmosReadings.Temperature,
		row.AtmosReadings.Humidity,
		row.AtmosReadings.Pressure,
		row.IntervalSeconds,
	)
	if err != nil {
		return err
	}

	return nil
}

// ReadUnpublished reads all of the unpublished rows from the database.
func (sds *SqliteDataStore) ReadUnpublished() ([]WeatherDataRow, error) {
	var rows []weatherDataRow
	err := sds.db.Select(&rows, queryFetchUnpublishedDataRow)
	if err != nil {
		return nil, err
	}

	var measurements []WeatherDataRow
	for _, row := range rows {
		measurements = append(measurements, WeatherDataRow{
			Timestamp:       row.Timestamp,
			IntervalSeconds: row.IntervalSeconds,
			WindReadings: WindReadings{
				Speed:     row.WindSpeed,
				Direction: row.WindDirection,
				Gust:      row.WindGust,
			},
			RainReadings: RainReadings{
				Rainfall: row.Rainfall,
			},
			AtmosReadings: AtmoshphericReadings{
				Temperature: row.Temperature,
				Humidity:    row.Humidity,
				Pressure:    row.Pressure,
			},
		})
	}

	return measurements, nil
}

// UpdatePublished sets all rows to published where timestamp is between the bounds.
func (sds *SqliteDataStore) UpdatePublished(minTimestamp, maxTimestamp int64) error {
	_, err := sds.db.Exec(stmtUpdateDataRow, minTimestamp, maxTimestamp)
	if err != nil {
		// This doesn't matter too much, we'll just end up resending data upstream which can deal with not duplicating
		// data.
		log.WithError(err).
			WithField("component", "SqliteDataStore").
			WithField("event", "ReadUnpublished").
			Error("failed to update published rows")
		return err
	}

	return nil
}

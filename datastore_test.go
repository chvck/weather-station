package weatherstn

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/DATA-DOG/go-sqlmock"
)

func newAtmosReadings(temp float64, humidity float64, pressure float64) AtmoshphericReadings {
	return AtmoshphericReadings{
		Temperature: temp,
		Humidity:    humidity,
		Pressure:    pressure,
	}
}

func newRainReadings(rainfall float64) RainReadings {
	return RainReadings{Rainfall: rainfall}
}

func newWindReadings(speed float64, direction float32, gust float64) WindReadings {
	return WindReadings{
		Speed:     speed,
		Direction: direction,
		Gust:      gust,
	}
}

func TestSqliteDataStore_Write(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening mock database: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	store := NewSqliteDataStore(sqlxDB)

	layout := "2006-01-02T15:04:05.000Z"
	timeStr := "2020-01-29T11:45:26.371Z"
	expectedTime, err := time.Parse(layout, timeStr)
	if err != nil {
		t.Fatalf("unexpected error parsing time string: %v", err)
	}

	row := WeatherDataRow{
		Timestamp:       expectedTime,
		AtmosReadings:   newAtmosReadings(20.2, 998.5, 57.4),
		WindReadings:    newWindReadings(4.225, 22.5, 5.1),
		RainReadings:    newRainReadings(0.084),
		IntervalSeconds: 30,
	}

	mock.ExpectExec("INSERT INTO observations").
		WithArgs(
			expectedTime.Unix(),
			row.WindReadings.Speed,
			row.WindReadings.Direction,
			row.WindReadings.Gust,
			row.RainReadings.Rainfall,
			row.AtmosReadings.Temperature,
			row.AtmosReadings.Humidity,
			row.AtmosReadings.Pressure,
			row.IntervalSeconds,
		).
		WillReturnResult(sqlmock.NewResult(5, 1))

	err = store.Write(row)
	if err != nil {
		t.Fatalf("failed to write to data store: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock received unexpected arguments: %v", err)
	}
}

func TestSqliteDataStore_UpdatePublished(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening mock database: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	store := NewSqliteDataStore(sqlxDB)

	var minTimestamp int64 = 1580339947
	var maxTimestamp int64 = 1580347147

	mock.ExpectExec("UPDATE observations SET published=true WHERE timestamp BETWEEN (.+)").
		WithArgs(minTimestamp, maxTimestamp).
		WillReturnResult(sqlmock.NewResult(5, 1))

	err = store.UpdatePublished(minTimestamp, maxTimestamp)
	if err != nil {
		t.Fatalf("failed to update published with data store: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock received unexpected arguments: %v", err)
	}
}

package weatherstn

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/DATA-DOG/go-sqlmock"
	tmock "github.com/stretchr/testify/mock"
)

type MockDataStore struct {
	tmock.Mock
}

func (mds *MockDataStore) Write(row WeatherDataRow) error {
	args := mds.Called(row)
	return args.Error(0)
}

func (mds *MockDataStore) UpdatePublished(minTimestamp, maxTimestamp int64) error {
	args := mds.Called(minTimestamp, maxTimestamp)
	return args.Error(0)
}

func loadJSONTestDataset(dataset string, valuePtr interface{}) error {
	bytes, err := ioutil.ReadFile("testdata/" + dataset + ".json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &valuePtr)
	if err != nil {
		return err
	}

	return nil
}

func (mds *MockDataStore) ReadUnpublished() ([]WeatherDataRow, error) {
	args := mds.Called()
	return args.Get(0).([]WeatherDataRow), args.Error(1)
}

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

	var row *WeatherDataRow
	err = loadJSONTestDataset("one_observation", &row)
	if err != nil {
		t.Fatalf("unexpected error loading json file: %v", err)
	}

	mock.ExpectExec("INSERT INTO observations").
		WithArgs(
			row.Timestamp,
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

	err = store.Write(*row)
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

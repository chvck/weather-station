CREATE TABLE observations (
    id INTEGER PRIMARY KEY,
    timestamp INTEGER NOT NULL,
    wind_speed REAL DEFAULT 0.0,
    wind_direction REAL DEFAULT 0.0,
    wind_gust_speed REAL DEFAULT 0.0,
    rainfall REAL DEFAULT 0.0,
    temperature REAL DEFAULT 0.0,
    humidity REAL DEFAULT 0.0,
    pressure REAL DEFAULT 0.0,
    interval_secs INTEGER NOT NULL,
    published BOOLEAN NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX idx_uniq_timestamp ON observations(timestamp);
CREATE INDEX idx_published ON observations(published);

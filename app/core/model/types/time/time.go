package time

import (
	"database/sql/driver"
	"time"
)

type Time struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *Time) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)

	return nil
}

// Value implements the driver Valuer interface.
func (nt Time) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}

	return nt.Time, nil
}

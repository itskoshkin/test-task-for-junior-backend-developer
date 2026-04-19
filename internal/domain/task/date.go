package task

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func NewDate(year int, month time.Month, day int) Date {
	return Date{Year: year, Month: month, Day: day}
}

func DateFromTime(t time.Time) Date {
	y, m, d := t.Date()
	return Date{Year: y, Month: m, Day: d}
}

func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("parse date: %w", err)
	}

	return DateFromTime(t), nil
}

func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d Date) Time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

func (d Date) IsZero() bool { return d == Date{} }

func (d Date) Before(other Date) bool { return d.Time().Before(other.Time()) }

func (d Date) After(other Date) bool { return d.Time().After(other.Time()) }

func (d Date) Equal(other Date) bool { return d == other }

func (d Date) AddDays(n int) Date {
	return DateFromTime(d.Time().AddDate(0, 0, n))
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.String() + `"`), nil
}

// noinspection GoMixedReceiverTypes
func (d *Date) UnmarshalJSON(data []byte) error {
	s := string(data)

	if s == "null" {
		*d = Date{}
		return nil
	}

	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return errors.New("date: expected string")
	}

	s = s[1 : len(s)-1]

	if s == "" {
		*d = Date{}
		return nil
	}

	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}

	*d = parsed
	return nil
}

func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}

	return d.Time(), nil
}

// noinspection GoMixedReceiverTypes
func (d *Date) Scan(src any) error {
	if src == nil {
		*d = Date{}
		return nil
	}

	switch v := src.(type) {
	case time.Time:
		*d = DateFromTime(v)
		return nil
	case string:
		parsed, err := ParseDate(v)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case []byte:
		parsed, err := ParseDate(string(v))
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	default:
		return fmt.Errorf("date: cannot scan %T", src)
	}
}

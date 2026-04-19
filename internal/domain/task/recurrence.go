package task

import (
	"fmt"
)

type RecurrenceType string

const (
	RecurrenceDaily          RecurrenceType = "daily"
	RecurrenceMonthly        RecurrenceType = "monthly"
	RecurrenceSpecificDates  RecurrenceType = "specific_dates"
	RecurrenceMonthdayParity RecurrenceType = "monthday_parity"
	RecurrenceLastDayOfMonth RecurrenceType = "last_day_of_month"
)

type RecurrenceRule interface {
	Type() RecurrenceType
	Validate() error
}

type DailyRule struct {
	EveryN int
}

func (DailyRule) Type() RecurrenceType { return RecurrenceDaily }

func (r DailyRule) Validate() error {
	if r.EveryN < 1 {
		return fmt.Errorf("%w: daily: every_n must be >= 1, got %d", ErrInvalidRule, r.EveryN)
	}

	return nil
}

type MonthlyRule struct {
	DaysOfMonth []int
}

func (MonthlyRule) Type() RecurrenceType { return RecurrenceMonthly }

func (r MonthlyRule) Validate() error {
	if len(r.DaysOfMonth) == 0 {
		return fmt.Errorf("%w: monthly: days_of_month must not be empty", ErrInvalidRule)
	}

	seen := make(map[int]struct{}, len(r.DaysOfMonth))
	for _, d := range r.DaysOfMonth {
		if d < 1 || d > 30 {
			return fmt.Errorf("%w: monthly: day %d is out of range [1..30]", ErrInvalidRule, d)
		}
		if _, dup := seen[d]; dup {
			return fmt.Errorf("%w: monthly: day %d is duplicated", ErrInvalidRule, d)
		}
		seen[d] = struct{}{}
	}

	return nil
}

type SpecificDatesRule struct {
	Dates []Date
}

func (SpecificDatesRule) Type() RecurrenceType { return RecurrenceSpecificDates }

func (r SpecificDatesRule) Validate() error {
	if len(r.Dates) == 0 {
		return fmt.Errorf("%w: specific_dates: dates must not be empty", ErrInvalidRule)
	}

	seen := make(map[Date]struct{}, len(r.Dates))
	for _, d := range r.Dates {
		if d.IsZero() {
			return fmt.Errorf("%w: specific_dates: contains zero date", ErrInvalidRule)
		}
		if _, dup := seen[d]; dup {
			return fmt.Errorf("%w: specific_dates: date %s is duplicated", ErrInvalidRule, d)
		}
		seen[d] = struct{}{}
	}

	return nil
}

type Parity string

const (
	ParityEven Parity = "even"
	ParityOdd  Parity = "odd"
)

type MonthdayParityRule struct {
	Parity Parity
}

func (MonthdayParityRule) Type() RecurrenceType { return RecurrenceMonthdayParity }

func (r MonthdayParityRule) Validate() error {
	switch r.Parity {
	case ParityEven, ParityOdd:
		return nil
	case "":
		return fmt.Errorf("%w: monthday_parity: parity is required", ErrInvalidRule)
	default:
		return fmt.Errorf("%w: monthday_parity: invalid parity %q", ErrInvalidRule, r.Parity)
	}
}

type LastDayOfMonthRule struct{}

func (LastDayOfMonthRule) Type() RecurrenceType { return RecurrenceLastDayOfMonth }

func (LastDayOfMonthRule) Validate() error { return nil }

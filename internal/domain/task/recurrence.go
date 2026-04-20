package task

import (
	"fmt"
	"slices"
	"time"
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
	// Occurrences returns rule dates within [from, to].
	// anchor defines the grid origin for DailyRule ("every N days since anchor"); other rules ignore it
	Occurrences(anchor, from, to Date) []Date
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

func (r DailyRule) Occurrences(anchor, from, to Date) []Date {
	if from.After(to) {
		return nil
	}

	// Skip anchor forward to the first grid point (anchor + k*N) that is >= from, so we don't iterate day-by-day from the template's start_date
	start := anchor
	if start.Before(from) {
		diff := daysInBetween(anchor, from)
		k := diff / r.EveryN
		if diff%r.EveryN != 0 {
			k++
		}
		start = anchor.AddDays(k * r.EveryN)
	}

	if start.After(to) {
		return nil
	}

	var out []Date
	for d := start; !d.After(to); d = d.AddDays(r.EveryN) {
		out = append(out, d)
	}

	return out
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

func (r MonthlyRule) Occurrences(_, from, to Date) []Date {
	if from.After(to) {
		return nil
	}

	days := append([]int(nil), r.DaysOfMonth...)
	slices.Sort(days)

	var out []Date
	y, m := from.Year, from.Month
	for {
		last := daysInMonth(y, m)
		for _, dom := range days {
			if dom > last {
				continue
			}
			d := NewDate(y, m, dom)
			if d.Before(from) {
				continue
			}
			if d.After(to) {
				return out
			}
			out = append(out, d)
		}

		m++
		if m > time.December {
			m = time.January
			y++
		}
	}
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

func (r SpecificDatesRule) Occurrences(_, from, to Date) []Date {
	if from.After(to) {
		return nil
	}

	dates := append([]Date(nil), r.Dates...)
	slices.SortFunc(dates, func(a, b Date) int {
		switch {
		case a.Before(b):
			return -1
		case a.After(b):
			return 1
		default:
			return 0
		}
	})

	var out []Date
	for _, d := range dates {
		if d.Before(from) {
			continue
		}
		if d.After(to) {
			break
		}
		out = append(out, d)
	}

	return out
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

func (r MonthdayParityRule) Occurrences(_, from, to Date) []Date {
	if from.After(to) {
		return nil
	}

	wantEven := r.Parity == ParityEven

	var out []Date
	for d := from; !d.After(to); d = d.AddDays(1) {
		if (d.Day%2 == 0) == wantEven {
			out = append(out, d)
		}
	}

	return out
}

type LastDayOfMonthRule struct{}

func (LastDayOfMonthRule) Type() RecurrenceType { return RecurrenceLastDayOfMonth }

func (LastDayOfMonthRule) Validate() error { return nil }

func (LastDayOfMonthRule) Occurrences(_, from, to Date) []Date {
	if from.After(to) {
		return nil
	}

	var out []Date
	y, m := from.Year, from.Month
	for {
		d := NewDate(y, m, daysInMonth(y, m))
		if !d.Before(from) {
			if d.After(to) {
				return out
			}
			out = append(out, d)
		}

		m++
		if m > time.December {
			m = time.January
			y++
		}
	}
}

// daysInMonth uses time.Date normalization: day=0 of month+1 rolls back to the last day of the target month, which automatically handles Feb/leap years
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func daysInBetween(from, to Date) int {
	return int(to.Time().Sub(from.Time()) / (24 * time.Hour))
}

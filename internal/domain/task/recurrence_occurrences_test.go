package task

import (
	"reflect"
	"testing"
	"time"
)

func d(y int, m time.Month, day int) Date { return NewDate(y, m, day) }

func TestDailyRule_Occurrences(t *testing.T) {
	tests := []struct {
		name   string
		rule   DailyRule
		anchor Date
		from   Date
		to     Date
		want   []Date
	}{
		{
			name:   "every 1 day, window at anchor",
			rule:   DailyRule{EveryN: 1},
			anchor: d(2026, time.April, 1),
			from:   d(2026, time.April, 1),
			to:     d(2026, time.April, 3),
			want:   []Date{d(2026, time.April, 1), d(2026, time.April, 2), d(2026, time.April, 3)},
		},
		{
			name:   "every 7 days, window ahead of anchor",
			rule:   DailyRule{EveryN: 7},
			anchor: d(2026, time.April, 1),
			from:   d(2026, time.April, 5),
			to:     d(2026, time.April, 20),
			want:   []Date{d(2026, time.April, 8), d(2026, time.April, 15)},
		},
		{
			name:   "every 3 days, anchor aligned with from",
			rule:   DailyRule{EveryN: 3},
			anchor: d(2026, time.April, 10),
			from:   d(2026, time.April, 10),
			to:     d(2026, time.April, 20),
			want:   []Date{d(2026, time.April, 10), d(2026, time.April, 13), d(2026, time.April, 16), d(2026, time.April, 19)},
		},
		{
			name:   "every 3 days, anchor not aligned with window start",
			rule:   DailyRule{EveryN: 3},
			anchor: d(2026, time.April, 10),
			from:   d(2026, time.April, 20),
			to:     d(2026, time.April, 25),
			want:   []Date{d(2026, time.April, 22), d(2026, time.April, 25)},
		},
		{
			name:   "anchor after window",
			rule:   DailyRule{EveryN: 1},
			anchor: d(2026, time.April, 10),
			from:   d(2026, time.April, 1),
			to:     d(2026, time.April, 3),
			want:   nil,
		},
		{
			name:   "anchor inside window but late",
			rule:   DailyRule{EveryN: 2},
			anchor: d(2026, time.April, 20),
			from:   d(2026, time.April, 10),
			to:     d(2026, time.April, 25),
			want:   []Date{d(2026, time.April, 20), d(2026, time.April, 22), d(2026, time.April, 24)},
		},
		{
			name:   "empty window (from > to)",
			rule:   DailyRule{EveryN: 1},
			anchor: d(2026, time.April, 1),
			from:   d(2026, time.April, 10),
			to:     d(2026, time.April, 9),
			want:   nil,
		},
		{
			name:   "window crosses month",
			rule:   DailyRule{EveryN: 5},
			anchor: d(2026, time.April, 28),
			from:   d(2026, time.April, 28),
			to:     d(2026, time.May, 10),
			want:   []Date{d(2026, time.April, 28), d(2026, time.May, 3), d(2026, time.May, 8)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.rule.Occurrences(tc.anchor, tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Occurrences() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMonthlyRule_Occurrences(t *testing.T) {
	tests := []struct {
		name string
		rule MonthlyRule
		from Date
		to   Date
		want []Date
	}{
		{
			name: "single day, single month",
			rule: MonthlyRule{DaysOfMonth: []int{15}},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 30),
			want: []Date{d(2026, time.April, 15)},
		},
		{
			name: "multiple days, unordered input — output sorted",
			rule: MonthlyRule{DaysOfMonth: []int{15, 1, 20}},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 30),
			want: []Date{d(2026, time.April, 1), d(2026, time.April, 15), d(2026, time.April, 20)},
		},
		{
			name: "day 30 in February (non-leap) — skipped",
			rule: MonthlyRule{DaysOfMonth: []int{30}},
			from: d(2025, time.February, 1),
			to:   d(2025, time.March, 31),
			want: []Date{d(2025, time.March, 30)},
		},
		{
			name: "day 29 in February (leap) — fires",
			rule: MonthlyRule{DaysOfMonth: []int{29}},
			from: d(2024, time.February, 1),
			to:   d(2024, time.February, 29),
			want: []Date{d(2024, time.February, 29)},
		},
		{
			name: "day 29 in February (non-leap) — skipped",
			rule: MonthlyRule{DaysOfMonth: []int{29}},
			from: d(2025, time.February, 1),
			to:   d(2025, time.February, 28),
			want: nil,
		},
		{
			name: "window crosses year boundary",
			rule: MonthlyRule{DaysOfMonth: []int{1}},
			from: d(2026, time.November, 15),
			to:   d(2027, time.February, 15),
			want: []Date{d(2026, time.December, 1), d(2027, time.January, 1), d(2027, time.February, 1)},
		},
		{
			name: "window excludes day in starting month",
			rule: MonthlyRule{DaysOfMonth: []int{1}},
			from: d(2026, time.April, 5),
			to:   d(2026, time.May, 10),
			want: []Date{d(2026, time.May, 1)},
		},
		{
			name: "empty window",
			rule: MonthlyRule{DaysOfMonth: []int{15}},
			from: d(2026, time.April, 20),
			to:   d(2026, time.April, 10),
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.rule.Occurrences(Date{}, tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Occurrences() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSpecificDatesRule_Occurrences(t *testing.T) {
	tests := []struct {
		name string
		rule SpecificDatesRule
		from Date
		to   Date
		want []Date
	}{
		{
			name: "dates inside window, sorted",
			rule: SpecificDatesRule{Dates: []Date{d(2026, time.April, 20), d(2026, time.April, 10)}},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 30),
			want: []Date{d(2026, time.April, 10), d(2026, time.April, 20)},
		},
		{
			name: "dates partially outside window",
			rule: SpecificDatesRule{Dates: []Date{d(2026, time.March, 30), d(2026, time.April, 10), d(2026, time.May, 5)}},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 30),
			want: []Date{d(2026, time.April, 10)},
		},
		{
			name: "no intersection",
			rule: SpecificDatesRule{Dates: []Date{d(2026, time.January, 1)}},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 30),
			want: nil,
		},
		{
			name: "empty window",
			rule: SpecificDatesRule{Dates: []Date{d(2026, time.April, 15)}},
			from: d(2026, time.April, 20),
			to:   d(2026, time.April, 10),
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.rule.Occurrences(Date{}, tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Occurrences() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMonthdayParityRule_Occurrences(t *testing.T) {
	tests := []struct {
		name string
		rule MonthdayParityRule
		from Date
		to   Date
		want []Date
	}{
		{
			name: "odd days in a short window",
			rule: MonthdayParityRule{Parity: ParityOdd},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 5),
			want: []Date{d(2026, time.April, 1), d(2026, time.April, 3), d(2026, time.April, 5)},
		},
		{
			name: "even days in a short window",
			rule: MonthdayParityRule{Parity: ParityEven},
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 5),
			want: []Date{d(2026, time.April, 2), d(2026, time.April, 4)},
		},
		{
			name: "odd parity includes the 31st",
			rule: MonthdayParityRule{Parity: ParityOdd},
			from: d(2026, time.January, 29),
			to:   d(2026, time.February, 1),
			want: []Date{d(2026, time.January, 29), d(2026, time.January, 31), d(2026, time.February, 1)},
		},
		{
			name: "even parity excludes the 31st",
			rule: MonthdayParityRule{Parity: ParityEven},
			from: d(2026, time.January, 30),
			to:   d(2026, time.February, 1),
			want: []Date{d(2026, time.January, 30)},
		},
		{
			name: "odd parity includes leap Feb 29",
			rule: MonthdayParityRule{Parity: ParityOdd},
			from: d(2024, time.February, 28),
			to:   d(2024, time.March, 1),
			want: []Date{d(2024, time.February, 29), d(2024, time.March, 1)},
		},
		{
			name: "empty window",
			rule: MonthdayParityRule{Parity: ParityEven},
			from: d(2026, time.April, 5),
			to:   d(2026, time.April, 4),
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.rule.Occurrences(Date{}, tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Occurrences() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLastDayOfMonthRule_Occurrences(t *testing.T) {
	tests := []struct {
		name string
		from Date
		to   Date
		want []Date
	}{
		{
			name: "February non-leap — 28",
			from: d(2025, time.February, 1),
			to:   d(2025, time.February, 28),
			want: []Date{d(2025, time.February, 28)},
		},
		{
			name: "February leap — 29",
			from: d(2024, time.February, 1),
			to:   d(2024, time.February, 29),
			want: []Date{d(2024, time.February, 29)},
		},
		{
			name: "April — 30",
			from: d(2026, time.April, 1),
			to:   d(2026, time.April, 30),
			want: []Date{d(2026, time.April, 30)},
		},
		{
			name: "January — 31",
			from: d(2026, time.January, 1),
			to:   d(2026, time.January, 31),
			want: []Date{d(2026, time.January, 31)},
		},
		{
			name: "multi-month non-leap Jan–April",
			from: d(2025, time.January, 1),
			to:   d(2025, time.April, 30),
			want: []Date{d(2025, time.January, 31), d(2025, time.February, 28), d(2025, time.March, 31), d(2025, time.April, 30)},
		},
		{
			name: "window excludes last day of the month",
			from: d(2026, time.April, 15),
			to:   d(2026, time.April, 20),
			want: nil,
		},
		{
			name: "window starts after last day of first month",
			from: d(2026, time.April, 30),
			to:   d(2026, time.May, 31),
			want: []Date{d(2026, time.April, 30), d(2026, time.May, 31)},
		},
		{
			name: "empty window",
			from: d(2026, time.May, 1),
			to:   d(2026, time.April, 30),
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := LastDayOfMonthRule{}.Occurrences(Date{}, tc.from, tc.to)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Occurrences() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDaysInMonth(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  int
	}{
		{2026, time.January, 31},
		{2026, time.February, 28},
		{2024, time.February, 29},
		{2000, time.February, 29}, // divisible by 400
		{1900, time.February, 28}, // divisible by 100 but not 400
		{2026, time.April, 30},
		{2026, time.December, 31},
	}
	for _, tc := range cases {
		if got := daysInMonth(tc.year, tc.month); got != tc.want {
			t.Errorf("daysInMonth(%d, %s) = %d, want %d", tc.year, tc.month, got, tc.want)
		}
	}
}

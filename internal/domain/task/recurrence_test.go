package task

import (
	"errors"
	"testing"
	"time"
)

func TestRuleTypes(t *testing.T) {
	cases := []struct {
		rule RecurrenceRule
		want RecurrenceType
	}{
		{DailyRule{EveryN: 1}, RecurrenceDaily},
		{MonthlyRule{DaysOfMonth: []int{1}}, RecurrenceMonthly},
		{SpecificDatesRule{Dates: []Date{NewDate(2026, time.April, 19)}}, RecurrenceSpecificDates},
		{MonthdayParityRule{Parity: ParityEven}, RecurrenceMonthdayParity},
		{LastDayOfMonthRule{}, RecurrenceLastDayOfMonth},
	}

	for _, tc := range cases {
		if got := tc.rule.Type(); got != tc.want {
			t.Errorf("%T.Type() = %q, want %q", tc.rule, got, tc.want)
		}
	}
}

func TestDailyRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    DailyRule
		wantErr bool
	}{
		{"every 1 day", DailyRule{EveryN: 1}, false},
		{"every 7 days", DailyRule{EveryN: 7}, false},
		{"zero", DailyRule{EveryN: 0}, true},
		{"negative", DailyRule{EveryN: -3}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, tc.rule, tc.wantErr)
		})
	}
}

func TestMonthlyRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    MonthlyRule
		wantErr bool
	}{
		{"single day", MonthlyRule{DaysOfMonth: []int{15}}, false},
		{"boundary 1", MonthlyRule{DaysOfMonth: []int{1}}, false},
		{"boundary 30", MonthlyRule{DaysOfMonth: []int{30}}, false},
		{"multiple days", MonthlyRule{DaysOfMonth: []int{1, 10, 20, 30}}, false},
		{"empty", MonthlyRule{DaysOfMonth: []int{}}, true},
		{"nil slice", MonthlyRule{DaysOfMonth: nil}, true},
		{"zero day", MonthlyRule{DaysOfMonth: []int{0}}, true},
		{"day 31 not allowed", MonthlyRule{DaysOfMonth: []int{31}}, true},
		{"day beyond 31", MonthlyRule{DaysOfMonth: []int{40}}, true},
		{"negative day", MonthlyRule{DaysOfMonth: []int{-1}}, true},
		{"duplicates", MonthlyRule{DaysOfMonth: []int{5, 10, 5}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, tc.rule, tc.wantErr)
		})
	}
}

func TestSpecificDatesRule_Validate(t *testing.T) {
	d1 := NewDate(2026, time.April, 19)
	d2 := NewDate(2026, time.May, 1)

	tests := []struct {
		name    string
		rule    SpecificDatesRule
		wantErr bool
	}{
		{"single date", SpecificDatesRule{Dates: []Date{d1}}, false},
		{"multiple dates", SpecificDatesRule{Dates: []Date{d1, d2}}, false},
		{"empty", SpecificDatesRule{Dates: []Date{}}, true},
		{"nil slice", SpecificDatesRule{Dates: nil}, true},
		{"contains zero date", SpecificDatesRule{Dates: []Date{d1, {}}}, true},
		{"duplicates", SpecificDatesRule{Dates: []Date{d1, d2, d1}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, tc.rule, tc.wantErr)
		})
	}
}

func TestMonthdayParityRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    MonthdayParityRule
		wantErr bool
	}{
		{"even", MonthdayParityRule{Parity: ParityEven}, false},
		{"odd", MonthdayParityRule{Parity: ParityOdd}, false},
		{"empty", MonthdayParityRule{Parity: ""}, true},
		{"unknown", MonthdayParityRule{Parity: "weekly"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, tc.rule, tc.wantErr)
		})
	}
}

func TestLastDayOfMonthRule_Validate(t *testing.T) {
	if err := (LastDayOfMonthRule{}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidationErrorsWrapInvalidRule(t *testing.T) {
	invalid := []RecurrenceRule{
		DailyRule{EveryN: 0},
		MonthlyRule{DaysOfMonth: []int{31}},
		SpecificDatesRule{Dates: nil},
		MonthdayParityRule{Parity: "bogus"},
	}
	
	for _, r := range invalid {
		err := r.Validate()
		if err == nil {
			t.Errorf("%T: expected error", r)
			continue
		}
		if !errors.Is(err, ErrInvalidRule) {
			t.Errorf("%T: error does not wrap ErrInvalidRule: %v", r, err)
		}
	}
}

func assertValidation(t *testing.T, r RecurrenceRule, wantErr bool) {
	t.Helper()
	err := r.Validate()
	if (err != nil) != wantErr {
		t.Errorf("%T.Validate() err = %v, wantErr = %v", r, err, wantErr)
	}
}

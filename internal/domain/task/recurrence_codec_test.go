package task

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestEncodeDecodeRule_Roundtrip(t *testing.T) {
	cases := []struct {
		name     string
		rule     RecurrenceRule
		wantType RecurrenceType
		wantJSON string
	}{
		{
			name:     "daily",
			rule:     DailyRule{EveryN: 3},
			wantType: RecurrenceDaily,
			wantJSON: `{"every_n":3}`,
		},
		{
			name:     "monthly",
			rule:     MonthlyRule{DaysOfMonth: []int{1, 15}},
			wantType: RecurrenceMonthly,
			wantJSON: `{"days_of_month":[1,15]}`,
		},
		{
			name:     "specific_dates",
			rule:     SpecificDatesRule{Dates: []Date{NewDate(2026, time.April, 1), NewDate(2026, time.April, 15)}},
			wantType: RecurrenceSpecificDates,
			wantJSON: `{"dates":["2026-04-01","2026-04-15"]}`,
		},
		{
			name:     "monthday_parity",
			rule:     MonthdayParityRule{Parity: ParityEven},
			wantType: RecurrenceMonthdayParity,
			wantJSON: `{"parity":"even"}`,
		},
		{
			name:     "last_day_of_month",
			rule:     LastDayOfMonthRule{},
			wantType: RecurrenceLastDayOfMonth,
			wantJSON: `{}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotJSON, err := EncodeRule(tc.rule)
			if err != nil {
				t.Fatalf("EncodeRule: %v", err)
			}
			if gotType != tc.wantType {
				t.Errorf("type = %q, want %q", gotType, tc.wantType)
			}
			if string(gotJSON) != tc.wantJSON {
				t.Errorf("json = %s, want %s", gotJSON, tc.wantJSON)
			}

			decoded, err := DecodeRule(gotType, gotJSON)
			if err != nil {
				t.Fatalf("DecodeRule: %v", err)
			}
			if !reflect.DeepEqual(decoded, tc.rule) {
				t.Errorf("roundtrip = %#v, want %#v", decoded, tc.rule)
			}
		})
	}
}

func TestDecodeRule_UnknownType(t *testing.T) {
	_, err := DecodeRule("nonsense", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown rule type")
	}
	if !errors.Is(err, ErrInvalidRule) {
		t.Errorf("error should wrap ErrInvalidRule, got %v", err)
	}
}

func TestDecodeRule_InvalidJSON(t *testing.T) {
	cases := []RecurrenceType{
		RecurrenceDaily,
		RecurrenceMonthly,
		RecurrenceSpecificDates,
		RecurrenceMonthdayParity,
	}

	for _, rt := range cases {
		t.Run(string(rt), func(t *testing.T) {
			if _, err := DecodeRule(rt, json.RawMessage(`not-json`)); err == nil {
				t.Fatalf("expected error for %q", rt)
			}
		})
	}
}

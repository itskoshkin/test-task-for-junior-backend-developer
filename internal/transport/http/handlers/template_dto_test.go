package handlers

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

func TestRuleDTO_RoundtripJSON(t *testing.T) {
	cases := []struct {
		name string
		rule taskdomain.RecurrenceRule
		raw  string
	}{
		{
			name: "daily",
			rule: taskdomain.DailyRule{EveryN: 3},
			raw:  `{"type":"daily","params":{"every_n":3}}`,
		},
		{
			name: "monthly",
			rule: taskdomain.MonthlyRule{DaysOfMonth: []int{1, 15}},
			raw:  `{"type":"monthly","params":{"days_of_month":[1,15]}}`,
		},
		{
			name: "specific_dates",
			rule: taskdomain.SpecificDatesRule{Dates: []taskdomain.Date{taskdomain.NewDate(2026, time.April, 1)}},
			raw:  `{"type":"specific_dates","params":{"dates":["2026-04-01"]}}`,
		},
		{
			name: "monthday_parity",
			rule: taskdomain.MonthdayParityRule{Parity: taskdomain.ParityOdd},
			raw:  `{"type":"monthday_parity","params":{"parity":"odd"}}`,
		},
		{
			name: "last_day_of_month",
			rule: taskdomain.LastDayOfMonthRule{},
			raw:  `{"type":"last_day_of_month","params":{}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dto, err := ruleDTOFromDomain(tc.rule)
			if err != nil {
				t.Fatalf("ruleDTOFromDomain: %v", err)
			}
			encoded, err := json.Marshal(dto)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(encoded) != tc.raw {
				t.Errorf("marshal = %s, want %s", encoded, tc.raw)
			}

			var decoded ruleDTO
			if err := json.Unmarshal([]byte(tc.raw), &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			rule, err := decoded.toDomain()
			if err != nil {
				t.Fatalf("toDomain: %v", err)
			}
			if !reflect.DeepEqual(rule, tc.rule) {
				t.Errorf("roundtrip = %#v, want %#v", rule, tc.rule)
			}
		})
	}
}

func TestRuleDTO_LastDayOfMonth_OmittedParams(t *testing.T) {
	// Client may send no params field at all for last_day_of_month.
	var dto ruleDTO
	if err := json.Unmarshal([]byte(`{"type":"last_day_of_month"}`), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rule, err := dto.toDomain()
	if err != nil {
		t.Fatalf("toDomain: %v", err)
	}
	if _, ok := rule.(taskdomain.LastDayOfMonthRule); !ok {
		t.Errorf("expected LastDayOfMonthRule, got %T", rule)
	}
}

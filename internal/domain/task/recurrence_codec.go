package task

import (
	"encoding/json"
	"fmt"
)

type dailyParams struct {
	EveryN int `json:"every_n"`
}

type monthlyParams struct {
	DaysOfMonth []int `json:"days_of_month"`
}

type specificDatesParams struct {
	Dates []Date `json:"dates"`
}

type monthdayParityParams struct {
	Parity Parity `json:"parity"`
}

func EncodeRule(rule RecurrenceRule) (RecurrenceType, json.RawMessage, error) {
	var payload any

	switch r := rule.(type) {
	case DailyRule:
		payload = dailyParams{EveryN: r.EveryN}
	case MonthlyRule:
		payload = monthlyParams{DaysOfMonth: r.DaysOfMonth}
	case SpecificDatesRule:
		payload = specificDatesParams{Dates: r.Dates}
	case MonthdayParityRule:
		payload = monthdayParityParams{Parity: r.Parity}
	case LastDayOfMonthRule:
		payload = struct{}{}
	default:
		return "", nil, fmt.Errorf("encode rule: unsupported type %T", rule)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("encode rule: %w", err)
	}

	return rule.Type(), raw, nil
}

func DecodeRule(rtype RecurrenceType, params json.RawMessage) (RecurrenceRule, error) {
	switch rtype {
	case RecurrenceDaily:
		var p dailyParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("decode daily: %w", err)
		}
		return DailyRule{EveryN: p.EveryN}, nil
	case RecurrenceMonthly:
		var p monthlyParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("decode monthly: %w", err)
		}
		return MonthlyRule{DaysOfMonth: p.DaysOfMonth}, nil
	case RecurrenceSpecificDates:
		var p specificDatesParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("decode specific_dates: %w", err)
		}
		return SpecificDatesRule{Dates: p.Dates}, nil
	case RecurrenceMonthdayParity:
		var p monthdayParityParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("decode monthday_parity: %w", err)
		}
		return MonthdayParityRule{Parity: p.Parity}, nil
	case RecurrenceLastDayOfMonth:
		return LastDayOfMonthRule{}, nil
	default:
		return nil, fmt.Errorf("%w: unknown rule type %q", ErrInvalidRule, rtype)
	}
}

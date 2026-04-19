package task

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDate_String(t *testing.T) {
	tests := []struct {
		name string
		in   Date
		want string
	}{
		{"typical", NewDate(2026, time.April, 19), "2026-04-19"},
		{"single-digit month and day", NewDate(2026, time.January, 5), "2026-01-05"},
		{"leap day", NewDate(2024, time.February, 29), "2024-02-29"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Date
		wantErr bool
	}{
		{"typical", "2026-04-19", NewDate(2026, time.April, 19), false},
		{"leap day", "2024-02-29", NewDate(2024, time.February, 29), false},
		{"wrong format", "19.04.2026", Date{}, true},
		{"non-existent date", "2025-02-29", Date{}, true},
		{"empty", "", Date{}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDate(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseDate(%q) err = %v, wantErr = %v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("ParseDate(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   Date
		want string
	}{
		{"non-zero", NewDate(2026, time.April, 19), `"2026-04-19"`},
		{"zero", Date{}, `null`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != tc.want {
				t.Errorf("Marshal = %s, want %q", got, tc.want)
			}
		})
	}
}

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Date
		wantErr bool
	}{
		{"typical", `"2026-04-19"`, NewDate(2026, time.April, 19), false},
		{"null", `null`, Date{}, false},
		{"empty string", `""`, Date{}, false},
		{"unquoted", `2026-04-19`, Date{}, true},
		{"non-existent date", `"2025-02-30"`, Date{}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got Date
			err := json.Unmarshal([]byte(tc.in), &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Unmarshal(%s) err = %v, wantErr = %v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("Unmarshal(%s) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestDate_JSONRoundtrip(t *testing.T) {
	in := NewDate(2026, time.April, 19)
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out Date
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Errorf("roundtrip: got %v, want %v", out, in)
	}
}

func TestDate_Compare(t *testing.T) {
	a := NewDate(2026, time.April, 19)
	b := NewDate(2026, time.April, 20)

	if !a.Before(b) {
		t.Error("a should be before b")
	}
	if !b.After(a) {
		t.Error("b should be after a")
	}
	if !a.Equal(a) {
		t.Error("a should equal itself")
	}
	if a.Before(a) || a.After(a) {
		t.Error("a should neither be before nor after itself")
	}
}

func TestDate_AddDays(t *testing.T) {
	tests := []struct {
		name string
		from Date
		n    int
		want Date
	}{
		{"next day", NewDate(2026, time.April, 19), 1, NewDate(2026, time.April, 20)},
		{"cross month", NewDate(2026, time.April, 30), 1, NewDate(2026, time.May, 1)},
		{"cross year", NewDate(2026, time.December, 31), 1, NewDate(2027, time.January, 1)},
		{"into leap day", NewDate(2024, time.February, 28), 1, NewDate(2024, time.February, 29)},
		{"skip leap day (non-leap year)", NewDate(2025, time.February, 28), 1, NewDate(2025, time.March, 1)},
		{"backwards", NewDate(2026, time.April, 1), -1, NewDate(2026, time.March, 31)},
		{"zero", NewDate(2026, time.April, 19), 0, NewDate(2026, time.April, 19)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.from.AddDays(tc.n); got != tc.want {
				t.Errorf("%v.AddDays(%d) = %v, want %v", tc.from, tc.n, got, tc.want)
			}
		})
	}
}

func TestDate_IsZero(t *testing.T) {
	if !(Date{}).IsZero() {
		t.Error("zero value should be zero")
	}
	if NewDate(2026, time.April, 19).IsZero() {
		t.Error("non-zero value should not be zero")
	}
}

func TestDate_Scan(t *testing.T) {
	tests := []struct {
		name    string
		src     any
		want    Date
		wantErr bool
	}{
		{"time.Time at midnight", time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC), NewDate(2026, time.April, 19), false},
		{"time.Time mid-day — day part extracted", time.Date(2026, time.April, 19, 15, 30, 0, 0, time.UTC), NewDate(2026, time.April, 19), false},
		{"string", "2026-04-19", NewDate(2026, time.April, 19), false},
		{"bytes", []byte("2026-04-19"), NewDate(2026, time.April, 19), false},
		{"nil", nil, Date{}, false},
		{"unsupported type", 42, Date{}, true},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Date
			err := d.Scan(tc.src)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Scan(%v) err = %v, wantErr = %v", tc.src, err, tc.wantErr)
			}
			if !tc.wantErr && d != tc.want {
				t.Errorf("Scan(%v) = %v, want %v", tc.src, d, tc.want)
			}
		})
	}
}

func TestDate_Value(t *testing.T) {
	d := NewDate(2026, time.April, 19)
	v, err := d.Value()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := v.(time.Time)
	if !ok {
		t.Fatalf("Value() = %T, want time.Time", v)
	}
	want := time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Value() = %v, want %v", got, want)
	}

	zero := Date{}
	v, err = zero.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != nil {
		t.Errorf("zero Value() = %v, want nil", v)
	}
}

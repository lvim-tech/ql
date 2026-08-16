package netstat

import (
	"math"
	"testing"
	"time"
)

// parsePeriod is a hand-rolled parser fed straight from the command line, and
// it had no tests at all. The relative forms are checked by how wide a window
// they open, which is the only thing the caller cares about.
func TestParsePeriodRelative(t *testing.T) {
	cases := []struct {
		arg  string
		want time.Duration
	}{
		{"30min", 30 * time.Minute},
		{"30minutes", 30 * time.Minute},
		{"30minute", 30 * time.Minute},
		{"1hour", time.Hour},
		{"2hours", 2 * time.Hour},
		{"1.5hours", 90 * time.Minute},
		{"3days", 72 * time.Hour},
		{"1day", 24 * time.Hour},
		{"2weeks", 14 * 24 * time.Hour},
	}

	for _, tc := range cases {
		start, end, err := parsePeriod(tc.arg)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.arg, err)
			continue
		}

		got := end.Sub(start)
		// A second of slack: parsePeriod reads the clock twice.
		if math.Abs(float64(got-tc.want)) > float64(2*time.Second) {
			t.Errorf("%s: window = %v, want %v", tc.arg, got, tc.want)
		}
	}
}

// "30minutes" used to come back as a zero-width window: the unit was trimmed
// as the literal "s" then "min", which left "30minute" and parsed as 0.
func TestParsePeriodMinuteSpellingsAgree(t *testing.T) {
	var windows []time.Duration

	for _, arg := range []string{"30min", "30minute", "30minutes"} {
		start, end, err := parsePeriod(arg)
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if end.Sub(start) < time.Minute {
			t.Errorf("%s: window collapsed to %v", arg, end.Sub(start))
		}
		windows = append(windows, end.Sub(start).Round(time.Second))
	}

	if windows[0] != windows[1] || windows[1] != windows[2] {
		t.Errorf("spellings disagree: %v", windows)
	}
}

func TestParsePeriodNamed(t *testing.T) {
	for _, arg := range []string{"today", "TODAY", "yesterday", "week", "month"} {
		start, end, err := parsePeriod(arg)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", arg, err)
			continue
		}
		if !start.Before(end) {
			t.Errorf("%s: start %v is not before end %v", arg, start, end)
		}
	}
}

func TestParsePeriodAbsolute(t *testing.T) {
	start, end, err := parsePeriod("2025-05-12")
	if err != nil {
		t.Fatalf("single date: %v", err)
	}
	if start.Format("2006-01-02") != "2025-05-12" {
		t.Errorf("start = %v, want 2025-05-12", start)
	}
	if end.Sub(start) < 23*time.Hour {
		t.Errorf("single date should cover the whole day, got %v", end.Sub(start))
	}

	start, end, err = parsePeriod("2025-05-12_14:00to2025-05-12_16:30")
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	if got, want := end.Sub(start), 150*time.Minute; got != want {
		t.Errorf("range window = %v, want %v", got, want)
	}
}

func TestParsePeriodRejectsGarbage(t *testing.T) {
	for _, arg := range []string{"", "nonsense", "2025-13-45", "hoursago", "-5min", "2025-05-12_25:00to2025-05-12_26:00"} {
		if _, _, err := parsePeriod(arg); err == nil {
			t.Errorf("%q was accepted", arg)
		}
	}
}

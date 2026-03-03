package openlistsync

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, v string) time.Time {
	t.Helper()
	tt, err := time.Parse("2006-01-02 15:04", v)
	if err != nil {
		t.Fatalf("parse time failed: %v", err)
	}
	return tt
}

func TestParseCrontabInvalid(t *testing.T) {
	if _, err := ParseCrontab("* * * *"); err == nil {
		t.Fatalf("expected invalid field count error")
	}
	if _, err := ParseCrontab("61 * * * *"); err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestCrontabNextEveryMinute(t *testing.T) {
	s, err := ParseCrontab("* * * * *")
	if err != nil {
		t.Fatalf("parse crontab error: %v", err)
	}
	got, err := s.Next(mustTime(t, "2026-03-03 10:00"))
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	want := mustTime(t, "2026-03-03 10:01")
	if !got.Equal(want) {
		t.Fatalf("next=%s, want=%s", got, want)
	}
}

func TestCrontabNextStep(t *testing.T) {
	s, err := ParseCrontab("*/15 * * * *")
	if err != nil {
		t.Fatalf("parse crontab error: %v", err)
	}
	got, err := s.Next(mustTime(t, "2026-03-03 10:01"))
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	want := mustTime(t, "2026-03-03 10:15")
	if !got.Equal(want) {
		t.Fatalf("next=%s, want=%s", got, want)
	}
}

func TestCrontabNextDaily(t *testing.T) {
	s, err := ParseCrontab("30 2 * * *")
	if err != nil {
		t.Fatalf("parse crontab error: %v", err)
	}
	got, err := s.Next(mustTime(t, "2026-03-03 02:31"))
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	want := mustTime(t, "2026-03-04 02:30")
	if !got.Equal(want) {
		t.Fatalf("next=%s, want=%s", got, want)
	}
}

func TestCrontabDowSevenIsSunday(t *testing.T) {
	s, err := ParseCrontab("0 9 * * 7")
	if err != nil {
		t.Fatalf("parse crontab error: %v", err)
	}
	got, err := s.Next(mustTime(t, "2026-03-03 09:00")) // Tuesday
	if err != nil {
		t.Fatalf("next error: %v", err)
	}
	want := mustTime(t, "2026-03-08 09:00") // Sunday
	if !got.Equal(want) {
		t.Fatalf("next=%s, want=%s", got, want)
	}
}

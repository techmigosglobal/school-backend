package handlers

import "testing"

func TestMonthYearRange(t *testing.T) {
	start, end, ok := monthYearRange("04", "2026")
	if !ok {
		t.Fatal("expected valid range")
	}
	if start.Format("2006-01-02") != "2026-04-01" {
		t.Fatalf("unexpected start date: %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-05-01" {
		t.Fatalf("unexpected end date: %s", end.Format("2006-01-02"))
	}
}

func TestMonthYearRangeInvalid(t *testing.T) {
	_, _, ok := monthYearRange("13", "2026")
	if ok {
		t.Fatal("expected invalid range for month 13")
	}
}

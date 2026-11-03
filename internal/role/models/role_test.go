package models

import (
	"testing"
	"time"
)

func TestWorkingHours_IsAllowedNow(t *testing.T) {
	// Disabled working hours allows everything
	disabledWH := &WorkingHours{Enabled: false}
	if !disabledWH.IsAllowedNow() {
		t.Errorf("Expected disabled working hours to allow access")
	}

	// Nil working hours allows everything
	var nilWH *WorkingHours
	if !nilWH.IsAllowedNow() {
		t.Errorf("Expected nil working hours to allow access")
	}

	// Always allowed if StartTime and EndTime are empty
	noTimesWH := &WorkingHours{
		Enabled: true,
	}
	if !noTimesWH.IsAllowedNow() {
		t.Errorf("Expected empty times to allow access")
	}

	// Check day restriction: only allow day that is definitely NOT today
	now := time.Now()
	otherDay := (int(now.Weekday()) + 1) % 7
	disallowedDayWH := &WorkingHours{
		Enabled: true,
		Days:    []int{otherDay},
	}
	if disallowedDayWH.IsAllowedNow() {
		t.Errorf("Expected working hours to reject access on non-allowed day")
	}

	// Allowed day: include today
	allowedDayWH := &WorkingHours{
		Enabled: true,
		Days:    []int{int(now.Weekday())},
	}
	if !allowedDayWH.IsAllowedNow() {
		t.Errorf("Expected working hours to allow access on allowed day")
	}
}

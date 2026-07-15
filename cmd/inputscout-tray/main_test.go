package main

import "testing"

func TestBatteryTier(t *testing.T) {
	for _, test := range []struct {
		percentage int
		want       int
	}{
		{5, 0},
		{10, 0},
		{11, 1},
		{20, 1},
		{21, 2},
		{100, 2},
	} {
		if got := batteryTier(test.percentage); got != test.want {
			t.Errorf("batteryTier(%d) = %d, want %d", test.percentage, got, test.want)
		}
	}
}

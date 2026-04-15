package cmd

import (
	"math"
	"testing"
)

func TestFormatInt(t *testing.T) {
	cases := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, "0"},
		{"small", 123, "123"},
		{"thousands", 1234, "1,234"},
		{"millions", 1234567, "1,234,567"},
		{"negative", -1234, "-1,234"},
		{"negative millions", -1234567, "-1,234,567"},
		{"MinInt64", math.MinInt64, "-9,223,372,036,854,775,808"},
		{"MaxInt64", math.MaxInt64, "9,223,372,036,854,775,807"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatInt(tc.n)
			if got != tc.want {
				t.Errorf("formatInt(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

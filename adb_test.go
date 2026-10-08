package main

import (
	"math"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{5368709120, "5.0 GB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.expected {
			t.Errorf("formatBytes(%d) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestPercentCalculation(t *testing.T) {
	total := int64(1000)
	used := int64(250)
	pct := float64(used) / float64(total) * 100

	if math.Abs(pct-25.0) > 0.001 {
		t.Errorf("expected 25.0, got %f", pct)
	}
}

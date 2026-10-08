package main

import (
	"math"
	"testing"
)

func TestParseMemoryKB(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"4096K", 4096},
		{"4096k", 4096},
		{"512M", 512 * 1024},
		{"512m", 512 * 1024},
		{"5.8G", 5.8 * 1024 * 1024},
		{"5.8g", 5.8 * 1024 * 1024},
		{"5,932,592K", 5932592},
		{"1000", 1000},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseMemoryKB(tt.input)
		if math.Abs(got-tt.expected) > 0.01 {
			t.Errorf("parseMemoryKB(%q) = %v; esperado %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsNumeric(t *testing.T) {
	if !isNumeric("1234") {
		t.Errorf("isNumeric(\"1234\") deveria ser true")
	}
	if isNumeric("12a4") {
		t.Errorf("isNumeric(\"12a4\") deveria ser false")
	}
	if isNumeric("") {
		t.Errorf("isNumeric(\"\") deveria ser false")
	}
}

func TestTruncateString(t *testing.T) {
	if got := truncateString("AndroidDevice", 7); got != "Android" {
		t.Errorf("truncateString(\"AndroidDevice\", 7) = %q; esperado \"Android\"", got)
	}
	if got := truncateString("ADB", 10); got != "ADB" {
		t.Errorf("truncateString(\"ADB\", 10) = %q; esperado \"ADB\"", got)
	}
}

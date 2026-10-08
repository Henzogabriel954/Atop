package main

import (
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input       string
		shouldBeNil bool
		expected    color.RGBA
	}{
		{"#10B981", false, color.RGBA{R: 16, G: 185, B: 129, A: 255}},
		{"#ffffff", false, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"#000000", false, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{"#invalid", true, color.RGBA{}},
		{"10B981", true, color.RGBA{}},
		{"#12", true, color.RGBA{}},
		{"", true, color.RGBA{}},
	}

	for _, tt := range tests {
		got := parseHexColor(tt.input)
		if tt.shouldBeNil {
			if got != nil {
				t.Errorf("parseHexColor(%q) deveria ser nil; recebido %v", tt.input, got)
			}
		} else {
			if got == nil {
				t.Errorf("parseHexColor(%q) retornou nil inesperado", tt.input)
				continue
			}
			rgba, ok := got.(color.RGBA)
			if !ok || rgba != tt.expected {
				t.Errorf("parseHexColor(%q) = %v; esperado %v", tt.input, got, tt.expected)
			}
		}
	}
}

package main

import (
	"image/color"
	"math"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
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
		got := formatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("formatBytes(%d) = %q; esperado %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestStorageReportSliceAngles(t *testing.T) {
	report := &StorageReport{
		TotalBytes: 100 * 1024 * 1024, // 100 MB
		UsedBytes:  60 * 1024 * 1024,  // 60 MB
		FreeBytes:  40 * 1024 * 1024,  // 40 MB
		Slices: []StorageSlice{
			{Name: "Fotos", Bytes: 30 * 1024 * 1024, Color: color.RGBA{R: 255, A: 255}},
			{Name: "Vídeos", Bytes: 30 * 1024 * 1024, Color: color.RGBA{G: 255, A: 255}},
			{Name: "Livre", Bytes: 40 * 1024 * 1024, Color: color.RGBA{B: 255, A: 255}},
		},
	}

	totalAngle := 0.0
	for _, s := range report.Slices {
		angle := (float64(s.Bytes) / float64(report.TotalBytes)) * 2.0 * math.Pi
		totalAngle += angle
	}

	if math.Abs(totalAngle-2.0*math.Pi) > 0.001 {
		t.Errorf("A soma dos ângulos da pizza = %v rad; esperado %v rad (2*pi)", totalAngle, 2.0*math.Pi)
	}

	// Testa que buildPieChartRaster instancia sem panics
	raster := buildPieChartRaster(report, 100)
	if raster == nil {
		t.Errorf("buildPieChartRaster retornou nil")
	}
}

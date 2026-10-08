package main

import (
	"image/color"
	"testing"
)

func TestStorageCategoryParsing(t *testing.T) {
	cats := []StorageCategory{
		{Name: "Fotos & Videos", Path: "/sdcard/DCIM", Color: color.NRGBA{R: 66, G: 133, B: 244, A: 255}, Bytes: 1048576},
		{Name: "Musicas & Audio", Path: "/sdcard/Music", Color: color.NRGBA{R: 52, G: 168, B: 83, A: 255}, Bytes: 2097152},
		{Name: "Downloads", Path: "/sdcard/Download", Color: color.NRGBA{R: 251, G: 188, B: 5, A: 255}, Bytes: 3145728},
	}

	var total int64
	for _, c := range cats {
		total += c.Bytes
	}

	expected := int64(1048576 + 2097152 + 3145728)
	if total != expected {
		t.Errorf("Total bytes mismatch: got %d, want %d", total, expected)
	}

	formatted := formatBytes(total)
	if formatted != "6.0 MB" {
		t.Errorf("formatBytes(%d) = %s, want 6.0 MB", total, formatted)
	}
}

func TestPieChartAngles(t *testing.T) {
	cats := []StorageCategory{
		{Name: "Cat A", Bytes: 50, Color: color.NRGBA{255, 0, 0, 255}},
		{Name: "Cat B", Bytes: 50, Color: color.NRGBA{0, 255, 0, 255}},
	}
	var total int64 = 100

	angles := make([]float64, len(cats))
	for i, c := range cats {
		angles[i] = (float64(c.Bytes) / float64(total)) * 360.0
	}

	if angles[0] != 180.0 || angles[1] != 180.0 {
		t.Errorf("Expected 180 and 180 degrees, got %f and %f", angles[0], angles[1])
	}
}

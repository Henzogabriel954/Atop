package main

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2/theme"
)

func TestCleanThemeColors(t *testing.T) {
	th := &CleanTheme{}
	bg := th.Color(theme.ColorNameBackground, theme.VariantDark)
	r, g, b, a := bg.RGBA()
	if a == 0 {
		t.Errorf("CleanTheme background alpha is 0")
	}

	// Verifica se a cor de fundo é escura e minimalista (RGB baixo)
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)

	if r8 > 50 || g8 > 50 || b8 > 50 {
		t.Errorf("CleanTheme background is too bright for dark minimal theme: (%d, %d, %d)", r8, g8, b8)
	}
}

func TestCleanThemeSizes(t *testing.T) {
	th := &CleanTheme{}
	padding := th.Size(theme.SizeNamePadding)
	if padding <= 0 {
		t.Errorf("Invalid padding size: %f", padding)
	}

	text := th.Size(theme.SizeNameText)
	if text <= 0 {
		t.Errorf("Invalid text size: %f", text)
	}
}

func TestGetAccentColor(t *testing.T) {
	state := NewAppState()
	state.ColorPalette = "blue"
	c := state.GetAccentColor()
	if c == nil {
		t.Errorf("Accent color should not be nil")
	}

	state.ColorPalette = "green"
	cGreen := state.GetAccentColor()
	if _, ok := cGreen.(color.NRGBA); !ok {
		t.Errorf("Accent color should be NRGBA")
	}
}

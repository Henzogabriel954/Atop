package main

import (
	"bufio"
	"image/color"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Cores padrão refinadas e minimalistas
var (
	// Cor de destaque moderna e vibrante (Emerald Green #10B981)
	fallbackAccentColor = color.RGBA{R: 16, G: 185, B: 129, A: 255}
	// Fundo escuro elegante (#121214)
	darkBackgroundColor = color.RGBA{R: 18, G: 18, B: 20, A: 255}
	// Fundo de cartões e painéis (#1c1c1f)
	cardBackgroundColor = color.RGBA{R: 28, G: 28, B: 31, A: 255}
)

type DynamicTheme struct {
	accentColor color.Color
}

func NewDynamicTheme() fyne.Theme {
	col := getWalColor()
	return &DynamicTheme{accentColor: col}
}

func (t *DynamicTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return darkBackgroundColor
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return cardBackgroundColor
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameSelection:
		return t.accentColor
	case theme.ColorNameButton:
		return color.RGBA{R: 35, G: 35, B: 40, A: 255}
	case theme.ColorNameDisabled:
		return color.RGBA{R: 55, G: 55, B: 60, A: 255}
	case theme.ColorNameSeparator:
		return color.RGBA{R: 40, G: 40, B: 45, A: 255}
	default:
		return theme.DarkTheme().Color(name, variant)
	}
}

func (t *DynamicTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (t *DynamicTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (t *DynamicTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}

func getWalColor() color.Color {
	// Apenas verifica Pywal se estiver em Linux
	if runtime.GOOS != "linux" {
		return fallbackAccentColor
	}

	usr, err := user.Current()
	if err != nil {
		return fallbackAccentColor
	}

	path := filepath.Join(usr.HomeDir, ".cache", "wal", "colors")
	file, err := os.Open(path)
	if err != nil {
		return fallbackAccentColor
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if len(lines) > 1 && len(lines[1]) > 0 {
		parsed := parseHexColor(lines[1])
		if parsed != nil {
			return parsed
		}
	}
	return fallbackAccentColor
}

func parseHexColor(s string) color.Color {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return nil
	}
	r, errR := strconv.ParseUint(s[1:3], 16, 8)
	g, errG := strconv.ParseUint(s[3:5], 16, 8)
	b, errB := strconv.ParseUint(s[5:7], 16, 8)
	if errR != nil || errG != nil || errB != nil {
		return nil
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

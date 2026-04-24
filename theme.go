package main

import (
	"bufio"
	"image/color"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Cor de fallback
var fallbackColor = color.RGBA{R: 35, G: 35, B: 35, A: 255}

type DynamicTheme struct {
	accentColor color.Color
}

func NewDynamicTheme() fyne.Theme {
	col := getWalColor()
	return &DynamicTheme{accentColor: col}
}

func (t *DynamicTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Cor de fundo SÓLIDA
	if name == theme.ColorNameBackground {
		return color.RGBA{R: 20, G: 20, B: 20, A: 255}
	}

	// Cores do Pywal
	if name == theme.ColorNamePrimary || 
	   name == theme.ColorNameFocus || 
	   name == theme.ColorNameSelection || 
	   name == theme.ColorNameButton { 
		return t.accentColor
	}
	
	return theme.DarkTheme().Color(name, variant)
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
	usr, err := user.Current()
	if err != nil { return fallbackColor }
	
	path := filepath.Join(usr.HomeDir, ".cache", "wal", "colors")
	file, err := os.Open(path)
	if err != nil { return fallbackColor }
	defer file.Close()
	
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() { lines = append(lines, scanner.Text()) }
	
	if len(lines) > 1 {
		return parseHexColor(lines[1])
	}
	return fallbackColor
}

func parseHexColor(s string) color.Color {
	if len(s) != 7 || s[0] != '#' { return fallbackColor }
	r, _ := strconv.ParseUint(s[1:3], 16, 8)
	g, _ := strconv.ParseUint(s[3:5], 16, 8)
	b, _ := strconv.ParseUint(s[5:7], 16, 8)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

package main

import (
	"bufio"
	"image/color"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type CleanTheme struct{}

var _ fyne.Theme = (*CleanTheme)(nil)

func (m *CleanTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 18, G: 20, B: 24, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 30, G: 34, B: 42, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 24, G: 26, B: 30, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 100, G: 106, B: 118, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 240, G: 242, B: 245, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 45, G: 52, B: 64, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 24, G: 28, B: 35, A: 255}
	case theme.ColorNameMenuBackground:
		return color.NRGBA{R: 22, G: 25, B: 31, A: 255}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 22, G: 25, B: 31, A: 235}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 74, G: 144, B: 226, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 50, G: 56, B: 68, A: 180}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 120}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 130, G: 136, B: 148, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 55, G: 65, B: 81, A: 255}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 74, G: 144, B: 226, A: 75}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (m *CleanTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *CleanTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *CleanTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInlineIcon:
		return 16
	case theme.SizeNameScrollBar:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameText:
		return 13
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameSubHeadingText:
		return 15
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameInputBorder:
		return 1
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func LoadConfig() map[string]string {
	cfg := make(map[string]string)
	f, err := os.Open("config.txt")
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, extraSplit("=")...)
		if len(parts) == 2 {
			cfg[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return cfg
}

func extraSplit(sep string) []string {
	return []string{sep, "2"}
}

func SaveConfig(cfg map[string]string) error {
	f, err := os.Create("config.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	for k, v := range cfg {
		if _, err := f.WriteString(k + "=" + v + "\n"); err != nil {
			return err
		}
	}
	return nil
}

package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowSettingsDialog(parent fyne.Window, state *AppState) {
	d := fyne.CurrentApp().NewWindow("Configurações - Atop")
	d.Resize(fyne.NewSize(360, 320))

	cfg := LoadConfig()

	// Entrada de IP
	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("Ex: 192.168.1.100")
	if val, ok := cfg["ip"]; ok {
		ipEntry.SetText(val)
	}

	// Entrada de Porta
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("5555")
	if val, ok := cfg["port"]; ok {
		portEntry.SetText(val)
	} else {
		portEntry.SetText("5555")
	}

	// Seleção de Paleta de Cor de Destaque
	colorSelect := widget.NewSelect([]string{"Azul", "Verde", "Roxo", "Laranja"}, func(s string) {
		switch s {
		case "Azul":
			state.ColorPalette = "blue"
		case "Verde":
			state.ColorPalette = "green"
		case "Roxo":
			state.ColorPalette = "purple"
		case "Laranja":
			state.ColorPalette = "orange"
		}
	})
	switch state.ColorPalette {
	case "green":
		colorSelect.SetSelected("Verde")
	case "purple":
		colorSelect.SetSelected("Roxo")
	case "orange":
		colorSelect.SetSelected("Laranja")
	default:
		colorSelect.SetSelected("Azul")
	}

	// Histórico de IPs salvos
	historyLabel := widget.NewLabelWithStyle("[•] Conexões Recentes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	historyList := container.NewVBox()

	updateHistory := func() {
		historyList.Objects = nil
		hist := GetSavedConnections()
		for _, h := range hist {
			target := h
			btn := widget.NewButton("▸ "+target, func() {
				parts := strings.Split(target, ":")
				if len(parts) == 2 {
					ipEntry.SetText(parts[0])
					portEntry.SetText(parts[1])
				}
			})
			btn.Importance = widget.LowImportance
			historyList.Add(btn)
		}
		historyList.Refresh()
	}
	updateHistory()

	saveBtn := widget.NewButtonWithIcon("Salvar", nil, func() {
		cfg["ip"] = ipEntry.Text
		cfg["port"] = portEntry.Text
		cfg["palette"] = state.ColorPalette
		if err := SaveConfig(cfg); err == nil {
			state.AddLog(fmt.Sprintf("[+] Configurações salvas: %s:%s", ipEntry.Text, portEntry.Text))
		}
		d.Close()
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Cancelar", func() {
		d.Close()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("[•] Preferências de Conexão", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Endereço IP do Dispositivo:"),
		ipEntry,
		widget.NewLabel("Porta ADB:"),
		portEntry,
		widget.NewLabel("Cor de Destaque da Interface:"),
		colorSelect,
		widget.NewSeparator(),
		historyLabel,
		historyList,
		widget.NewSeparator(),
		container.NewHBox(cancelBtn, saveBtn),
	)

	d.SetContent(container.NewPadded(form))
	d.Show()
}

func GetSavedConnections() []string {
	cfg := LoadConfig()
	raw, ok := cfg["history"]
	if !ok || raw == "" {
		return []string{}
	}
	return strings.Split(raw, ",")
}

func AddSavedConnection(target string) {
	cfg := LoadConfig()
	hist := GetSavedConnections()
	for _, h := range hist {
		if h == target {
			return
		}
	}
	hist = append(hist, target)
	if len(hist) > 5 {
		hist = hist[len(hist)-5:]
	}
	cfg["history"] = strings.Join(hist, ",")
	_ = SaveConfig(cfg)
}

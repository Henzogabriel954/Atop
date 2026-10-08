package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func showSettingsScreen() {
	stopDeviceListPolling()
	clearDeviceListListener()
	stopMonitoring()
	clearMonitorListeners()

	state.Window.Canvas().SetOnTypedRune(nil)

	// --- Wi-Fi Auto-Connect (mDNS) ---
	wifiCheck := widget.NewCheck("Conexão Wi-Fi Automática (mDNS)", func(on bool) {
		state.App.Preferences().SetBool(PrefWifiAutoConnect, on)
		if on {
			startMdnsLoop()
		} else {
			stopMdnsLoop()
		}
	})
	wifiCheck.SetChecked(state.App.Preferences().BoolWithFallback(PrefWifiAutoConnect, true))

	wifiDesc := widget.NewLabel("Busca dispositivos em ciclos contínuos de 5 minutos, conectando automaticamente mesmo se a depuração for ativada após abrir o app.")
	wifiDesc.Wrapping = fyne.TextWrapWord
	wifiDesc.TextStyle = fyne.TextStyle{Italic: true}

	// --- Fixar Porta 5555 ---
	fixPortCheck := widget.NewCheck("Fixar Porta 5555 ao conectar via Wi-Fi", func(on bool) {
		state.App.Preferences().SetBool(PrefFixPort, on)
	})
	fixPortCheck.SetChecked(state.App.Preferences().BoolWithFallback(PrefFixPort, false))

	fixPortDesc := widget.NewLabel("Após conectar via mDNS, executa 'adb tcpip 5555' para abrir porta estática no dispositivo.")
	fixPortDesc.Wrapping = fyne.TextWrapWord
	fixPortDesc.TextStyle = fyne.TextStyle{Italic: true}

	// --- Conexão Manual ---
	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("Ex: 192.168.1.100:5555")

	connectBtn := widget.NewButtonWithIcon("Conectar Dispositivo", theme.NavigateNextIcon(), func() {
		addr := strings.TrimSpace(ipEntry.Text)
		if addr == "" {
			return
		}

		if !strings.Contains(addr, ":") {
			addr = addr + ":5555"
		}

		go func() {
			err := connectManual(addr)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Falha ao conectar: %v", err), state.Window)
			} else {
				dialog.ShowInformation("Sucesso", "Conectado a "+addr, state.Window)
				val, _ := state.DeviceListRefresher.Get()
				state.DeviceListRefresher.Set(!val)
			}
		}()
	})

	// --- Intervalo de Atualização (Reativo) ---
	currentInterval := state.App.Preferences().IntWithFallback(PrefRefreshInterval, 1000)
	var currentIntervalStr string
	switch currentInterval {
	case 500:
		currentIntervalStr = "500ms (Muito rápido)"
	case 2000:
		currentIntervalStr = "2s (Econômico)"
	case 5000:
		currentIntervalStr = "5s (Ultra leve)"
	default:
		currentIntervalStr = "1s (Padrão)"
	}

	intervalSelect := widget.NewSelect([]string{"500ms (Muito rápido)", "1s (Padrão)", "2s (Econômico)", "5s (Ultra leve)"}, func(val string) {
		var ms int
		switch val {
		case "500ms (Muito rápido)":
			ms = 500
		case "2s (Econômico)":
			ms = 2000
		case "5s (Ultra leve)":
			ms = 5000
		default:
			ms = 1000
		}
		state.App.Preferences().SetInt(PrefRefreshInterval, ms)
		newDur := time.Duration(ms) * time.Millisecond
		state.RefreshInterval = newDur

		select {
		case state.RefreshIntervalUpdate <- newDur:
		default:
		}
	})
	intervalSelect.SetSelected(currentIntervalStr)

	// --- Botão Voltar ---
	backBtn := widget.NewButtonWithIcon("Voltar", theme.NavigateBackIcon(), func() {
		showDeviceListScreen()
	})

	// --- Layout ---
	header := container.NewBorder(nil, nil, backBtn, nil,
		container.NewCenter(widget.NewLabelWithStyle("Configurações", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
	)

	content := container.NewPadded(
		container.NewVBox(
			header,
			widget.NewSeparator(),

			widget.NewCard("Rede e Conexão Automática", "", container.NewVBox(
				wifiCheck,
				wifiDesc,
				fixPortCheck,
				fixPortDesc,
			)),

			widget.NewCard("Conexão Manual por IP", "", container.NewVBox(
				ipEntry,
				connectBtn,
			)),

			widget.NewCard("Taxa de Amostragem do Monitor", "", container.NewVBox(
				widget.NewLabel("Intervalo de leitura de CPU e memória:"),
				intervalSelect,
			)),

			layout.NewSpacer(),
		),
	)

	state.Window.SetContent(content)
}

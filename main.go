package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.NewWithID("com.henzo.atop")
	a.Settings().SetTheme(&CleanTheme{})

	w := a.NewWindow("Atop - Android Device Monitor")
	w.Resize(fyne.NewSize(520, 680))
	w.SetFixedSize(false)

	appIcon := GetAppIcon()
	if appIcon != nil {
		w.SetIcon(appIcon)
	}

	state := NewAppState()

	// Cria e configura UI
	ui := BuildUI(w, state)
	w.SetContent(ui)

	// Registra atalhos de teclado minimalistas
	RegisterShortcuts(w, state)

	// Inicia rotina de atualizacao de telemetria
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if state.IsMonitoring() {
					state.FetchMetrics()
				}
			case <-state.StopChan:
				ticker.Stop()
				return
			}
		}
	}()

	// Loop contínuo de varredura mDNS a cada 5 minutos
	go StartMDNSPeriodicScanner(state)

	w.SetOnClosed(func() {
		state.Cleanup()
	})

	w.ShowAndRun()
}

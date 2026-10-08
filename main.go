package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
)

func main() {
	checkDependencies()

	a := app.NewWithID("com.henzo.atop")
	w := a.NewWindow("Atop - Android Device Monitor")

	if celularIconResource != nil {
		w.SetIcon(celularIconResource)
	}

	a.Settings().SetTheme(NewDynamicTheme())

	// Intervalo de refresh das preferências (padrão 1000ms)
	refreshMs := a.Preferences().IntWithFallback(PrefRefreshInterval, 1000)

	state = &AppState{
		App:                   a,
		Window:                w,
		CpuPercent:            binding.NewFloat(),
		RamTotal:              binding.NewFloat(),
		RamUsed:               binding.NewFloat(),
		SwapTotal:             binding.NewFloat(),
		SwapUsed:              binding.NewFloat(),
		BatteryPercent:        binding.NewFloat(),
		Temperature:           binding.NewFloat(),
		TableRefresher:        binding.NewBool(),
		DeviceListRefresher:   binding.NewBool(),
		MdnsMap:               make(map[string]string),
		RefreshInterval:       time.Duration(refreshMs) * time.Millisecond,
		RefreshIntervalUpdate: make(chan time.Duration, 4),
	}

	w.Resize(fyne.NewSize(520, 720))
	w.SetMaster()

	showDeviceListScreen()

	// Auto-connect Wi-Fi via mDNS (padrão: LIGADO)
	if a.Preferences().BoolWithFallback(PrefWifiAutoConnect, true) {
		startMdnsLoop()
	}

	w.ShowAndRun()
}

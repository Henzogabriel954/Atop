package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
)

func main() {
	checkDependencies()

	a := app.New()
	w := a.NewWindow("Go ADB Manager")

	// w.SetDecorated(false)
	// w.SetTransparent(true)

	a.Settings().SetTheme(NewDynamicTheme())

	state = &AppState{
		App:                 a,
		Window:              w,
		CpuPercent:          binding.NewFloat(),
		RamTotal:            binding.NewFloat(),
		RamUsed:             binding.NewFloat(),
		SwapTotal:           binding.NewFloat(),
		SwapUsed:            binding.NewFloat(),
		TableRefresher:      binding.NewBool(),
		DeviceListRefresher: binding.NewBool(),
		MdnsMap:             make(map[string]string),
	}

	w.Resize(fyne.NewSize(500, 700))
	w.SetMaster()

	showDeviceListScreen()

	go mdnsAutoConnectLoop()

	w.ShowAndRun()
}

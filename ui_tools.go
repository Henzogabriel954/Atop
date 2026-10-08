package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowToolsDialog(parent fyne.Window, state *AppState) {
	d := fyne.CurrentApp().NewWindow("Ferramentas Rápidas - Atop")
	d.Resize(fyne.NewSize(380, 420))

	serial := state.DeviceSerial
	if serial == "" {
		d.SetContent(container.NewPadded(
			container.NewVBox(
				widget.NewLabelWithStyle("[!] Nenhum dispositivo conectado", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Conecte um dispositivo Android via USB ou Wi-Fi antes de usar as ferramentas."),
				widget.NewButton("Fechar", func() { d.Close() }),
			),
		))
		d.Show()
		return
	}

	statusLabel := widget.NewLabel("")

	rebootBtn := widget.NewButton("Reiniciar Dispositivo", func() {
		statusLabel.SetText("[•] Reiniciando aparelho...")
		go func() {
			err := RebootDevice(serial)
			if err != nil {
				statusLabel.SetText("[!] Erro: " + err.Error())
			} else {
				statusLabel.SetText("[+] Comando de reinício enviado.")
			}
		}()
	})

	rebootRecBtn := widget.NewButton("Reiniciar no Recovery", func() {
		statusLabel.SetText("[•] Reiniciando no Recovery...")
		go func() {
			err := RebootRecovery(serial)
			if err != nil {
				statusLabel.SetText("[!] Erro: " + err.Error())
			} else {
				statusLabel.SetText("[+] Reiniciando no Recovery.")
			}
		}()
	})

	rebootBlBtn := widget.NewButton("Reiniciar no Bootloader / Fastboot", func() {
		statusLabel.SetText("[•] Reiniciando no Bootloader...")
		go func() {
			err := RebootBootloader(serial)
			if err != nil {
				statusLabel.SetText("[!] Erro: " + err.Error())
			} else {
				statusLabel.SetText("[+] Reiniciando no Bootloader.")
			}
		}()
	})

	airplaneBtn := widget.NewButton("Alternar Modo Avião (Reset de Rede)", func() {
		statusLabel.SetText("[•] Alternando modo avião...")
		go func() {
			err := ToggleAirplaneMode(serial)
			if err != nil {
				statusLabel.SetText("[!] Erro: " + err.Error())
			} else {
				statusLabel.SetText("[+] Modo avião alternado com sucesso.")
			}
		}()
	})

	clearLogcatBtn := widget.NewButton("Limpar Buffer do Logcat", func() {
		statusLabel.SetText("[•] Limpando logcat...")
		go func() {
			err := ClearLogcat(serial)
			if err != nil {
				statusLabel.SetText("[!] Erro: " + err.Error())
			} else {
				statusLabel.SetText("[+] Logcat limpo.")
			}
		}()
	})

	screenshotBtn := widget.NewButton("Capturar Screenshot do Aparelho", func() {
		statusLabel.SetText("[•] Capturando tela...")
		go func() {
			path, err := TakeScreenshot(serial)
			if err != nil {
				statusLabel.SetText("[!] Erro: " + err.Error())
			} else {
				statusLabel.SetText("[+] Salvo em: " + path)
			}
		}()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("[•] Ações e Comandos Rápidos", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Dispositivo alvo: "+serial),
		widget.NewSeparator(),
		screenshotBtn,
		airplaneBtn,
		clearLogcatBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("[!] Opções de Reinicialização", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		rebootBtn,
		rebootRecBtn,
		rebootBlBtn,
		widget.NewSeparator(),
		statusLabel,
		widget.NewButton("Fechar", func() { d.Close() }),
	)

	d.SetContent(container.NewPadded(content))
	d.Show()
}

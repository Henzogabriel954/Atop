package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func showToolsScreen() {
	stopMonitoring()
	clearMonitorListeners()

	state.Window.Canvas().SetOnTypedRune(nil)

	// --- Ferramenta 1: Análise de Armazenamento (Gráfico de Pizza) ---
	storageBtn := widget.NewButtonWithIcon("[◈] Analisar Armazenamento", theme.StorageIcon(), func() {
		showStorageScreen()
	})
	storageDesc := widget.NewLabel("Varre o armazenamento do dispositivo e exibe um gráfico de pizza detalhado com as categorias que mais consomem espaço.")
	storageDesc.Wrapping = fyne.TextWrapWord
	storageDesc.TextStyle = fyne.TextStyle{Italic: true}

	// --- Ferramenta 2: Gravação de Tela ---
	var recordBtn *widget.Button
	recordBtn = widget.NewButton("[REC] Iniciar Gravação", func() {
		if state.IsRecording {
			recordBtn.SetText("Salvando gravação...")
			recordBtn.Disable()

			go func() {
				path, err := stopScreenRecording(state.CurrentDevice.Serial)
				fyne.Do(func() {
					recordBtn.Enable()
					recordBtn.SetText("[REC] Iniciar Gravação")
					recordBtn.Importance = widget.MediumImportance
					recordBtn.Refresh()

					if err != nil {
						dialog.ShowError(err, state.Window)
					} else {
						dialog.ShowInformation("Gravação Concluída", "Vídeo salvo com sucesso em:\n"+path, state.Window)
					}
				})
			}()
		} else {
			err := startScreenRecording(state.CurrentDevice.Serial)
			if err != nil {
				dialog.ShowError(err, state.Window)
				return
			}
			recordBtn.SetText("[■] Parar Gravação")
			recordBtn.Importance = widget.DangerImportance
			recordBtn.Refresh()
		}
	})

	if state.IsRecording {
		recordBtn.SetText("[■] Parar Gravação")
		recordBtn.Importance = widget.DangerImportance
	}

	recordDesc := widget.NewLabel("Grava a tela do dispositivo móvel (máx 3 min) e salva o arquivo em ~/Videos/.")
	recordDesc.Wrapping = fyne.TextWrapWord
	recordDesc.TextStyle = fyne.TextStyle{Italic: true}

	// --- Ferramenta 3: Instalar APK ---
	installBtn := widget.NewButtonWithIcon("[+] Instalar Pacote APK", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			apkPath := reader.URI().Path()
			reader.Close()

			progressBar := widget.NewProgressBarInfinite()
			progressDialog := dialog.NewCustomWithoutButtons("Instalando APK no celular...", progressBar, state.Window)
			progressDialog.Show()

			go func() {
				installErr := installAPK(state.CurrentDevice.Serial, apkPath)
				progressDialog.Hide()
				if installErr != nil {
					dialog.ShowError(installErr, state.Window)
				} else {
					dialog.ShowInformation("Instalação Concluída", "Aplicativo instalado com sucesso!", state.Window)
				}
			}()
		}, state.Window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".apk"}))
		fd.Show()
	})

	installDesc := widget.NewLabel("Selecione um arquivo .apk local no seu computador para enviar e instalar no dispositivo conectado.")
	installDesc.Wrapping = fyne.TextWrapWord
	installDesc.TextStyle = fyne.TextStyle{Italic: true}

	// --- Botão Voltar ---
	backBtn := widget.NewButtonWithIcon("Voltar ao Monitor", theme.NavigateBackIcon(), func() {
		showMonitorScreen()
	})

	// --- Layout ---
	header := container.NewBorder(nil, nil, backBtn, nil,
		container.NewCenter(widget.NewLabelWithStyle("Ferramentas do Dispositivo", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
	)

	content := container.NewPadded(
		container.NewVBox(
			header,
			widget.NewSeparator(),

			widget.NewCard("Armazenamento e Disco", "", container.NewVBox(
				storageBtn,
				storageDesc,
			)),

			widget.NewCard("Captura e Gravação de Tela", "", container.NewVBox(
				recordBtn,
				recordDesc,
			)),

			widget.NewCard("Gerenciamento de Aplicativos", "", container.NewVBox(
				installBtn,
				installDesc,
			)),

			layout.NewSpacer(),
		),
	)

	state.Window.SetContent(content)
}

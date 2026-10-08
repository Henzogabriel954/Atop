package main

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// --- WIDGET OTIMIZADO: BARRA FINA COM CACHE DE RENDERIZAÇÃO ---
type ThinBar struct {
	Title     *canvas.Text
	ValueText *canvas.Text
	BarFill   *canvas.Rectangle
	BarBg     *canvas.Rectangle
	Container *fyne.Container
}

func NewThinBar(label string) *ThinBar {
	title := canvas.NewText(label, theme.ForegroundColor())
	title.TextSize = 11
	title.TextStyle = fyne.TextStyle{Bold: true}

	valText := canvas.NewText("0%", theme.ForegroundColor())
	valText.TextSize = 11
	valText.Alignment = fyne.TextAlignTrailing

	bg := canvas.NewRectangle(color.RGBA{35, 35, 40, 200})
	bg.CornerRadius = 3
	fill := canvas.NewRectangle(fallbackAccentColor)
	fill.CornerRadius = 3

	barContainer := container.NewMax(bg, container.NewHBox(fill))
	content := container.NewBorder(nil, nil, title, valText, barContainer)

	return &ThinBar{
		Title:     title,
		ValueText: valText,
		BarFill:   fill,
		BarBg:     bg,
		Container: content,
	}
}

type percentLayout struct {
	percent float64
}

func (l *percentLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	bg := objects[0]
	fill := objects[1]
	bg.Resize(size)
	bg.Move(fyne.NewPos(0, 0))
	fillWidth := float32(float64(size.Width) * l.percent)
	fill.Resize(fyne.NewSize(fillWidth, size.Height))
	fill.Move(fyne.NewPos(0, 0))
}

func (l *percentLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(80, 6)
}

type ActiveBar struct {
	Widget    *ThinBar
	Layout    *percentLayout
	BarBox    *fyne.Container
	lastVal   float64
	lastText  string
	lastColor color.Color
}

func NewActiveBar(label string) *ActiveBar {
	tb := NewThinBar(label)
	pl := &percentLayout{percent: 0}
	barBox := container.New(pl, tb.BarBg, tb.BarFill)
	finalContainer := container.NewBorder(nil, nil,
		container.NewPadded(tb.Title),
		container.NewPadded(tb.ValueText),
		container.NewPadded(barBox),
	)
	tb.Container = finalContainer
	return &ActiveBar{Widget: tb, Layout: pl, BarBox: barBox}
}

func (ab *ActiveBar) Update(val float64, text string, accent color.Color) {
	if val > 1.0 {
		val = 1.0
	}
	if val < 0.0 {
		val = 0.0
	}

	targetColor := accent
	if val > 0.85 {
		targetColor = color.RGBA{R: 240, G: 80, B: 80, A: 255}
	} else if val > 0.70 {
		targetColor = color.RGBA{R: 240, G: 190, B: 50, A: 255}
	}

	// Otimização: evita refreshes no OpenGL se o valor visual não mudou
	if math.Abs(val-ab.lastVal) < 0.003 && ab.lastText == text && ab.lastColor == targetColor {
		return
	}

	ab.lastVal = val
	ab.lastText = text
	ab.lastColor = targetColor

	ab.Layout.percent = val
	ab.Widget.ValueText.Text = text
	ab.Widget.BarFill.FillColor = targetColor

	ab.Widget.ValueText.Refresh()
	ab.Widget.BarFill.Refresh()
	ab.BarBox.Refresh()
}

// --- GERENCIAMENTO LIMPO DE CICLO DE VIDA DE LISTENERS ---
func clearMonitorListeners() {
	state.LifecycleMutex.Lock()
	defer state.LifecycleMutex.Unlock()

	for _, l := range state.MonitorListeners {
		state.CpuPercent.RemoveListener(l)
		state.RamUsed.RemoveListener(l)
		state.SwapUsed.RemoveListener(l)
		state.BatteryPercent.RemoveListener(l)
		state.Temperature.RemoveListener(l)
		state.TableRefresher.RemoveListener(l)
	}
	state.MonitorListeners = nil
}

func clearDeviceListListener() {
	state.LifecycleMutex.Lock()
	defer state.LifecycleMutex.Unlock()

	if state.DeviceListListener != nil {
		state.DeviceListRefresher.RemoveListener(state.DeviceListListener)
		state.DeviceListListener = nil
	}
}

func stopDeviceListPolling() {
	state.LifecycleMutex.Lock()
	defer state.LifecycleMutex.Unlock()

	if state.DeviceListCancel != nil {
		state.DeviceListCancel()
		state.DeviceListCancel = nil
	}
}

func stopMonitoring() {
	state.LifecycleMutex.Lock()
	if state.MonitorCancelCtx != nil {
		state.MonitorCancelCtx()
		state.MonitorCancelCtx = nil
	}
	state.IsMonitoring = false
	state.LifecycleMutex.Unlock()
}

// --- TELA: LISTA DE DISPOSITIVOS ---
func showDeviceListScreen() {
	stopMonitoring()
	clearMonitorListeners()
	clearDeviceListListener()
	stopDeviceListPolling()

	state.Window.Canvas().SetOnTypedRune(nil)

	state.LifecycleMutex.Lock()
	pollCtx, pollCancel := context.WithCancel(context.Background())
	state.DeviceListCancel = pollCancel
	state.LifecycleMutex.Unlock()

	var devices []Device
	var devMutex sync.Mutex
	devices = fetchDevices()

	list := widget.NewList(
		func() int {
			devMutex.Lock()
			defer devMutex.Unlock()
			return len(devices)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(celularIconResource)
			return container.NewHBox(
				icon,
				widget.NewLabel("Modelo"),
				layout.NewSpacer(),
				widget.NewLabel("Serial"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			devMutex.Lock()
			defer devMutex.Unlock()
			if id < len(devices) {
				dev := devices[id]
				box := item.(*fyne.Container)
				lblModel := box.Objects[1].(*widget.Label)
				lblSerial := box.Objects[3].(*widget.Label)

				lblModel.SetText(dev.Model)
				lblSerial.SetText(dev.Serial)

				if dev.IsUnauthorized {
					lblModel.Importance = widget.WarningImportance
				} else {
					lblModel.Importance = widget.MediumImportance
				}
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		devMutex.Lock()
		if id >= len(devices) {
			devMutex.Unlock()
			return
		}
		dev := devices[id]
		devMutex.Unlock()

		if dev.IsUnauthorized {
			dialog.ShowInformation("Dispositivo Não Autorizado",
				"O dispositivo "+dev.Serial+" ainda não autorizou a depuração USB.\n\nDesbloqueie a tela do celular e marque 'Sempre permitir deste computador'.",
				state.Window)
			list.UnselectAll()
			return
		}

		if strings.Contains(dev.Serial, ":") {
			var d *dialog.CustomDialog
			content := container.NewVBox(
				widget.NewLabel(dev.Serial),
				widget.NewButtonWithIcon("Monitorar", theme.MediaPlayIcon(), func() {
					d.Hide()
					state.CurrentDevice = dev
					showMonitorScreen()
				}),
				widget.NewButtonWithIcon("Desconectar", theme.CancelIcon(), func() {
					disconnectDevice(dev.Serial)
					d.Hide()
				}),
			)
			d = dialog.NewCustom(dev.Model, "Cancelar", content, state.Window)
			d.Show()
		} else {
			state.CurrentDevice = dev
			showMonitorScreen()
		}
		list.UnselectAll()
	}

	// Listener único registrado de forma segura
	devListener := binding.NewDataListener(func() { list.Refresh() })
	state.DeviceListListener = devListener
	state.DeviceListRefresher.AddListener(devListener)

	// Polling de dispositivos sem vazamento de goroutines
	go func(ctx context.Context) {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				newDevs := fetchDevices()
				devMutex.Lock()
				devices = newDevs
				devMutex.Unlock()
				val, _ := state.DeviceListRefresher.Get()
				state.DeviceListRefresher.Set(!val)
			}
		}
	}(pollCtx)

	// Status do mDNS no rodapé
	mdnsStatus := canvas.NewText("[•] Radar mDNS: Inativo", color.RGBA{R: 140, G: 140, B: 150, A: 255})
	mdnsStatus.TextSize = 11

	updateMdnsStatus := func() {
		state.MdnsMutex.Lock()
		active := state.MdnsRunning
		state.MdnsMutex.Unlock()

		if active {
			mdnsStatus.Text = "[•] Radar Wi-Fi: Ativo (Scan 5m)"
			mdnsStatus.Color = color.RGBA{R: 16, G: 185, B: 129, A: 255}
		} else {
			mdnsStatus.Text = "[•] Radar Wi-Fi: Desativado"
			mdnsStatus.Color = color.RGBA{R: 140, G: 140, B: 150, A: 255}
		}
		mdnsStatus.Refresh()
	}
	updateMdnsStatus()

	go func(ctx context.Context) {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				updateMdnsStatus()
			}
		}
	}(pollCtx)

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showSettingsScreen()
	})

	btnRescanMdns := widget.NewButtonWithIcon("Buscar Wi-Fi Agora", theme.SearchIcon(), func() {
		restartMdnsLoop()
		newDevs := fetchDevices()
		devMutex.Lock()
		devices = newDevs
		devMutex.Unlock()
		val, _ := state.DeviceListRefresher.Get()
		state.DeviceListRefresher.Set(!val)
	})

	header := container.NewBorder(nil, nil, nil, settingsBtn,
		container.NewCenter(widget.NewLabelWithStyle("Dispositivos Conectados", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
	)

	footer := container.NewCenter(container.NewHBox(
		container.NewCenter(mdnsStatus),
		btnRescanMdns,
	))
	state.Window.SetContent(container.NewBorder(header, footer, nil, nil, list))
}

// --- TELA: MONITORAMENTO MINIMALISTA E FLUIDA ---
func showMonitorScreen() {
	stopDeviceListPolling()
	clearDeviceListListener()
	clearMonitorListeners()
	stopMonitoring()

	state.LifecycleMutex.Lock()
	monCtx, monCancel := context.WithCancel(context.Background())
	state.MonitorCancelCtx = monCancel
	state.IsMonitoring = true
	state.LifecycleMutex.Unlock()

	// Cor de acento vibrante
	var accentColor color.Color = fallbackAccentColor
	if dt, ok := state.App.Settings().Theme().(*DynamicTheme); ok && dt.accentColor != nil {
		accentColor = dt.accentColor
	}

	// --- 1. CABEÇALHO MINIMALISTA E COMPACTO ---
	titleLabel := widget.NewLabelWithStyle(state.CurrentDevice.Model, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	infoLabel := canvas.NewText("Identificando dispositivo...", color.RGBA{R: 150, G: 150, B: 160, A: 255})
	infoLabel.TextSize = 11

	recBadge := canvas.NewText("[REC]", color.RGBA{R: 240, G: 80, B: 80, A: 255})
	recBadge.TextSize = 11
	recBadge.TextStyle = fyne.TextStyle{Bold: true}
	recBadge.Hidden = true

	btnBack := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		stopScrcpy()
		showDeviceListScreen()
	})

	btnTools := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), func() {
		showToolsScreen()
	})

	headerCenter := container.NewVBox(
		container.NewHBox(layout.NewSpacer(), titleLabel, recBadge, layout.NewSpacer()),
		container.NewCenter(infoLabel),
	)
	header := container.NewBorder(nil, nil, btnBack, btnTools, headerCenter)

	// Carrega detalhes do dispositivo uma vez
	go func() {
		info := fetchDeviceInfoOnce(state.CurrentDevice.Serial)
		state.DeviceInfoMutex.Lock()
		state.DeviceInfoCache = info
		state.DeviceInfoMutex.Unlock()

		desc := ""
		if info.AndroidVersion != "" {
			desc += "Android " + info.AndroidVersion
		}
		if info.Uptime != "" {
			if desc != "" {
				desc += "  •  "
			}
			desc += info.Uptime
		}
		if desc == "" {
			desc = state.CurrentDevice.Serial
		}
		infoLabel.Text = desc
		infoLabel.Refresh()
	}()

	// --- 2. BARRAS DE ESTATÍSTICAS ---
	barCPU := NewActiveBar("CPU")
	barRAM := NewActiveBar("RAM")
	barSWAP := NewActiveBar("SWP")
	barBAT := NewActiveBar("BAT")
	barTEMP := NewActiveBar("TMP")

	statsGrid := container.NewVBox(
		barCPU.Widget.Container,
		barRAM.Widget.Container,
		barSWAP.Widget.Container,
		container.NewGridWithColumns(2, barBAT.Widget.Container, barTEMP.Widget.Container),
	)

	// --- 3. CONTROLES SCRCPY EM LINHA COMPACTA (Estilo Pílulas com Símbolos) ---
	btnMirror := widget.NewButton("◈ Espelho", func() { toggleScrcpy(ModeMirror) })
	btnGhost := widget.NewButton("◇ Ghost", func() { toggleScrcpy(ModeGhost, "--turn-screen-off", "--no-video", "--no-audio") })
	btnMouse := widget.NewButton("▸ Mouse", func() { toggleScrcpy(ModeMouse, "--no-video", "--no-audio", "-M") })

	scrcpyRow := container.NewGridWithColumns(3, btnMirror, btnGhost, btnMouse)

	btnReboot := widget.NewButtonWithIcon("Reiniciar", theme.ViewRefreshIcon(), func() {
		dialog.ShowConfirm("Reiniciar Aparelho", "Deseja realmente reiniciar o celular?", func(ok bool) {
			if ok {
				exec.Command("adb", "-s", state.CurrentDevice.Serial, "reboot").Start()
			}
		}, state.Window)
	})

	btnLock := widget.NewButtonWithIcon("Ligar/Desligar Tela", theme.MediaStopIcon(), func() {
		exec.Command("adb", "-s", state.CurrentDevice.Serial, "shell", "input", "keyevent", "26").Run()
	})

	quickActions := container.NewGridWithColumns(2, btnLock, btnReboot)
	controlsCard := container.NewVBox(scrcpyRow, quickActions)

	// --- 4. TABELA DE PROCESSOS LEVE (Pool) ---
	const maxLabels = 25
	procLabels := make([]*canvas.Text, maxLabels)
	procContainer := container.NewVBox()

	tableHeader := canvas.NewText(fmt.Sprintf("%-7s %-24s %7s %8s", "PID", "PROCESSO", "CPU%", "RAM"), theme.ForegroundColor())
	tableHeader.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	tableHeader.TextSize = 11
	procContainer.Add(tableHeader)

	for i := 0; i < maxLabels; i++ {
		lbl := canvas.NewText("", theme.ForegroundColor())
		lbl.TextSize = 11
		lbl.TextStyle = fyne.TextStyle{Monospace: true}
		lbl.Hidden = true
		procLabels[i] = lbl
		procContainer.Add(lbl)
	}

	// --- 5. REGISTRO DE LISTENERS DE CICLO DE VIDA ÚNICO ---
	cpuListener := binding.NewDataListener(func() {
		v, _ := state.CpuPercent.Get()
		barCPU.Update(v, fmt.Sprintf("%.1f%%", v*100), accentColor)
	})
	ramListener := binding.NewDataListener(func() {
		u, _ := state.RamUsed.Get()
		t, _ := state.RamTotal.Get()
		if t > 0 {
			barRAM.Update(u/t, fmt.Sprintf("%.1f/%.1f GB", u/1024/1024, t/1024/1024), accentColor)
		}
	})
	swapListener := binding.NewDataListener(func() {
		u, _ := state.SwapUsed.Get()
		t, _ := state.SwapTotal.Get()
		if t > 0 {
			barSWAP.Update(u/t, fmt.Sprintf("%.1f/%.1f GB", u/1024/1024, t/1024/1024), accentColor)
		}
	})
	batListener := binding.NewDataListener(func() {
		lvl, _ := state.BatteryPercent.Get()
		state.DeviceInfoMutex.Lock()
		charging := state.DeviceInfoCache.BatteryCharging
		state.DeviceInfoMutex.Unlock()

		txt := fmt.Sprintf("%.0f%%", lvl)
		if charging {
			txt += " +"
		}
		barBAT.Update(lvl/100.0, txt, accentColor)
	})
	tempListener := binding.NewDataListener(func() {
		temp, _ := state.Temperature.Get()
		barTEMP.Update(temp/60.0, fmt.Sprintf("%.1f°C", temp), accentColor)
	})

	tableListener := binding.NewDataListener(func() {
		state.TableMutex.Lock()
		procs := state.ProcessList
		state.TableMutex.Unlock()

		for i := 0; i < maxLabels; i++ {
			if i < len(procs) {
				p := procs[i]
				line := fmt.Sprintf("%-7s %-24s %7s %8s",
					truncateString(p.PID, 7),
					truncateString(p.Name, 24),
					p.CPU,
					p.RAM)

				procLabels[i].Text = line

				cpuVal, _ := strconv.ParseFloat(strings.TrimSuffix(p.CPU, "%"), 64)
				if cpuVal > 50 {
					procLabels[i].Color = color.RGBA{R: 240, G: 80, B: 80, A: 255}
				} else if cpuVal > 20 {
					procLabels[i].Color = color.RGBA{R: 240, G: 190, B: 50, A: 255}
				} else {
					procLabels[i].Color = theme.ForegroundColor()
				}
				procLabels[i].Hidden = false
			} else {
				procLabels[i].Hidden = true
			}
			procLabels[i].Refresh()
		}

		// Destaque nos botões Scrcpy
		btnGhost.Importance = widget.MediumImportance
		btnMouse.Importance = widget.MediumImportance
		btnMirror.Importance = widget.MediumImportance

		if state.CurrentMode == ModeGhost {
			btnGhost.Importance = widget.HighImportance
		} else if state.CurrentMode == ModeMouse {
			btnMouse.Importance = widget.HighImportance
		} else if state.CurrentMode == ModeMirror {
			btnMirror.Importance = widget.HighImportance
		}
		btnGhost.Refresh()
		btnMouse.Refresh()
		btnMirror.Refresh()

		// Status de gravação
		state.RecordingMutex.Lock()
		isRec := state.IsRecording
		state.RecordingMutex.Unlock()
		if isRec {
			recBadge.Hidden = false
		} else {
			recBadge.Hidden = true
		}
		recBadge.Refresh()
	})

	// Anexa listeners à lista de desregistro
	state.LifecycleMutex.Lock()
	state.MonitorListeners = []binding.DataListener{cpuListener, ramListener, swapListener, batListener, tempListener, tableListener}
	state.LifecycleMutex.Unlock()

	state.CpuPercent.AddListener(cpuListener)
	state.RamUsed.AddListener(ramListener)
	state.SwapUsed.AddListener(swapListener)
	state.BatteryPercent.AddListener(batListener)
	state.Temperature.AddListener(tempListener)
	state.TableRefresher.AddListener(tableListener)

	// --- 6. ATALHOS DE TECLADO ---
	state.Window.Canvas().SetOnTypedRune(func(r rune) {
		switch r {
		case 's', 'S':
			toggleScrcpy(ModeGhost, "--turn-screen-off", "--no-video", "--no-audio")
		case 'm', 'M':
			toggleScrcpy(ModeMouse, "--no-video", "--no-audio", "-M")
		case 'e', 'E':
			toggleScrcpy(ModeMirror)
		case 'r', 'R':
			dialog.ShowConfirm("Reiniciar Aparelho", "Deseja reiniciar o dispositivo?", func(b bool) {
				if b {
					exec.Command("adb", "-s", state.CurrentDevice.Serial, "reboot").Start()
				}
			}, state.Window)
		case 'l', 'L':
			exec.Command("adb", "-s", state.CurrentDevice.Serial, "shell", "input", "keyevent", "26").Run()
		case 't', 'T':
			showToolsScreen()
		case 'q', 'Q':
			stopScrcpy()
			showDeviceListScreen()
		}
	})

	// --- 7. MONTAGEM DO LAYOUT MINIMALISTA ---
	content := container.NewPadded(
		container.NewVBox(
			header,
			widget.NewSeparator(),
			statsGrid,
			widget.NewSeparator(),
			controlsCard,
			widget.NewSeparator(),
			procContainer,
		),
	)

	state.Window.SetContent(content)
	go monitorLoop(monCtx)
}

func truncateString(str string, num int) string {
	if len(str) > num {
		return str[0:num]
	}
	return str
}

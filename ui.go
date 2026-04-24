package main

import (
	"fmt"
	"image/color"
	"os/exec"
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

// --- WIDGET CUSTOMIZADO: BARRA FINA (Mantido igual) ---
type ThinBar struct {
	Title     *canvas.Text
	ValueText *canvas.Text
	BarFill   *canvas.Rectangle
	BarBg     *canvas.Rectangle
	Container *fyne.Container
}

func NewThinBar(label string) *ThinBar {
	title := canvas.NewText(label, theme.ForegroundColor())
	title.TextSize = 12
	title.TextStyle = fyne.TextStyle{Bold: true}

	valText := canvas.NewText("0%", theme.ForegroundColor())
	valText.TextSize = 12
	valText.Alignment = fyne.TextAlignTrailing

	bg := canvas.NewRectangle(color.RGBA{40, 40, 40, 150})
	bg.CornerRadius = 2
	fill := canvas.NewRectangle(theme.PrimaryColor())
	fill.CornerRadius = 2

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
	if len(objects) < 2 { return }
	bg := objects[0]
	fill := objects[1]
	bg.Resize(size)
	bg.Move(fyne.NewPos(0, 0))
	fillWidth := float32(float64(size.Width) * l.percent)
	fill.Resize(fyne.NewSize(fillWidth, size.Height))
	fill.Move(fyne.NewPos(0, 0))
}
func (l *percentLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(100, 8) 
}

type ActiveBar struct {
	Widget *ThinBar
	Layout *percentLayout
	BarBox *fyne.Container
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
	if val > 1.0 { val = 1.0 }
	if val < 0.0 { val = 0.0 }
	ab.Layout.percent = val
	ab.Widget.ValueText.Text = text


	ab.Widget.ValueText.Refresh()
	if val > 0.8 {
		ab.Widget.BarFill.FillColor = color.RGBA{R: 240, G: 80, B: 80, A: 255}
	} else {
		ab.Widget.BarFill.FillColor = accent
	}
	ab.Widget.BarFill.Refresh()
	ab.BarBox.Refresh()
}

// --- TELAS ---

func showDeviceListScreen() {
	state.Window.Canvas().SetOnTypedRune(nil) 

    state.IsMonitoring = false
	if state.MonitorStopChk != nil {
		close(state.MonitorStopChk)
		state.MonitorStopChk = nil
	}
	if state.DeviceListStopChk != nil {
		close(state.DeviceListStopChk)
	}
	state.DeviceListStopChk = make(chan bool)

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
			var iconWidget fyne.CanvasObject = widget.NewLabel("📱")

			res, err := fyne.LoadResourceFromPath("celular.png")
			if err == nil {
				icon := widget.NewIcon(res)
				iconWidget = icon
			}

			return container.NewHBox(
				iconWidget,
				widget.NewLabel("Modelo"),
				layout.NewSpacer(),
				widget.NewLabel("Serial"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			devMutex.Lock()
			if id < len(devices) {
				dev := devices[id]
				item.(*fyne.Container).Objects[1].(*widget.Label).SetText(dev.Model)
				item.(*fyne.Container).Objects[3].(*widget.Label).SetText(dev.Serial)
			}
			devMutex.Unlock()
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		devMutex.Lock()
		if id < len(devices) {
			state.CurrentDevice = devices[id]
			showMonitorScreen()
		}
		devMutex.Unlock()
	}

	state.DeviceListRefresher.AddListener(binding.NewDataListener(func() { list.Refresh() }))

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-state.DeviceListStopChk:
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
	}()
    
    header := container.NewCenter(widget.NewLabelWithStyle("Selecione o Dispositivo", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	state.Window.SetContent(container.NewBorder(header, nil, nil, nil, list))
}

func showMonitorScreen() {
	if state.DeviceListStopChk != nil {
		close(state.DeviceListStopChk)
		state.DeviceListStopChk = nil
	}

	state.IsMonitoring = true
	state.MonitorStopChk = make(chan bool)

	// --- 1. CONFIGURAÇÃO DE BARRAS ---
	barCPU := NewActiveBar("CPU")
	barRAM := NewActiveBar("RAM")
	barSWAP := NewActiveBar("SWAP")

	var accentColor color.Color = color.RGBA{R: 67, G: 191, B: 109, A: 255} 
	if dt, ok := state.App.Settings().Theme().(*DynamicTheme); ok {
		accentColor = dt.accentColor
	}

	statsContainer := container.NewVBox(
		barCPU.Widget.Container,
		barRAM.Widget.Container,
		barSWAP.Widget.Container,
	)

	// --- 2. LISTA DE PROCESSOS DINÂMICA (Pool de Labels) ---
	
	// Criamos um pool grande de labels (ex: 30)
	// Só mostraremos os que couberem na tela
	const maxLabels = 30 
	procLabels := make([]*canvas.Text, maxLabels)
	procContainer := container.NewVBox()
	
	// Header da Tabela
	headerText := canvas.NewText(fmt.Sprintf("%-8s %-22s %8s %8s", "PID", "NOME", "CPU%", "RAM"), theme.ForegroundColor())
	headerText.TextStyle = fyne.TextStyle{Bold: true}
	headerText.TextSize = 12
	procContainer.Add(headerText)
	
	// Espaçador visual
	procContainer.Add(canvas.NewRectangle(color.Transparent)) 

	// Inicializa o Pool de Labels (todos ocultos inicialmente)
	for i := 0; i < maxLabels; i++ {
		lbl := canvas.NewText("", theme.ForegroundColor())
		lbl.TextSize = 12
		lbl.TextStyle = fyne.TextStyle{Monospace: true} 
		lbl.Hidden = true // Começa escondido
		procLabels[i] = lbl
		procContainer.Add(lbl)
	}

	// --- 3. BOTÕES ---
	btnGhost := widget.NewButton("Ghost (S)", func() { toggleScrcpy(ModeGhost, "--turn-screen-off", "--no-video", "--no-audio") })
	btnMouse := widget.NewButton("Mouse (M)", func() { toggleScrcpy(ModeMouse, "--no-video", "--no-audio", "-M") })
	btnMirror := widget.NewButton("Mirror (E)", func() { toggleScrcpy(ModeMirror) })
	btnReboot := widget.NewButton("Reboot (R)", func() { 
        dialog.ShowConfirm("Reboot", "Reiniciar?", func(b bool) {
            if b { exec.Command("adb", "-s", state.CurrentDevice.Serial, "reboot").Start() }
        }, state.Window)
    })
    
    btnBack := widget.NewButton("Voltar (Q)", func() {
        stopScrcpy()
        showDeviceListScreen()
    })

	actions := container.NewGridWithColumns(5, btnMirror, btnGhost, btnMouse, btnReboot, btnBack)

	// --- 4. ATUALIZAÇÃO DE DADOS ---
	state.CpuPercent.AddListener(binding.NewDataListener(func() {
		v, _ := state.CpuPercent.Get()
		barCPU.Update(v, fmt.Sprintf("%.1f%%", v*100), accentColor)
	}))
	
	state.RamUsed.AddListener(binding.NewDataListener(func() {
		u, _ := state.RamUsed.Get()
		t, _ := state.RamTotal.Get()
		if t > 0 {
			barRAM.Update(u/t, fmt.Sprintf("%.1f/%.1f GB", u/1024/1024, t/1024/1024), accentColor)
		}
	}))

	state.SwapUsed.AddListener(binding.NewDataListener(func() {
		u, _ := state.SwapUsed.Get()
		t, _ := state.SwapTotal.Get()
		if t > 0 {
			barSWAP.Update(u/t, fmt.Sprintf("%.1f/%.1f GB", u/1024/1024, t/1024/1024), accentColor)
		}
	}))

	// LISTENER PRINCIPAL (Atualiza Tabela e Calcula Tamanho)
	state.TableRefresher.AddListener(binding.NewDataListener(func() {
		state.TableMutex.Lock()
		defer state.TableMutex.Unlock()
		
		// 1. Calcular espaço disponível
		winHeight := state.Window.Canvas().Size().Height
		
		// Altura Fixa Aproximada (Topo + Barras + Header + Botões + Margens)
		// Topo (Title+Sep): ~40
		// Barras: ~100
		// Header Tabela: ~30
		// Botões + Spacer: ~60
		// Margens (Padded x3): ~40
		// TOTAL OVERHEAD: ~270px (estimativa conservadora)
		overhead := float32(270)
		
		availableSpace := winHeight - overhead
		if availableSpace < 0 { availableSpace = 0 }
		
		// Altura de uma linha de texto (Size 12 + espaçamento padrão do VBox)
		rowHeight := float32(19) // Ajuste fino se necessário
		
		// Quantas linhas cabem?
		visibleRows := int(availableSpace / rowHeight)
		if visibleRows > maxLabels { visibleRows = maxLabels }

		// 2. Preencher e Mostrar/Esconder
		for i := 0; i < maxLabels; i++ {
			// Se o índice for menor que o espaço disponível E tivermos dados
			if i < visibleRows && i < len(state.ProcessList) {
				p := state.ProcessList[i]
				
				// Aumentei o truncamento do nome para 22 chars
				line := fmt.Sprintf("%-8s %-22s %8s %8s", 
                    truncateString(p.PID, 8), 
                    truncateString(p.Name, 22), 
                    p.CPU, 
                    p.RAM)
				
				procLabels[i].Text = line
				procLabels[i].Color = theme.ForegroundColor()
				procLabels[i].Hidden = false // MOSTRA
			} else {
				procLabels[i].Hidden = true  // ESCONDE (Não cabe ou sem dados)
			}
			procLabels[i].Refresh()
		}

		// Atualiza Botões
		resetImportance := func() {
			btnGhost.Importance = widget.MediumImportance
			btnMouse.Importance = widget.MediumImportance
			btnMirror.Importance = widget.MediumImportance
		}
		resetImportance()
		
		if state.CurrentMode == ModeGhost { btnGhost.Importance = widget.DangerImportance }
		if state.CurrentMode == ModeMouse { btnMouse.Importance = widget.DangerImportance }
		if state.CurrentMode == ModeMirror { btnMirror.Importance = widget.DangerImportance }
		
		btnGhost.Refresh(); btnMouse.Refresh(); btnMirror.Refresh()
	}))

	// --- 5. ATALHOS ---
	state.Window.Canvas().SetOnTypedRune(func(r rune) {
		switch r {
		case 's', 'S':
			toggleScrcpy(ModeGhost, "--turn-screen-off", "--no-video", "--no-audio")
		case 'm', 'M':
			toggleScrcpy(ModeMouse, "--no-video", "--no-audio", "-M")
		case 'e', 'E':
			toggleScrcpy(ModeMirror)
		case 'r', 'R':
            exec.Command("adb", "-s", state.CurrentDevice.Serial, "reboot").Start()
		case 'l', 'L':
			exec.Command("adb", "-s", state.CurrentDevice.Serial, "shell", "input", "keyevent", "26").Run()
		case 'q', 'Q':
			stopScrcpy()
			showDeviceListScreen()
		}
	})

	// --- 6. LAYOUT ---
	margin := container.NewPadded(
		container.NewPadded(
			container.NewPadded(
				container.NewVBox(
					widget.NewLabelWithStyle(state.CurrentDevice.Model, fyne.TextAlignCenter, fyne.TextStyle{Bold:true}),
					widget.NewSeparator(),
					statsContainer,
					widget.NewSeparator(),
					procContainer,
					layout.NewSpacer(),
					actions,
				),
			),
		),
	)

	state.Window.SetContent(margin)
	go monitorLoop()
}

func truncateString(str string, num int) string {
	if len(str) > num {
		return str[0:num]
	}
	return str
}

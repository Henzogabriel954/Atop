package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type pieAngleSlice struct {
	startAngle float64
	endAngle   float64
	color      color.RGBA
}

func showStorageScreen() {
	stopMonitoring()
	clearMonitorListeners()

	state.Window.Canvas().SetOnTypedRune(func(r rune) {
		if r == 'q' || r == 'Q' {
			showToolsScreen()
		}
	})

	statusLabel := widget.NewLabelWithStyle("[•] Clique no botão abaixo para escanear o armazenamento.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	progressBar := widget.NewProgressBarInfinite()
	progressContainer := container.NewPadded(progressBar)
	progressContainer.Hide()

	chartContainer := container.NewCenter()
	legendContainer := container.NewVBox()
	summaryLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	var btnScan *widget.Button

	runScan := func() {
		btnScan.Disable()
		progressContainer.Show()
		statusLabel.SetText("[•] Escaneando partições e diretórios do dispositivo via ADB...")
		statusLabel.Refresh()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			report, err := scanDeviceStorage(ctx, state.CurrentDevice.Serial)

			fyne.Do(func() {
				progressContainer.Hide()
				btnScan.Enable()

				if err != nil || report == nil {
					statusLabel.SetText("[!] Falha ao realizar a varredura de armazenamento.")
					dialog.ShowError(fmt.Errorf("Erro ao obter dados: %v", err), state.Window)
					return
				}

				statusLabel.SetText("[+] Varredura concluída com sucesso.")

				// Atualiza Gráfico de Pizza
				chartRaster := buildPieChartRaster(report, 220)
				chartContainer.Objects = []fyne.CanvasObject{chartRaster}
				chartContainer.Refresh()

				// Atualiza Legenda
				legendContainer.Objects = nil
				for _, s := range report.Slices {
					// Indicador visual colorido com símbolo ■
					swatch := canvas.NewText("■", s.Color)
					swatch.TextSize = 14
					swatch.TextStyle = fyne.TextStyle{Bold: true}

					catText := canvas.NewText(fmt.Sprintf("%-18s %8s (%4.1f%%)", s.Name, s.Formatted, s.Percentage), theme.ForegroundColor())
					catText.TextSize = 12
					catText.TextStyle = fyne.TextStyle{Monospace: true}

					row := container.NewHBox(swatch, catText)
					legendContainer.Add(row)
				}
				legendContainer.Refresh()

				// Atualiza Resumo de Capacidade
				freePct := 0.0
				usedPct := 0.0
				if report.TotalBytes > 0 {
					usedPct = (float64(report.UsedBytes) / float64(report.TotalBytes)) * 100.0
					freePct = (float64(report.FreeBytes) / float64(report.TotalBytes)) * 100.0
				}

				summaryLabel.SetText(fmt.Sprintf("Total: %s  |  Usado: %s (%.1f%%)  |  Livre: %s (%.1f%%)",
					formatBytes(report.TotalBytes),
					formatBytes(report.UsedBytes),
					usedPct,
					formatBytes(report.FreeBytes),
					freePct,
				))
				summaryLabel.Refresh()
			})
		}()
	}

	btnScan = widget.NewButtonWithIcon("[▸] Iniciar Varredura de Armazenamento", theme.SearchIcon(), func() {
		runScan()
	})
	btnScan.Importance = widget.HighImportance

	backBtn := widget.NewButtonWithIcon("Voltar", theme.NavigateBackIcon(), func() {
		showToolsScreen()
	})

	deviceInfo := canvas.NewText(state.CurrentDevice.Model+" ("+state.CurrentDevice.Serial+")", color.RGBA{R: 160, G: 160, B: 170, A: 255})
	deviceInfo.TextSize = 11

	headerCenter := container.NewVBox(
		container.NewCenter(widget.NewLabelWithStyle("Análise de Armazenamento", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		container.NewCenter(deviceInfo),
	)
	header := container.NewBorder(nil, nil, backBtn, nil, headerCenter)

	// Inicia com um placeholder do gráfico
	emptyRaster := buildPieChartRaster(nil, 220)
	chartContainer.Objects = []fyne.CanvasObject{emptyRaster}

	chartAndLegend := container.NewVBox(
		chartContainer,
		container.NewCenter(summaryLabel),
		widget.NewSeparator(),
		container.NewCenter(legendContainer),
	)

	scrollContent := container.NewVScroll(chartAndLegend)
	scrollContent.SetMinSize(fyne.NewSize(450, 420))

	mainCard := container.NewVBox(
		btnScan,
		statusLabel,
		progressContainer,
		widget.NewSeparator(),
	)

	body := container.NewBorder(mainCard, nil, nil, nil, scrollContent)
	content := container.NewPadded(container.NewBorder(header, nil, nil, nil, body))

	state.Window.SetContent(content)

	// Dispara varredura automática ao entrar na tela
	go runScan()
}

// buildPieChartRaster desenha um gráfico de pizza estilo Donut nítido e anti-aliased
func buildPieChartRaster(report *StorageReport, dimension int) *canvas.Raster {
	var angleSlices []pieAngleSlice

	if report != nil && len(report.Slices) > 0 && report.TotalBytes > 0 {
		currentAngle := 0.0
		for _, s := range report.Slices {
			if s.Bytes <= 0 {
				continue
			}
			sliceAngle := (float64(s.Bytes) / float64(report.TotalBytes)) * 2.0 * math.Pi
			angleSlices = append(angleSlices, pieAngleSlice{
				startAngle: currentAngle,
				endAngle:   currentAngle + sliceAngle,
				color:      s.Color,
			})
			currentAngle += sliceAngle
		}
	}

	raster := canvas.NewRaster(func(w, h int) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, w, h))

		cx := float64(w) / 2.0
		cy := float64(h) / 2.0
		radius := math.Min(cx, cy) - 8.0
		if radius < 10 {
			radius = 10
		}
		innerRadius := radius * 0.42

		innerR2 := innerRadius * innerRadius
		outerR2 := radius * radius

		// Se não houver dados, desenha um anel cinza de placeholder
		if len(angleSlices) == 0 {
			placeholderCol := color.RGBA{R: 50, G: 50, B: 55, A: 255}
			for y := 0; y < h; y++ {
				dy := float64(y) - cy
				for x := 0; x < w; x++ {
					dx := float64(x) - cx
					d2 := dx*dx + dy*dy
					if d2 >= innerR2 && d2 <= outerR2 {
						img.SetRGBA(x, y, placeholderCol)
					}
				}
			}
			return img
		}

		for y := 0; y < h; y++ {
			dy := float64(y) - cy
			for x := 0; x < w; x++ {
				dx := float64(x) - cx
				d2 := dx*dx + dy*dy

				if d2 >= innerR2 && d2 <= outerR2 {
					// Ângulo em radianos normalizado de 0 a 2*pi
					angle := math.Atan2(dy, dx)
					if angle < 0 {
						angle += 2.0 * math.Pi
					}

					// Localiza a fatia correspondente
					var sliceCol color.RGBA = color.RGBA{R: 60, G: 60, B: 60, A: 255}
					for _, as := range angleSlices {
						if angle >= as.startAngle && angle <= as.endAngle {
							sliceCol = as.color
							break
						}
					}

					// Anti-aliasing suave nas bordas externa e interna
					dist := math.Sqrt(d2)
					alpha := 255.0
					if dist > radius-1.0 {
						alpha = 255.0 * (radius - dist)
					} else if dist < innerRadius+1.0 {
						alpha = 255.0 * (dist - innerRadius)
					}
					if alpha > 255.0 {
						alpha = 255.0
					}
					if alpha < 0.0 {
						alpha = 0.0
					}

					if alpha > 0 {
						img.SetRGBA(x, y, color.RGBA{
							R: uint8(float64(sliceCol.R) * (alpha / 255.0)),
							G: uint8(float64(sliceCol.G) * (alpha / 255.0)),
							B: uint8(float64(sliceCol.B) * (alpha / 255.0)),
							A: uint8(alpha),
						})
					}
				}
			}
		}

		return img
	})

	raster.SetMinSize(fyne.NewSize(float32(dimension), float32(dimension)))
	return raster
}

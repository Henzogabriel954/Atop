package main

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// PieChartRaster desenha um gráfico de pizza vetorial limpo e minimalista
type PieChartRaster struct {
	widget.BaseWidget
	categories []StorageCategory
	raster     *canvas.Raster
}

func NewPieChartRaster(categories []StorageCategory) *PieChartRaster {
	p := &PieChartRaster{
		categories: categories,
	}
	p.raster = canvas.NewRaster(p.draw)
	p.ExtendBaseWidget(p)
	return p
}

func (p *PieChartRaster) SetCategories(cats []StorageCategory) {
	p.categories = cats
	p.Refresh()
}

func (p *PieChartRaster) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.raster)
}

func (p *PieChartRaster) draw(w, h int) image_Image {
	img := newNRGBA(w, h)

	if len(p.categories) == 0 || w <= 0 || h <= 0 {
		return img
	}

	var totalBytes int64
	for _, c := range p.categories {
		totalBytes += c.Bytes
	}
	if totalBytes == 0 {
		return img
	}

	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	outerRadius := math.Min(cx, cy) - 4.0
	innerRadius := outerRadius * 0.52 // Estilo Donut elegante e minimalista

	// Calcula fatias com ângulos (em radianos)
	type slice struct {
		startAngle float64
		endAngle   float64
		col        color.NRGBA
	}
	slices := make([]slice, 0, len(p.categories))

	currentAngle := -math.Pi / 2 // Inicia no topo (12 horas)
	for _, cat := range p.categories {
		fraction := float64(cat.Bytes) / float64(totalBytes)
		sweep := fraction * 2.0 * math.Pi
		nrgba, ok := cat.Color.(color.NRGBA)
		if !ok {
			nrgba = color.NRGBA{R: 120, G: 120, B: 120, A: 255}
		}
		slices = append(slices, slice{
			startAngle: currentAngle,
			endAngle:   currentAngle + sweep,
			col:        nrgba,
		})
		currentAngle += sweep
	}

	for y := 0; y < h; y++ {
		dy := float64(y) - cy
		for x := 0; x < w; x++ {
			dx := float64(x) - cx
			dist := math.Hypot(dx, dy)

			if dist >= innerRadius && dist <= outerRadius {
				angle := math.Atan2(dy, dx)
				if angle < -math.Pi/2 {
					angle += 2.0 * math.Pi
				}

				for _, s := range slices {
					inSlice := false
					if s.endAngle <= math.Pi*1.5 {
						if angle >= s.startAngle && angle < s.endAngle {
							inSlice = true
						}
					} else {
						// Casos de quebra de ciclo
						if angle >= s.startAngle || angle < (s.endAngle-2.0*math.Pi) {
							inSlice = true
						}
					}

					if inSlice {
						img.SetNRGBA(x, y, s.col)
						break
					}
				}
			}
		}
	}

	return img
}

type image_Image interface {
	ColorModel() color.Model
	Bounds() image_Rectangle
	At(x, y int) color.Color
}

type image_Rectangle struct {
	Min, Max struct{ X, Y int }
}

func (r image_Rectangle) Dx() int { return r.Max.X - r.Min.X }
func (r image_Rectangle) Dy() int { return r.Max.Y - r.Min.Y }

type customNRGBA struct {
	Pix    []uint8
	Stride int
	Rect   image_Rectangle
}

func newNRGBA(w, h int) *customNRGBA {
	return &customNRGBA{
		Pix:    make([]uint8, 4*w*h),
		Stride: 4 * w,
		Rect:   image_Rectangle{Max: struct{ X, Y int }{w, h}},
	}
}

func (p *customNRGBA) ColorModel() color.Model { return color.NRGBAModel }
func (p *customNRGBA) Bounds() image_Rectangle { return p.Rect }
func (p *customNRGBA) At(x, y int) color.Color {
	i := y*p.Stride + x*4
	return color.NRGBA{R: p.Pix[i], G: p.Pix[i+1], B: p.Pix[i+2], A: p.Pix[i+3]}
}
func (p *customNRGBA) SetNRGBA(x, y int, c color.NRGBA) {
	i := y*p.Stride + x*4
	p.Pix[i] = c.R
	p.Pix[i+1] = c.G
	p.Pix[i+2] = c.B
	p.Pix[i+3] = c.A
}

func ShowStorageAnalysisDialog(parent fyne.Window, state *AppState) {
	d := fyne.CurrentApp().NewWindow("Análise de Armazenamento - Atop")
	d.Resize(fyne.NewSize(500, 580))

	serial := state.DeviceSerial
	if serial == "" {
		d.SetContent(container.NewPadded(
			container.NewVBox(
				widget.NewLabelWithStyle("[!] Nenhum dispositivo conectado", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Conecte um dispositivo Android via USB ou Wi-Fi antes de analisar o armazenamento."),
				widget.NewButton("Fechar", func() { d.Close() }),
			),
		))
		d.Show()
		return
	}

	header := widget.NewLabelWithStyle("[•] Análise de Armazenamento Interno", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	deviceLabel := widget.NewLabelWithStyle("Dispositivo: "+serial, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	statusProgress := widget.NewProgressBarInfinite()
	statusText := widget.NewLabel("Iniciando varredura rápida de diretórios...")

	chartContainer := container.NewCenter()
	legendBox := container.NewVBox()

	scanBtn := widget.NewButton("Escanear Novamente", nil)

	var runScan func()
	runScan = func() {
		statusProgress.Show()
		statusText.SetText("Examinando pastas (/sdcard/DCIM, Download, Android, etc)...")
		scanBtn.Disable()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			report, err := ScanStorageDetailed(ctx, serial)

			time.Sleep(200 * time.Millisecond) // Suaviza a transição
			statusProgress.Hide()
			scanBtn.Enable()

			if err != nil {
				statusText.SetText("[!] Falha ao analisar: " + err.Error())
				return
			}

			statusText.SetText(fmt.Sprintf("[+] Varredura concluída. Total analisado: %s", formatBytes(report.TotalUsed)))

			pie := NewPieChartRaster(report.Categories)
			pieContainer := container.NewGridWrap(fyne.NewSize(200, 200), pie)
			chartContainer.Objects = []fyne.CanvasObject{pieContainer}
			chartContainer.Refresh()

			legendBox.Objects = nil
			for _, cat := range report.Categories {
				pct := 0.0
				if report.TotalUsed > 0 {
					pct = float64(cat.Bytes) / float64(report.TotalUsed) * 100
				}
				row := container.NewHBox(
					widget.NewLabel(fmt.Sprintf("■ %s:", cat.Name)),
					widget.NewLabelWithStyle(fmt.Sprintf("%s (%.1f%%)", formatBytes(cat.Bytes), pct), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				)
				legendBox.Add(row)
			}
			legendBox.Refresh()
		}()
	}

	scanBtn.OnTapped = runScan

	body := container.NewVBox(
		header,
		deviceLabel,
		widget.NewSeparator(),
		statusProgress,
		statusText,
		chartContainer,
		legendBox,
		widget.NewSeparator(),
		container.NewHBox(
			scanBtn,
			widget.NewButton("Fechar", func() { d.Close() }),
		),
	)

	d.SetContent(container.NewPadded(body))
	d.Show()

	runScan()
}

package main

import (
	"context"
	"image/color"
	"sync"
	"sync/atomic"
)

type AppState struct {
	mu           sync.RWMutex
	monitoring   atomic.Bool
	DeviceSerial string
	DeviceIP     string
	DevicePort   string
	DeviceModel  string
	BatteryLevel string
	BatteryTemp  string
	CPUUsage     float64
	RAMUsage     float64
	StorageUsage float64
	FPS          float64
	NetworkPing  string
	Logs         []string
	ThemeMode    string // "dark", "light"
	ColorPalette string // "blue", "green", "purple", "orange"

	StopChan   chan struct{}
	CancelScan context.CancelFunc

	// Callbacks para UI
	OnMetricsUpdated func()
	OnLogAdded       func(log string)
	OnDeviceChanged  func(serial string)
}

func NewAppState() *AppState {
	return &AppState{
		DeviceSerial: "",
		DevicePort:   "5555",
		ThemeMode:    "dark",
		ColorPalette: "blue",
		Logs:         make([]string, 0, 100),
		StopChan:     make(chan struct{}),
	}
}

func (s *AppState) SetMonitoring(val bool) {
	s.monitoring.Store(val)
}

func (s *AppState) IsMonitoring() bool {
	return s.monitoring.Load()
}

func (s *AppState) AddLog(msg string) {
	s.mu.Lock()
	if len(s.Logs) >= 100 {
		s.Logs = s.Logs[1:]
	}
	s.Logs = append(s.Logs, msg)
	callback := s.OnLogAdded
	s.mu.Unlock()

	if callback != nil {
		callback(msg)
	}
}

func (s *AppState) SetDevice(serial, model string) {
	s.mu.Lock()
	s.DeviceSerial = serial
	s.DeviceModel = model
	callback := s.OnDeviceChanged
	s.mu.Unlock()

	if callback != nil {
		callback(serial)
	}
}

func (s *AppState) FetchMetrics() {
	if s.DeviceSerial == "" {
		return
	}

	metrics := FetchDeviceMetrics(s.DeviceSerial)

	s.mu.Lock()
	s.BatteryLevel = metrics.BatteryLevel
	s.BatteryTemp = metrics.BatteryTemp
	s.CPUUsage = metrics.CPUUsage
	s.RAMUsage = metrics.RAMUsage
	s.FPS = metrics.FPS
	s.StorageUsage = metrics.StorageUsage
	s.NetworkPing = metrics.Ping
	callback := s.OnMetricsUpdated
	s.mu.Unlock()

	if callback != nil {
		callback()
	}
}

func (s *AppState) Cleanup() {
	s.SetMonitoring(false)
	s.mu.Lock()
	if s.CancelScan != nil {
		s.CancelScan()
	}
	s.mu.Unlock()
	select {
	case <-s.StopChan:
	default:
		close(s.StopChan)
	}
}

func (s *AppState) GetAccentColor() color.Color {
	s.mu.RLock()
	palette := s.ColorPalette
	s.mu.RUnlock()

	switch palette {
	case "green":
		return color.NRGBA{R: 76, G: 175, B: 80, A: 255}
	case "purple":
		return color.NRGBA{R: 156, G: 39, B: 176, A: 255}
	case "orange":
		return color.NRGBA{R: 255, G: 152, B: 0, A: 255}
	default:
		return color.NRGBA{R: 66, G: 133, B: 244, A: 255}
	}
}

package main

import (
	"context"
	"image/color"
	"os/exec"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

type ScrcpyMode int

const (
	ModeOff ScrcpyMode = iota
	ModeGhost
	ModeMouse
	ModeMirror
)

// Chaves de preferências persistidas
const (
	PrefWifiAutoConnect = "wifi_auto_connect"
	PrefFixPort         = "fix_port_on_connect"
	PrefRefreshInterval = "refresh_interval_ms"
)

type Device struct {
	Serial         string
	Model          string
	IsUnauthorized bool
}

type ProcessInfo struct {
	PID  string
	Name string
	CPU  string
	RAM  string
}

type DeviceInfo struct {
	BatteryLevel    int
	BatteryCharging bool
	Temperature     float64 // Celsius
	AndroidVersion  string
	Uptime          string
}

// Estruturas para análise de armazenamento e gráfico de pizza
type StorageSlice struct {
	Name       string
	Path       string
	Bytes      int64
	Formatted  string
	Percentage float64
	Color      color.RGBA
}

type StorageReport struct {
	TotalBytes int64
	UsedBytes  int64
	FreeBytes  int64
	Slices     []StorageSlice
}

type AppState struct {
	App           fyne.App
	Window        fyne.Window
	CurrentDevice Device
	ScrcpyCmd     *exec.Cmd
	IsMonitoring  bool
	CurrentMode   ScrcpyMode

	// Controle de ciclo de vida seguro por contexto
	MonitorCancelCtx context.CancelFunc
	DeviceListCancel context.CancelFunc
	LifecycleMutex   sync.Mutex

	// Bindings de monitoramento
	CpuPercent          binding.Float
	RamTotal            binding.Float
	RamUsed             binding.Float
	SwapTotal           binding.Float
	SwapUsed            binding.Float
	BatteryPercent      binding.Float
	Temperature         binding.Float
	TableRefresher      binding.Bool
	DeviceListRefresher binding.Bool

	// Gerenciamento de ciclo de vida de listeners (elimina vazamentos)
	MonitorListeners   []binding.DataListener
	DeviceListListener binding.DataListener

	// Processos
	ProcessList []ProcessInfo
	TableMutex  sync.Mutex

	// mDNS
	MdnsMap     map[string]string
	MdnsMutex   sync.Mutex
	MdnsRunning bool
	MdnsCancel  context.CancelFunc

	// Gravação de tela
	RecordingCmd   *exec.Cmd
	RecordingMutex sync.Mutex
	IsRecording    bool

	// Info do dispositivo (cache)
	DeviceInfoCache DeviceInfo
	DeviceInfoMutex sync.Mutex

	// Configuração e notificação de refresh
	RefreshInterval       time.Duration
	RefreshIntervalUpdate chan time.Duration
}

var state *AppState

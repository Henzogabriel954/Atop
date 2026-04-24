package main

import (
	"os/exec"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type ScrcpyMode int

const (
	ModeOff ScrcpyMode = iota
	ModeGhost
	ModeMouse
	ModeMirror
)

type Device struct {
	Serial string
	Model  string
}

type ProcessInfo struct {
	PID  string
	Name string
	CPU  string
	RAM  string
}

type AppState struct {
	App                 fyne.App
	Window              fyne.Window
	CurrentDevice       Device
	ScrcpyCmd           *exec.Cmd
	IsMonitoring        bool
	MonitorStopChk      chan bool
	DeviceListStopChk   chan bool
	CurrentMode         ScrcpyMode
	CpuPercent          binding.Float
	RamTotal            binding.Float
	RamUsed             binding.Float
	SwapTotal           binding.Float
	SwapUsed            binding.Float
	TableRefresher      binding.Bool
	DeviceListRefresher binding.Bool
	ProcessList         []ProcessInfo
	TableWidget         *widget.Table
	TableMutex          sync.Mutex
	MdnsMap             map[string]string
	MdnsMutex           sync.Mutex
}

var state *AppState

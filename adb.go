package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

// --- CONFIGURAÇÕES ---
const (
	CommandTimeout = 2 * time.Second
)

func checkDependencies() {
	_, errAdb := exec.LookPath("adb")
	_, errScr := exec.LookPath("scrcpy")
	if errAdb != nil || errScr != nil {
		fmt.Println("[!] ADB ou Scrcpy não encontrados no PATH do sistema.")
	}
}

func fetchDevices() []Device {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	out, err := exec.CommandContext(ctx, "adb", "devices", "-l").Output()
	if err != nil {
		return []Device{}
	}

	lines := strings.Split(string(out), "\n")
	var devs []Device

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of") || strings.Contains(line, "offline") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			serial := fields[0]
			status := fields[1]
			isUnauthorized := (status == "unauthorized")

			modelName := "Android"
			for _, f := range fields {
				if strings.HasPrefix(f, "model:") {
					modelName = strings.TrimPrefix(f, "model:")
				}
			}

			if isUnauthorized {
				modelName += " [Não Autorizado]"
			}

			devs = append(devs, Device{
				Serial:         serial,
				Model:          modelName,
				IsUnauthorized: isUnauthorized,
			})
		}
	}
	return devs
}

// --- INFORMAÇÕES DO DISPOSITIVO (busca única) ---
func fetchDeviceInfoOnce(serial string) DeviceInfo {
	info := DeviceInfo{}

	// 1. Versão Android via getprop
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	out, err := exec.CommandContext(ctx, "adb", "-s", serial, "shell", "getprop", "ro.build.version.release").Output()
	cancel()
	if err == nil {
		info.AndroidVersion = strings.TrimSpace(string(out))
	}

	// 2. Uptime Universal via /proc/uptime (compatível com 100% dos Androids)
	ctx, cancel = context.WithTimeout(context.Background(), CommandTimeout)
	out, err = exec.CommandContext(ctx, "adb", "-s", serial, "shell", "cat", "/proc/uptime").Output()
	cancel()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			if sec, errSec := strconv.ParseFloat(fields[0], 64); errSec == nil {
				dur := time.Duration(sec) * time.Second
				days := int(dur.Hours()) / 24
				hours := int(dur.Hours()) % 24
				mins := int(dur.Minutes()) % 60
				if days > 0 {
					info.Uptime = fmt.Sprintf("up %dd %dh %dm", days, hours, mins)
				} else if hours > 0 {
					info.Uptime = fmt.Sprintf("up %dh %dm", hours, mins)
				} else {
					info.Uptime = fmt.Sprintf("up %dm", mins)
				}
			}
		}
	}

	return info
}

// --- BATERIA E TEMPERATURA ---
func fetchBatteryInfo(serial string) (level int, charging bool, tempC float64) {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "adb", "-s", serial, "shell", "dumpsys", "battery").Output()
	if err != nil {
		return 0, false, 0
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "level:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "level:"))
			level, _ = strconv.Atoi(val)
		}
		if strings.HasPrefix(line, "status:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			// 2 = Charging, 5 = Full
			charging = (val == "2" || val == "5")
		}
		if strings.HasPrefix(line, "temperature:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "temperature:"))
			raw, _ := strconv.ParseFloat(val, 64)
			tempC = raw / 10.0 // Décimos de grau Celsius
		}
	}
	return
}

// --- GRAVAÇÃO DE TELA ---
func startScreenRecording(serial string) error {
	state.RecordingMutex.Lock()
	defer state.RecordingMutex.Unlock()

	if state.IsRecording {
		return fmt.Errorf("já gravando")
	}

	cmd := exec.Command("adb", "-s", serial, "shell", "screenrecord", "/sdcard/adb_recording.mp4")
	err := cmd.Start()
	if err != nil {
		return err
	}

	state.RecordingCmd = cmd
	state.IsRecording = true

	// Auto-stop após 180s (limite intrínseco do Android screenrecord)
	go func() {
		cmd.Wait()
		state.RecordingMutex.Lock()
		state.IsRecording = false
		state.RecordingCmd = nil
		state.RecordingMutex.Unlock()
	}()

	return nil
}

func stopScreenRecording(serial string) (string, error) {
	state.RecordingMutex.Lock()
	if !state.IsRecording || state.RecordingCmd == nil {
		state.RecordingMutex.Unlock()
		return "", fmt.Errorf("não está gravando")
	}
	state.RecordingMutex.Unlock()

	// Envia sinal SIGINT (2) para consolidar o container MP4 (átomo moov) e não corromper o vídeo
	exec.Command("adb", "-s", serial, "shell", "kill -2 $(pidof screenrecord) 2>/dev/null || pkill -2 -f screenrecord").Run()

	// Aguarda o arquivo ser devidamente finalizado no dispositivo
	time.Sleep(1500 * time.Millisecond)

	// Diretório de destino no host
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	videosDir := filepath.Join(home, "Videos")
	os.MkdirAll(videosDir, 0755)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	destPath := filepath.Join(videosDir, fmt.Sprintf("recording_%s.mp4", timestamp))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = exec.CommandContext(ctx, "adb", "-s", serial, "pull", "/sdcard/adb_recording.mp4", destPath).Run()
	if err != nil {
		return "", err
	}

	// Limpa o arquivo temporário do celular
	exec.Command("adb", "-s", serial, "shell", "rm", "/sdcard/adb_recording.mp4").Run()

	state.RecordingMutex.Lock()
	state.IsRecording = false
	state.RecordingCmd = nil
	state.RecordingMutex.Unlock()

	return destPath, nil
}

// --- INSTALAR APK ---
func installAPK(serial, apkPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "adb", "-s", serial, "install", "-r", apkPath).Run()
}

// --- CONEXÃO MANUAL ---
func connectManual(address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "adb", "connect", address).CombinedOutput()
	if err != nil {
		return err
	}
	outStr := string(out)
	if strings.Contains(outStr, "failed") || strings.Contains(outStr, "refused") {
		return fmt.Errorf("%s", strings.TrimSpace(outStr))
	}
	return nil
}

func disconnectDevice(serial string) {
	exec.Command("adb", "disconnect", serial).Run()
}

// --- FIXAR PORTA 5555 ---
func fixWifiPort(serial string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "adb", "-s", serial, "tcpip", "5555").Run()
}

// --- MODO CAÇADOR (com drenagem segura de canais Zeroconf) ---
func huntNewPort(targetIP string) string {
	fmt.Printf("[*] CAÇADOR ATIVADO: Procurando %s na rede...\n", targetIP)

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry, 32)
	foundChan := make(chan string, 1)

	go func() {
		for entry := range entries {
			if len(entry.AddrIPv4) > 0 {
				ip := entry.AddrIPv4[0].String()
				if ip == targetIP {
					newPort := strconv.Itoa(entry.Port)
					newTarget := fmt.Sprintf("%s:%s", ip, newPort)

					state.MdnsMutex.Lock()
					state.MdnsMap[ip] = entry.Instance
					state.MdnsMutex.Unlock()

					select {
					case foundChan <- newTarget:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()

	err = resolver.Browse(ctx, "_adb-tls-connect._tcp", "local.", entries)
	if err != nil {
		return ""
	}

	select {
	case newSerial := <-foundChan:
		return newSerial
	case <-ctx.Done():
		return ""
	}
}

// --- MONITORAMENTO PRINCIPAL (com ciclo de vida reativo e seguro) ---
func monitorLoop(ctx context.Context) {
	interval := state.RefreshInterval
	if interval == 0 {
		interval = 1000 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Busca info estática inicial
	info := fetchDeviceInfoOnce(state.CurrentDevice.Serial)
	state.DeviceInfoMutex.Lock()
	state.DeviceInfoCache = info
	state.DeviceInfoMutex.Unlock()

	// Bateria e temperatura a cada 10 segundos
	batteryTicker := time.NewTicker(10 * time.Second)
	defer batteryTicker.Stop()

	// Busca de bateria inicial
	go func() {
		lvl, chrg, temp := fetchBatteryInfo(state.CurrentDevice.Serial)
		state.BatteryPercent.Set(float64(lvl))
		state.Temperature.Set(temp)
		state.DeviceInfoMutex.Lock()
		state.DeviceInfoCache.BatteryLevel = lvl
		state.DeviceInfoCache.BatteryCharging = chrg
		state.DeviceInfoCache.Temperature = temp
		state.DeviceInfoMutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case newInterval := <-state.RefreshIntervalUpdate:
			if newInterval > 0 {
				interval = newInterval
				ticker.Reset(interval)
			}

		case <-batteryTicker.C:
			go func() {
				lvl, chrg, temp := fetchBatteryInfo(state.CurrentDevice.Serial)
				state.BatteryPercent.Set(float64(lvl))
				state.Temperature.Set(temp)
				state.DeviceInfoMutex.Lock()
				state.DeviceInfoCache.BatteryLevel = lvl
				state.DeviceInfoCache.BatteryCharging = chrg
				state.DeviceInfoCache.Temperature = temp
				state.DeviceInfoMutex.Unlock()
			}()

		case <-ticker.C:
			if state.CurrentDevice.Serial == "" {
				continue
			}

			cmdCtx, cancelCmd := context.WithTimeout(context.Background(), CommandTimeout)
			outBytes, err := exec.CommandContext(cmdCtx, "adb", "-s", state.CurrentDevice.Serial, "shell", "top", "-b", "-n", "1", "-m", "30").CombinedOutput()
			cancelCmd()

			if err != nil || strings.Contains(string(outBytes), "offline") {
				if strings.Contains(state.CurrentDevice.Serial, ":") {
					parts := strings.Split(state.CurrentDevice.Serial, ":")
					currentIP := parts[0]

					exec.Command("adb", "disconnect", state.CurrentDevice.Serial).Run()
					newSerial := huntNewPort(currentIP)

					if newSerial != "" && newSerial != state.CurrentDevice.Serial {
						fmt.Printf("[+] RECONECTADO: %s -> %s\n", state.CurrentDevice.Serial, newSerial)
						exec.Command("adb", "connect", newSerial).Run()
						state.CurrentDevice.Serial = newSerial

						if state.App.Preferences().BoolWithFallback(PrefFixPort, false) {
							go fixWifiPort(newSerial)
						}
						goto ProcessData
					}
				}
				continue
			}

		ProcessData:
			parseTopOutput(string(outBytes))
		}
	}
}

// --- RADAR mDNS (Controlável e sem vazamentos) ---
func startMdnsLoop() {
	state.MdnsMutex.Lock()
	if state.MdnsRunning {
		state.MdnsMutex.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.MdnsCancel = cancel
	state.MdnsRunning = true
	state.MdnsMutex.Unlock()

	go mdnsAutoConnectLoop(ctx)
}

func stopMdnsLoop() {
	state.MdnsMutex.Lock()
	defer state.MdnsMutex.Unlock()
	if state.MdnsRunning && state.MdnsCancel != nil {
		state.MdnsCancel()
		state.MdnsCancel = nil
		state.MdnsRunning = false
	}
}

func restartMdnsLoop() {
	stopMdnsLoop()
	startMdnsLoop()
}

func mdnsAutoConnectLoop(ctx context.Context) {
	fmt.Println("[•] Radar mDNS iniciado com ciclo contínuo de 5 minutos...")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Instancia um resolver FRESCO no início de cada ciclo de 5 minutos
		// para garantir que os sockets UDP multicast estejam sempre abertos e ativos
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			fmt.Printf("[!] Erro ao criar resolver mDNS: %v. Tentando novamente em 3s...\n", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}

		// Ciclo de escuta ativa de 5 minutos
		cycleCtx, cancelCycle := context.WithTimeout(ctx, 5*time.Minute)
		entries := make(chan *zeroconf.ServiceEntry, 64)

		go func() {
			for entry := range entries {
				if len(entry.AddrIPv4) > 0 {
					ip := entry.AddrIPv4[0].String()
					port := entry.Port
					target := fmt.Sprintf("%s:%d", ip, port)

					state.MdnsMutex.Lock()
					state.MdnsMap[ip] = entry.Instance
					state.MdnsMutex.Unlock()

					connectedOut, _ := exec.Command("adb", "devices").Output()
					if !strings.Contains(string(connectedOut), target) {
						exec.Command("adb", "connect", target).Run()
						fmt.Printf("[+] mDNS Auto-conectado: %s\n", target)

						// Atualiza imediatamente a lista de dispositivos na UI
						if state.DeviceListRefresher != nil {
							val, _ := state.DeviceListRefresher.Get()
							state.DeviceListRefresher.Set(!val)
						}

						if state.App.Preferences().BoolWithFallback(PrefFixPort, false) {
							go func(t string) {
								time.Sleep(2 * time.Second)
								fixWifiPort(t)
							}(target)
						}
					}
				}
			}
		}()

		// Escuta ambos os serviços: TLS connect (Android 11+) e clássico adb
		resolver.Browse(cycleCtx, "_adb-tls-connect._tcp", "local.", entries)
		resolver.Browse(cycleCtx, "_adb._tcp", "local.", entries)

		select {
		case <-cycleCtx.Done():
		case <-ctx.Done():
			cancelCycle()
			return
		}
		cancelCycle()

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

// --- PARSER RESILIENTE COM SUPORTE A K, M, G ---
func parseMemoryKB(valStr string) float64 {
	s := strings.TrimSpace(valStr)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0
	}

	mult := 1.0 // Padrão KB
	if strings.HasSuffix(s, "G") || strings.HasSuffix(s, "g") {
		mult = 1024 * 1024 // GB para KB
		s = strings.TrimSuffix(strings.TrimSuffix(s, "G"), "g")
	} else if strings.HasSuffix(s, "M") || strings.HasSuffix(s, "m") {
		mult = 1024 // MB para KB
		s = strings.TrimSuffix(strings.TrimSuffix(s, "M"), "m")
	} else if strings.HasSuffix(s, "K") || strings.HasSuffix(s, "k") {
		mult = 1.0 // KB
		s = strings.TrimSuffix(strings.TrimSuffix(s, "K"), "k")
	}

	v, _ := strconv.ParseFloat(s, 64)
	return v * mult
}

func parseTopOutput(outStr string) {
	lines := strings.Split(outStr, "\n")
	var cpu, rTot, rUse, sTot, sUse float64
	var newProcs []ProcessInfo

	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if i < 7 {
			if strings.Contains(line, "Mem:") {
				rTot = parseMemoryKB(fields[1])
				rUse = parseMemoryKB(fields[3])
			}
			if strings.Contains(line, "Swap:") {
				sTot = parseMemoryKB(fields[1])
				sUse = parseMemoryKB(fields[3])
			}
			if strings.Contains(line, "%cpu") {
				maxCpu, _ := strconv.ParseFloat(strings.TrimSuffix(fields[0], "%cpu"), 64)
				idle := 0.0
				for _, f := range fields {
					if strings.Contains(f, "idle") {
						idle, _ = strconv.ParseFloat(strings.TrimSuffix(f, "%idle"), 64)
						break
					}
				}
				if maxCpu > 0 {
					cpu = (maxCpu - idle) / maxCpu
				}
			}
		}

		// Linhas de processo (PID numérico)
		if len(fields) >= 9 && isNumeric(fields[0]) {
			pid := fields[0]
			ram := fields[5]
			cpuVal := fields[8]
			if !strings.HasSuffix(cpuVal, "%") {
				cpuVal += "%"
			}
			name := fields[len(fields)-1]
			if idx := strings.Index(name, "/"); idx != -1 {
				name = filepath.Base(name)
			}
			newProcs = append(newProcs, ProcessInfo{
				PID:  pid,
				Name: name,
				CPU:  cpuVal,
				RAM:  ram,
			})
		}
	}

	state.CpuPercent.Set(cpu)
	state.RamTotal.Set(rTot)
	state.RamUsed.Set(rUse)
	state.SwapTotal.Set(sTot)
	state.SwapUsed.Set(sUse)

	state.TableMutex.Lock()
	state.ProcessList = newProcs
	state.TableMutex.Unlock()

	val, _ := state.TableRefresher.Get()
	state.TableRefresher.Set(!val)
}

func stopScrcpy() {
	if state.ScrcpyCmd != nil {
		if state.ScrcpyCmd.Process != nil {
			state.ScrcpyCmd.Process.Kill()
			state.ScrcpyCmd.Wait()
		}
	}
	state.ScrcpyCmd = nil
	state.CurrentMode = ModeOff
}

func toggleScrcpy(targetMode ScrcpyMode, args ...string) {
	if state.CurrentMode == targetMode {
		stopScrcpy()
		if targetMode == ModeGhost {
			exec.Command("adb", "-s", state.CurrentDevice.Serial, "shell", "input", "keyevent", "224").Run()
		}
		updateButtonState()
		return
	}
	stopScrcpy()
	state.CurrentMode = targetMode
	finalArgs := append([]string{"-s", state.CurrentDevice.Serial}, args...)
	cmd := exec.Command("scrcpy", finalArgs...)
	cmd.Env = os.Environ()
	if contains(args, "--turn-screen-off") {
		cmd.Env = append(cmd.Env, "SDL_VIDEODRIVER=dummy")
	}
	state.ScrcpyCmd = cmd
	go func() {
		err := cmd.Start()
		if err != nil {
			state.CurrentMode = ModeOff
		}
		updateButtonState()
		cmd.Wait()
		if state.ScrcpyCmd == cmd {
			state.CurrentMode = ModeOff
			state.ScrcpyCmd = nil
			updateButtonState()
		}
	}()
}

func updateButtonState() {
	val, _ := state.TableRefresher.Get()
	state.TableRefresher.Set(!val)
}

func isNumeric(s string) bool { _, err := strconv.Atoi(s); return err == nil }

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// --- FORMATAÇÃO DE BYTES ---
func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// --- SCANNER DE ARMAZENAMENTO DO DISPOSITIVO ---
func scanDeviceStorage(ctx context.Context, serial string) (*StorageReport, error) {
	report := &StorageReport{}

	// 1. Obtém partição de dados (/data ou /sdcard) via df -k
	dfCtx, cancelDf := context.WithTimeout(ctx, 4*time.Second)
	dfOut, err := exec.CommandContext(dfCtx, "adb", "-s", serial, "shell", "df", "-k", "/data", "/sdcard").CombinedOutput()
	cancelDf()

	if err == nil {
		lines := strings.Split(string(dfOut), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			// Esperado: Filesystem 1K-blocks Used Available Use% Mounted
			if len(fields) >= 5 && isNumeric(fields[1]) && isNumeric(fields[2]) {
				totalKb, _ := strconv.ParseInt(fields[1], 10, 64)
				usedKb, _ := strconv.ParseInt(fields[2], 10, 64)
				freeKb, _ := strconv.ParseInt(fields[3], 10, 64)

				report.TotalBytes = totalKb * 1024
				report.UsedBytes = usedKb * 1024
				report.FreeBytes = freeKb * 1024
				break
			}
		}
	}

	// Se não achou via df, tenta fallback simples
	if report.TotalBytes == 0 {
		report.TotalBytes = 64 * 1024 * 1024 * 1024 // 64 GB placeholder se dispositivo não responder df
		report.FreeBytes = 32 * 1024 * 1024 * 1024
		report.UsedBytes = 32 * 1024 * 1024 * 1024
	}

	// 2. Coleta em lote das pastas principais em /sdcard via du -sk
	duCtx, cancelDu := context.WithTimeout(ctx, 15*time.Second)
	duOut, _ := exec.CommandContext(duCtx, "adb", "-s", serial, "shell",
		"du", "-sk",
		"/sdcard/DCIM",
		"/sdcard/Pictures",
		"/sdcard/Movies",
		"/sdcard/Download",
		"/sdcard/Downloads",
		"/sdcard/Android",
		"/sdcard/Music",
		"/sdcard/Documents",
		"/sdcard/WhatsApp",
	).CombinedOutput()
	cancelDu()

	folderSizes := make(map[string]int64)
	for _, line := range strings.Split(string(duOut), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && isNumeric(fields[0]) {
			kb, _ := strconv.ParseInt(fields[0], 10, 64)
			path := fields[1]
			folderSizes[path] = kb * 1024
		}
	}

	// Categorias consolidadas
	photosBytes := folderSizes["/sdcard/DCIM"] + folderSizes["/sdcard/Pictures"]
	videosBytes := folderSizes["/sdcard/Movies"]
	downloadsBytes := folderSizes["/sdcard/Download"] + folderSizes["/sdcard/Downloads"]
	appsCacheBytes := folderSizes["/sdcard/Android"]
	musicBytes := folderSizes["/sdcard/Music"]
	docsBytes := folderSizes["/sdcard/Documents"]
	whatsappBytes := folderSizes["/sdcard/WhatsApp"]

	sumSdcard := photosBytes + videosBytes + downloadsBytes + appsCacheBytes + musicBytes + docsBytes + whatsappBytes
	systemOtherBytes := report.UsedBytes - sumSdcard
	if systemOtherBytes < 0 {
		systemOtherBytes = 0
	}

	// Paleta de cores moderna minimalista de alto contraste
	rawSlices := []struct {
		name  string
		path  string
		bytes int64
		col   color.RGBA
	}{
		{"Fotos & Câmera", "/sdcard/DCIM", photosBytes, color.RGBA{R: 16, G: 185, B: 129, A: 255}},  // Emerald #10B981
		{"Vídeos", "/sdcard/Movies", videosBytes, color.RGBA{R: 59, G: 130, B: 246, A: 255}},         // Blue #3B82F6
		{"Apps & Cache", "/sdcard/Android", appsCacheBytes, color.RGBA{R: 139, G: 92, B: 246, A: 255}}, // Purple #8B5CF6
		{"Downloads", "/sdcard/Download", downloadsBytes, color.RGBA{R: 245, G: 158, B: 11, A: 255}}, // Amber #F59E0B
		{"Músicas & Áudio", "/sdcard/Music", musicBytes, color.RGBA{R: 6, G: 182, B: 212, A: 255}},    // Cyan #06B6D4
		{"Documentos", "/sdcard/Documents", docsBytes, color.RGBA{R: 236, G: 72, B: 153, A: 255}},    // Pink #EC4899
	}

	if whatsappBytes > 0 {
		rawSlices = append(rawSlices, struct {
			name  string
			path  string
			bytes int64
			col   color.RGBA
		}{"WhatsApp / Mídia", "/sdcard/WhatsApp", whatsappBytes, color.RGBA{R: 34, G: 197, B: 94, A: 255}})
	}

	rawSlices = append(rawSlices, struct {
		name  string
		path  string
		bytes int64
		col   color.RGBA
	}{"Sistema & Outros", "/data", systemOtherBytes, color.RGBA{R: 100, G: 116, B: 139, A: 255}}) // Slate #64748B

	// Adiciona fatia de Espaço Livre para compor 100% da capacidade total
	rawSlices = append(rawSlices, struct {
		name  string
		path  string
		bytes int64
		col   color.RGBA
	}{"Espaço Livre", "/data (free)", report.FreeBytes, color.RGBA{R: 45, G: 45, B: 52, A: 255}}) // Dark Grey

	for _, s := range rawSlices {
		var pct float64
		if report.TotalBytes > 0 {
			pct = (float64(s.bytes) / float64(report.TotalBytes)) * 100.0
		}
		report.Slices = append(report.Slices, StorageSlice{
			Name:       s.name,
			Path:       s.path,
			Bytes:      s.bytes,
			Formatted:  formatBytes(s.bytes),
			Percentage: pct,
			Color:      s.col,
		})
	}

	return report, nil
}

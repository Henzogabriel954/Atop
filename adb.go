package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

// --- CONFIGURAÇÕES ---
const (
	CommandTimeout = 2 * time.Second // Se o celular não responder em 2s, considera morto
)

func checkDependencies() {
	_, errAdb := exec.LookPath("adb")
	_, errScr := exec.LookPath("scrcpy")
	if errAdb != nil || errScr != nil {
		fmt.Println("⚠️ ADB ou Scrcpy não encontrados.")
	}
}

func fetchDevices() []Device {
	// Timeout curto para não travar a UI se o ADB estiver lento
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	out, err := exec.CommandContext(ctx, "adb", "devices", "-l").Output()
	if err != nil { return []Device{} }

	lines := strings.Split(string(out), "\n")
	var devs []Device

	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "List of") || strings.Contains(line, "offline") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			modelName := "Android"
			for _, f := range fields {
				if strings.HasPrefix(f, "model:") {
					modelName = strings.TrimPrefix(f, "model:")
				}
			}
			devs = append(devs, Device{Serial: fields[0], Model: modelName})
		}
	}
	return devs
}

// --- MODO CAÇADOR: Procura ativamente a nova porta de um IP ---
func huntNewPort(targetIP string) string {
	fmt.Printf("🕵️ CAÇADOR ATIVADO: Procurando %s na rede...\n", targetIP)
	
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil { return "" }

	entries := make(chan *zeroconf.ServiceEntry)
	foundChan := make(chan string)

	// Inicia o scanner
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if len(entry.AddrIPv4) > 0 {
				ip := entry.AddrIPv4[0].String()
				// Se achou o IP que estamos procurando
				if ip == targetIP {
					newPort := strconv.Itoa(entry.Port)
					newTarget := fmt.Sprintf("%s:%s", ip, newPort)
					
					// Salva o nome para a UI ficar bonita
					state.MdnsMutex.Lock()
					state.MdnsMap[ip] = entry.Instance
					state.MdnsMutex.Unlock()
					
					// Avisa que achou
					select {
					case foundChan <- newTarget:
					default:
					}
					return
				}
			}
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = resolver.Browse(ctx, "_adb-tls-connect._tcp", "local.", entries)
	
	select {
	case newSerial := <-foundChan:
		return newSerial
	case <-ctx.Done():
		return "" // Não achou em 5 segundos
	}
}

// --- MONITORAMENTO PRINCIPAL ---
func monitorLoop() {
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-state.MonitorStopChk:
			return
		case <-ticker.C:
			if state.CurrentDevice.Serial == "" { continue }

			// 1. EXECUÇÃO COM TIMEOUT RIGOROSO
			// Se o comando travar, o Context mata ele em 2 segundos
			ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
			outBytes, err := exec.CommandContext(ctx, "adb", "-s", state.CurrentDevice.Serial, "shell", "top", "-b", "-n", "1", "-m", "30").CombinedOutput()
			cancel()

			// 2. SE DEU MERDA (Erro, Timeout ou Offline)
			if err != nil || strings.Contains(string(outBytes), "offline") {
				fmt.Println("❌ Perda de conexão detectada!")

				// Se for conexão Wi-Fi (tem IP)
				if strings.Contains(state.CurrentDevice.Serial, ":") {
					parts := strings.Split(state.CurrentDevice.Serial, ":")
					currentIP := parts[0]

					// A. Mata a conexão velha imediatamente
					exec.Command("adb", "disconnect", state.CurrentDevice.Serial).Run()

					// B. Entra no modo Caçador para achar a porta nova
					newSerial := huntNewPort(currentIP)

					if newSerial != "" && newSerial != state.CurrentDevice.Serial {
						fmt.Printf("✅ RECONECTADO: %s -> %s\n", state.CurrentDevice.Serial, newSerial)
						
						// Conecta no novo
						exec.Command("adb", "connect", newSerial).Run()
						
						// Atualiza o estado global
						state.CurrentDevice.Serial = newSerial
						
						// Tenta de novo imediatamente
						goto ProcessData
					} else {
						fmt.Println("⏳ Celular ainda não apareceu na rede... tentando de novo.")
					}
				}
				continue
			}

		ProcessData:
			parseTopOutput(string(outBytes))
		}
	}
}

// --- RADAR GENÉRICO (Para quando não estamos monitorando nada) ---
func mdnsAutoConnectLoop() {
	fmt.Println("📡 Radar de fundo iniciado...")
	resolver, _ := zeroconf.NewResolver(nil)
	
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		entries := make(chan *zeroconf.ServiceEntry)

		go func(results <-chan *zeroconf.ServiceEntry) {
			for entry := range results {
				if len(entry.AddrIPv4) > 0 {
					ip := entry.AddrIPv4[0].String()
					port := entry.Port
					target := fmt.Sprintf("%s:%d", ip, port)

					// Guarda nome
					state.MdnsMutex.Lock()
					state.MdnsMap[ip] = entry.Instance
					state.MdnsMutex.Unlock()

					// Se não estiver conectado, conecta
					// (Não fazemos disconnect aqui para não brigar com o monitorLoop)
					connectedOut, _ := exec.Command("adb", "devices").Output()
					if !strings.Contains(string(connectedOut), target) {
						exec.Command("adb", "connect", target).Run()
					}
				}
			}
		}(entries)

		resolver.Browse(ctx, "_adb-tls-connect._tcp", "local.", entries)
		<-ctx.Done()
		cancel()
		time.Sleep(2 * time.Second)
	}
}

// --- PARSER E UTILS ---
func parseTopOutput(outStr string) {
	lines := strings.Split(outStr, "\n")
	var cpu, rTot, rUse, sTot, sUse float64
	var newProcs []ProcessInfo

	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 { continue }

		if i < 6 {
			if strings.Contains(line, "Mem:") {
				rTot = parseK(fields, 1)
				rUse = parseK(fields, 3)
			}
			if strings.Contains(line, "Swap:") {
				sTot = parseK(fields, 1)
				sUse = parseK(fields, 3)
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
				if maxCpu > 0 { cpu = ((maxCpu - idle) / maxCpu) }
			}
		}
		if len(fields) >= 10 && isNumeric(fields[0]) {
			pid := fields[0]
			ram := fields[5]
			cpuVal := fields[8] + "%"
			name := fields[len(fields)-1]
			if strings.Contains(name, ".") {
				parts := strings.Split(name, ".")
				name = parts[len(parts)-1]
			}
			newProcs = append(newProcs, ProcessInfo{pid, name, cpuVal, ram})
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
		if err != nil { state.CurrentMode = ModeOff }
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

func parseK(fields []string, idx int) float64 {
	if idx >= len(fields) { return 1 }
	s := strings.ReplaceAll(fields[idx], "K", "")
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func isNumeric(s string) bool { _, err := strconv.Atoi(s); return err == nil }
func contains(slice []string, item string) bool {
	for _, s := range slice { if s == item { return true } }
	return false
}

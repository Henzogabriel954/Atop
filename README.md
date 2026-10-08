# Atop - Android Telemetry & Control Manager

Aplicativo desktop ultraleve, fluido e minimalista construído em Go e Fyne v2 para gerenciamento avançado, telemetria em tempo real e controle de smartphones Android via ADB e Scrcpy.

---

## [•] Instalação Rápida (One-Liner)

Para compilar na hora e instalar automaticamente em seu sistema, execute:

```bash
curl -sSL https://raw.githubusercontent.com/Henzogabriel954/Atop/main/install.sh | bash
```

Após a conclusão da instalação, basta abrir o terminal e digitar:

```bash
atop
```

*(O instalador compila sob medida para sua arquitetura com `go build`, instala o binário em `~/.local/bin/atop` sem precisar de `sudo`, garante o `$PATH` e cria o atalho no menu de aplicativos do sistema).*

---

## [•] Funcionalidades Principais

1. **Telemetria de Sistema em Tempo Real:**
   - Consumo de CPU, RAM e SWAP com leitura universal e tolerante a unidades (KB, MB, GB).
   - Nível e status de bateria (com indicador de carregador `+`) e temperatura em graus Celsius.
   - Lista dinâmica de processos com consumo estilo `htop`.
   - Detecção de dispositivos não autorizados com aviso amigável de chave RSA.

2. **Radar Wi-Fi Automático (mDNS):**
   - Ciclo contínuo de escuta de **5 minutos** para descoberta e conexão sem fio automática.
   - Detecta o dispositivo mesmo se a depuração Wi-Fi for ativada após abrir o aplicativo.
   - Modo Caçador de portas para re-conexão autônoma em caso de perda de sinal ou reinicialização.
   - Botão de busca sob demanda `[Buscar Wi-Fi Agora]`.

3. **Analisador de Armazenamento com Gráfico de Pizza:**
   - Varredura em segundo plano via ADB das partições e diretórios do smartphone.
   - Gráfico de pizza (estilo Donut) nativo renderizado em memória com anti-aliasing.
   - Categorização detalhada: Fotos & Câmera, Vídeos, Apps & Cache, Downloads, Músicas, Documentos, WhatsApp, Sistema & Outros e Espaço Livre.
   - Legenda com cores semânticas, tamanhos em GB e percentuais.

4. **Controle Integrado do Scrcpy:**
   - `◈ Espelho`: Espelhamento de tela completo.
   - `◇ Ghost`: Controle em segundo plano com a tela do celular apagada (economia máxima de bateria).
   - `▸ Mouse`: Modo headless operando apenas com o cursor do mouse sem transmissão de vídeo/áudio.

5. **Ferramentas Integradas:**
   - Gravação de tela do dispositivo com sinal de encerramento limpo (`SIGINT`), preservando os cabeçalhos MP4 sem corromper arquivos.
   - Instalador rápido de pacotes APK com barra de progresso.

---

## [•] Atalhos de Teclado no Monitor

| Tecla | Ação |
| :---: | :--- |
| `S` | Ativar / Desativar modo Ghost (tela apagada) |
| `M` | Ativar / Desativar modo Mouse |
| `E` | Ativar / Desativar modo Espelho (Scrcpy) |
| `R` | Reiniciar o smartphone (com confirmação) |
| `L` | Ligar / Desligar tela (botão Power) |
| `T` | Abrir tela de Ferramentas |
| `Q` | Voltar à lista de dispositivos |

---

## [•] Requisitos do Sistema

- **Linux** (x86_64 ou ARM64)
- **Go 1.18+** (apenas para compilação via `install.sh`)
- **ADB** (`sudo apt install adb`)
- **Scrcpy** (`sudo apt install scrcpy`)

---

## [•] Compilação Manual

Se preferir compilar manualmente:

```bash
git clone https://github.com/Henzogabriel954/Atop.git
cd Atop
go build -ldflags="-s -w" -o atop .
./atop
```

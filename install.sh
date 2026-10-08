#!/usr/bin/env bash
set -e

# Cores e estilos
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # Sem Cor

echo -e "${BLUE}${BOLD}====================================================${NC}"
echo -e "${BLUE}${BOLD}   [•] Atop - Instalador Automático de Compilação  ${NC}"
echo -e "${BLUE}${BOLD}====================================================${NC}"

# 1. Verifica dependências básicas
if ! command -v git >/dev/null 2>&1; then
    echo -e "${RED}[!] Erro: 'git' não encontrado no sistema.${NC}"
    echo -e "${YELLOW}    Instale com: sudo apt update && sudo apt install -y git${NC}"
    exit 1
fi

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}[!] Erro: 'go' (Golang) não encontrado no sistema.${NC}"
    echo -e "${YELLOW}    Instale com: sudo apt update && sudo apt install -y golang${NC}"
    echo -e "${YELLOW}    Ou baixe a versão mais recente em: https://go.dev/dl/${NC}"
    exit 1
fi

# Avisos opcionais sobre ADB e Scrcpy
if ! command -v adb >/dev/null 2>&1; then
    echo -e "${YELLOW}[!] Aviso: 'adb' não foi detectado no PATH.${NC}"
    echo -e "${YELLOW}    Para utilizar o monitoramento, instale com: sudo apt install -y adb${NC}"
fi

if ! command -v scrcpy >/dev/null 2>&1; then
    echo -e "${YELLOW}[!] Aviso: 'scrcpy' não foi detectado no PATH.${NC}"
    echo -e "${YELLOW}    Para utilizar o espelhamento de tela, instale com: sudo apt install -y scrcpy${NC}"
fi

# 2. Cria diretório temporário isolado
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo -e "${BLUE}[•] Baixando a versão mais recente do código fonte...${NC}"
git clone --depth 1 https://github.com/Henzogabriel954/Atop.git "$TMP_DIR" >/dev/null 2>&1

cd "$TMP_DIR"

echo -e "${BLUE}[•] Compilando Atop sob medida para seu sistema (on-the-fly)...${NC}"
go build -ldflags="-s -w" -o atop .

# 3. Instalação em ~/.local/bin (não necessita de privilégios de root)
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"
cp atop "$INSTALL_DIR/atop"
chmod +x "$INSTALL_DIR/atop"

# 4. Garante que ~/.local/bin está presente no PATH do usuário
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    SHELL_PROFILE=""
    if [ -n "$BASH_VERSION" ]; then
        SHELL_PROFILE="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        SHELL_PROFILE="$HOME/.zshrc"
    elif [ -f "$HOME/.bashrc" ]; then
        SHELL_PROFILE="$HOME/.bashrc"
    elif [ -f "$HOME/.profile" ]; then
        SHELL_PROFILE="$HOME/.profile"
    fi

    if [ -n "$SHELL_PROFILE" ] && [ -f "$SHELL_PROFILE" ]; then
        if ! grep -q 'export PATH="$HOME/.local/bin:$PATH"' "$SHELL_PROFILE"; then
            echo '' >> "$SHELL_PROFILE"
            echo '# Atop path' >> "$SHELL_PROFILE"
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$SHELL_PROFILE"
            echo -e "${BLUE}[•] Configurado $INSTALL_DIR no seu $SHELL_PROFILE${NC}"
        fi
    fi
fi

# 5. Criação do lançador desktop no ambiente gráfico
DESKTOP_DIR="$HOME/.local/share/applications"
mkdir -p "$DESKTOP_DIR"
cat << 'EOF' > "$DESKTOP_DIR/atop.desktop"
[Desktop Entry]
Name=Atop
Comment=Gerenciador de Telemetria e Controle Android ADB
Exec=atop
Terminal=false
Type=Application
Categories=Development;Utility;
StartupNotify=true
EOF

echo ""
echo -e "${GREEN}${BOLD}[+] Instalação do Atop concluída com sucesso!${NC}"
echo -e "${GREEN}[+] O binário foi instalado em: ${BOLD}$INSTALL_DIR/atop${NC}"
echo -e "${BLUE}[•] Para abrir a aplicação agora, execute:${NC} ${GREEN}${BOLD}atop${NC}"
echo ""

#!/bin/bash
# =============================================================================
# OI - Script de Instalação Universal
# =============================================================================
# Instala a versão mais recente (ou especificada) do OI diretamente do GitHub.
#
# Uso:
#   curl -sSL https://raw.githubusercontent.com/MrJc01/crom-oi/main/scripts/install.sh | sudo bash
#   curl -sSL https://raw.githubusercontent.com/MrJc01/crom-oi/main/scripts/install.sh | sudo bash -s -- v1.0.0
#
# Opções:
#   Passar versão como argumento: ./install.sh v1.0.0
#   Sem argumento: instala a última versão estável
# =============================================================================

set -e

# =============================================================================
# CONFIGURAÇÃO (Altere aqui para seu fork)
# =============================================================================
REPO="MrJc01/crom-oi"
BINARY_NAME="oi"
INSTALL_DIR="/usr/local/bin"

# =============================================================================
# CORES E FORMATAÇÃO
# =============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info() { echo -e "${BLUE}ℹ️  $1${NC}" >&2; }
log_success() { echo -e "${GREEN}✅ $1${NC}" >&2; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}" >&2; }
log_error() { echo -e "${RED}❌ $1${NC}" >&2; }
log_step() { echo -e "${CYAN}→ $1${NC}" >&2; }

# =============================================================================
# FUNÇÕES DE DETECÇÃO
# =============================================================================

detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    
    case "$os" in
        linux)
            echo "linux"
            ;;
        darwin)
            log_error "macOS não é suportado ainda. Use Linux."
            exit 1
            ;;
        *)
            log_error "Sistema operacional não suportado: $os"
            exit 1
            ;;
    esac
}

detect_arch() {
    local arch
    arch=$(uname -m)
    
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armhf)
            log_warning "ARM 32-bit detectado. Tentando usar arm64..."
            echo "arm64"
            ;;
        *)
            log_error "Arquitetura não suportada: $arch"
            exit 1
            ;;
    esac
}

# =============================================================================
# FUNÇÕES DE DOWNLOAD
# =============================================================================

get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local version
    
    log_step "Obtendo última versão de ${REPO}..."
    
    if command -v curl &> /dev/null; then
        version=$(curl -sSL "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget &> /dev/null; then
        version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        log_error "Nem curl nem wget estão disponíveis. Instale um deles."
        exit 1
    fi
    
    if [[ -z "$version" ]]; then
        log_error "Não foi possível obter a última versão. Verifique se o repositório existe e tem releases."
        exit 1
    fi
    
    echo "$version"
}

download_binary() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local binary_name="${BINARY_NAME}-${os}-${arch}"
    local url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"
    local tmp_file="/tmp/${binary_name}"
    
    log_step "Baixando ${binary_name} (${version})..."
    log_info "URL: $url"
    
    if command -v curl &> /dev/null; then
        if ! curl -sSL --fail -o "$tmp_file" "$url"; then
            log_error "Falha ao baixar de $url"
            log_error "Verifique se a versão ${version} existe e tem binários para ${os}-${arch}"
            exit 1
        fi
    elif command -v wget &> /dev/null; then
        if ! wget -q -O "$tmp_file" "$url"; then
            log_error "Falha ao baixar de $url"
            exit 1
        fi
    else
        log_error "Nem curl nem wget estão disponíveis"
        exit 1
    fi
    
    echo "$tmp_file"
}

download_checksum() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local binary_name="${BINARY_NAME}-${os}-${arch}"
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}.sha256"
    local tmp_checksum="/tmp/${binary_name}.sha256"
    
    log_step "Baixando checksum..."
    
    if command -v curl &> /dev/null; then
        curl -sSL --fail -o "$tmp_checksum" "$checksum_url" 2>/dev/null || return 1
    elif command -v wget &> /dev/null; then
        wget -q -O "$tmp_checksum" "$checksum_url" 2>/dev/null || return 1
    fi
    
    echo "$tmp_checksum"
}

verify_checksum() {
    local binary_file="$1"
    local checksum_file="$2"
    
    if [[ ! -f "$checksum_file" ]]; then
        log_warning "Checksum não disponível, pulando verificação"
        return 0
    fi
    
    log_step "Verificando integridade do binário..."
    
    local expected_hash actual_hash
    expected_hash=$(cat "$checksum_file" | awk '{print $1}')
    actual_hash=$(sha256sum "$binary_file" | awk '{print $1}')
    
    if [[ "$expected_hash" == "$actual_hash" ]]; then
        log_success "Checksum verificado: ${actual_hash:0:16}..."
        return 0
    else
        log_error "Checksum não corresponde!"
        log_error "Esperado: $expected_hash"
        log_error "Obtido:   $actual_hash"
        exit 1
    fi
}

# =============================================================================
# FUNÇÕES DE INSTALAÇÃO
# =============================================================================

install_binary() {
    local binary_file="$1"
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"
    
    log_step "Instalando em ${install_path}..."
    
    # Backup se existir versão anterior
    if [[ -f "$install_path" ]]; then
        local backup="${install_path}.backup"
        log_info "Fazendo backup da versão anterior em ${backup}"
        mv "$install_path" "$backup"
    fi
    
    # Mover binário
    mv "$binary_file" "$install_path"
    chmod +x "$install_path"
    
    # Verificar instalação
    if "$install_path" --help &>/dev/null || "$install_path" version &>/dev/null || [[ -x "$install_path" ]]; then
        log_success "Binário instalado em ${install_path}"
    else
        log_warning "Binário instalado mas pode não estar funcionando corretamente"
    fi
}

# =============================================================================
# VERIFICAÇÃO DE DEPENDÊNCIAS
# =============================================================================

check_dependencies() {
    log_step "Verificando dependências..."
    
    local missing=()
    
    # Docker
    if command -v docker &> /dev/null; then
        log_success "Docker instalado: $(docker --version | head -1)"
    else
        missing+=("docker")
        log_warning "Docker não encontrado"
    fi
    
    # Docker daemon
    if docker info &>/dev/null 2>&1; then
        log_success "Docker daemon acessível"
    else
        log_warning "Docker daemon não acessível (execute: sudo usermod -aG docker \$USER && newgrp docker)"
    fi
    
    # Caddy (opcional)
    if command -v caddy &> /dev/null; then
        log_success "Caddy instalado: $(caddy version 2>/dev/null | head -1 || echo 'versão desconhecida')"
    else
        log_info "Caddy não instalado (opcional, use --no-caddy para deploy local)"
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo ""
        log_warning "Dependências faltando: ${missing[*]}"
        echo ""
        echo "Para instalar Docker:"
        echo "  curl -fsSL https://get.docker.com | sudo sh"
        echo "  sudo usermod -aG docker \$USER"
        echo "  newgrp docker"
        echo ""
    fi
}

# =============================================================================
# SETUP DO SYSTEMD (OPCIONAL)
# =============================================================================

setup_systemd_service() {
    local service_file="/etc/systemd/system/oi.service"
    
    if [[ -f "$service_file" ]]; then
        log_info "Serviço systemd já existe"
        return 0
    fi
    
    log_step "Deseja criar serviço systemd para o OI? (y/N)"
    read -r response
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        cat > "$service_file" << 'EOF'
[Unit]
Description=OI - Orquestrador de Intenção
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/oi up
ExecStop=/usr/local/bin/oi down
WorkingDirectory=/opt/oi
User=root

[Install]
WantedBy=multi-user.target
EOF
        
        mkdir -p /opt/oi
        systemctl daemon-reload
        log_success "Serviço systemd criado: oi.service"
        log_info "Para usar: systemctl enable --now oi"
    fi
}

# =============================================================================
# MAIN
# =============================================================================

main() {
    echo ""
    echo -e "${BOLD}${BLUE}=============================================="
    echo "  OI - Instalador Universal"
    echo "==============================================${NC}"
    echo ""
    
    # Verificar se é root
    if [[ $EUID -ne 0 ]]; then
        log_error "Este script precisa ser executado como root (use sudo)"
        exit 1
    fi
    
    # Determinar versão
    local version="${1:-}"
    if [[ -z "$version" ]]; then
        version=$(get_latest_version)
    fi
    log_info "Versão selecionada: ${version}"
    
    # Detectar sistema
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    log_info "Sistema detectado: ${os}-${arch}"
    
    echo ""
    
    # Download
    local binary_file checksum_file
    binary_file=$(download_binary "$version" "$os" "$arch")
    checksum_file=$(download_checksum "$version" "$os" "$arch") || true
    
    # Verificar checksum
    verify_checksum "$binary_file" "$checksum_file"
    
    # Instalar
    install_binary "$binary_file"
    
    echo ""
    
    # Verificar dependências
    check_dependencies
    
    # Setup systemd (interativo, apenas se terminal)
    if [[ -t 0 ]]; then
        setup_systemd_service
    fi
    
    # Limpar arquivos temporários
    rm -f "/tmp/${BINARY_NAME}-"* 2>/dev/null || true
    
    # Resultado final
    echo ""
    echo -e "${BOLD}${GREEN}=============================================="
    echo "  ✅ OI instalado com sucesso!"
    echo "==============================================${NC}"
    echo ""
    echo "Próximos passos:"
    echo "  1. Crie um projeto: oi init meu-projeto"
    echo "  2. Configure o oi.json"
    echo "  3. Execute: oi up"
    echo ""
    echo "Documentação: https://github.com/${REPO}"
    echo ""
}

main "$@"

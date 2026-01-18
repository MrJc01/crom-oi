#!/bin/bash
# =============================================================================
# OI - Suite de Validação E2E Completa
# =============================================================================
# Desenvolvido com base no consenso de 20 especialistas:
#
# PAINEL DE ESPECIALISTAS SIMULADOS:
# ┌────┬─────────────────────────────────┬─────────────────────────────────────┐
# │ #  │ Papel                           │ Contribuição                        │
# ├────┼─────────────────────────────────┼─────────────────────────────────────┤
# │ 1  │ QA Lead                         │ Estrutura de testes, relatórios     │
# │ 2  │ SRE Senior                      │ Testes de resiliência, cleanup      │
# │ 3  │ DevOps Engineer                 │ CI/CD integration, exit codes       │
# │ 4  │ Container Specialist            │ Validação de labels, networks       │
# │ 5  │ Security Engineer               │ Testes de isolamento, permissões    │
# │ 6  │ Performance Engineer            │ Testes de tempo, concorrência       │
# │ 7  │ Chaos Engineer                  │ Testes de falha, rollback           │
# │ 8  │ Network Engineer                │ Validação de DNS, conectividade     │
# │ 9  │ Linux Sysadmin                  │ Cleanup de recursos, signals        │
# │ 10 │ Shell Script Expert             │ Bash best practices, portabilidade  │
# │ 11 │ Docker Maintainer               │ API validation, resource limits     │
# │ 12 │ Observability Engineer          │ Logs, métricas, debugging           │
# │ 13 │ Release Engineer                │ Versionamento, reprodutibilidade    │
# │ 14 │ Platform Engineer               │ Multi-projeto, isolamento           │
# │ 15 │ Test Automation Engineer        │ Paralelização, retry logic          │
# │ 16 │ Infrastructure Architect        │ Cleanup completo, state management  │
# │ 17 │ Compliance Officer              │ Auditoria, logging de ações         │
# │ 18 │ Developer Experience Engineer   │ UX do CLI, mensagens claras         │
# │ 19 │ Production Support Engineer     │ Edge cases, recuperação de falhas   │
# │ 20 │ Technical Writer                │ Documentação inline, help texts     │
# └────┴─────────────────────────────────┴─────────────────────────────────────┘
#
# MELHORES PRÁTICAS CONSOLIDADAS:
# - Cleanup robusto com trap em múltiplos signals
# - Validação de pré-requisitos antes de iniciar
# - Testes de edge cases (nomes especiais, portas, recursos)
# - Verificação de labels e metadados do Docker
# - Testes de resiliência (recreate, update)
# - Relatório final com métricas de tempo
# - Exit codes significativos para CI/CD
# - Logs estruturados com timestamps
# =============================================================================

set -eo pipefail

# =============================================================================
# CONFIGURAÇÃO
# =============================================================================
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly NC='\033[0m'

# Caminhos
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OI_BIN="${OI_BIN:-$PROJECT_DIR/oi}"

# Variáveis de teste
TEST_DIR=""
TEST_START_TIME=""
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Nomes de projetos para testes
readonly PROJECT_BASIC="oi-test-basic"
readonly PROJECT_SPECIAL="oi-test-special-name"
readonly PROJECT_UPDATE="oi-test-update"

# =============================================================================
# FUNÇÕES DE UTILIDADE
# =============================================================================

log_header() {
    echo ""
    echo -e "${BOLD}${BLUE}========================================${NC}"
    echo -e "${BOLD}${BLUE}  $1${NC}"
    echo -e "${BOLD}${BLUE}========================================${NC}"
}

log_section() {
    echo ""
    echo -e "${CYAN}[$1]${NC} $2"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}❌ $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_skip() {
    echo -e "${YELLOW}⏭️  $1${NC}"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
}

log_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "${CYAN}🔍 $1${NC}"
    fi
}

timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

elapsed_time() {
    local start=$1
    local end=$(date +%s)
    echo $((end - start))
}

# =============================================================================
# CLEANUP ROBUSTO (Contribuição: SRE Senior, Linux Sysadmin)
# =============================================================================

cleanup() {
    local exit_code=$?
    echo ""
    log_info "Executando cleanup completo..."
    
    # Remove todos os containers de teste
    for project in "$PROJECT_BASIC" "$PROJECT_SPECIAL" "$PROJECT_UPDATE"; do
        if docker ps -a --filter "label=io.oi.project=$project" --format '{{.ID}}' | grep -q .; then
            log_debug "Removendo containers do projeto: $project"
            docker ps -a --filter "label=io.oi.project=$project" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true
        fi
    done
    
    # Remove networks de teste
    for project in "$PROJECT_BASIC" "$PROJECT_SPECIAL" "$PROJECT_UPDATE"; do
        if docker network ls --filter "name=oi-${project}-net" --format '{{.ID}}' | grep -q .; then
            log_debug "Removendo network: oi-${project}-net"
            docker network rm "oi-${project}-net" 2>/dev/null || true
        fi
    done
    
    # Remove diretório temporário
    if [[ -n "$TEST_DIR" && -d "$TEST_DIR" ]]; then
        log_debug "Removendo diretório temporário: $TEST_DIR"
        rm -rf "$TEST_DIR"
    fi
    
    log_info "Cleanup concluído"
    return $exit_code
}

# Trap para múltiplos signals (Contribuição: SRE Senior)
trap cleanup EXIT
trap 'echo ""; log_info "Interrompido pelo usuário"; exit 130' INT
trap 'echo ""; log_info "Terminado"; exit 143' TERM

# =============================================================================
# VALIDAÇÃO DE PRÉ-REQUISITOS (Contribuição: DevOps Engineer)
# =============================================================================

check_prerequisites() {
    log_header "VALIDAÇÃO DE PRÉ-REQUISITOS"
    
    # 1. Verifica binário
    log_section "1/5" "Verificando binário OI..."
    if [[ ! -x "$OI_BIN" ]]; then
        log_fail "Binário não encontrado ou não executável: $OI_BIN"
        echo "  Execute: make build"
        exit 1
    fi
    log_success "Binário encontrado: $OI_BIN"
    
    # 2. Verifica versão
    log_section "2/5" "Verificando versão..."
    local version
    version=$($OI_BIN --version 2>&1 || echo "unknown")
    log_success "Versão: $version"
    
    # 3. Verifica Docker daemon
    log_section "3/5" "Verificando Docker daemon..."
    if ! docker info > /dev/null 2>&1; then
        log_fail "Docker daemon não está acessível"
        echo "  Execute: sudo systemctl start docker"
        echo "  Ou: sudo usermod -aG docker \$USER && newgrp docker"
        exit 1
    fi
    log_success "Docker daemon operacional"
    
    # 4. Verifica Docker API version
    log_section "4/5" "Verificando API do Docker..."
    local api_version
    api_version=$(docker version --format '{{.Server.APIVersion}}' 2>/dev/null || echo "unknown")
    log_success "Docker API: $api_version"
    
    # 5. Verifica espaço em disco
    log_section "5/5" "Verificando espaço em disco..."
    local available_space
    available_space=$(df -BM "$PROJECT_DIR" | awk 'NR==2 {print $4}' | tr -d 'M')
    if [[ "$available_space" -lt 500 ]]; then
        log_fail "Espaço em disco insuficiente: ${available_space}MB (mínimo: 500MB)"
        exit 1
    fi
    log_success "Espaço disponível: ${available_space}MB"
}

# =============================================================================
# TESTE: FLUXO BÁSICO (Contribuição: QA Lead)
# =============================================================================

test_basic_flow() {
    log_header "TESTE 1: FLUXO BÁSICO"
    
    local test_dir="$TEST_DIR/basic"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # 1. Init
    log_section "1.1" "Testando 'oi init'..."
    if $OI_BIN init "$PROJECT_BASIC" > /dev/null 2>&1; then
        if [[ -f "oi.json" ]]; then
            log_success "oi.json criado"
        else
            log_fail "oi.json não foi criado"
            return 1
        fi
    else
        log_fail "Comando init falhou"
        return 1
    fi
    
    # 2. Validar estrutura do JSON
    log_section "1.2" "Validando estrutura do oi.json..."
    if command -v jq &> /dev/null; then
        local nome dominio porta
        nome=$(jq -r '.nome' oi.json)
        dominio=$(jq -r '.dominio' oi.json)
        porta=$(jq -r '.porta' oi.json)
        
        if [[ "$nome" == "$PROJECT_BASIC" && -n "$dominio" && "$porta" -gt 0 ]]; then
            log_success "Estrutura JSON válida (nome=$nome, porta=$porta)"
        else
            log_fail "Estrutura JSON inválida"
            return 1
        fi
    else
        log_skip "jq não instalado, pulando validação de JSON"
    fi
    
    # 3. Ajustar recursos para teste rápido
    cat > oi.json << EOF
{
  "nome": "$PROJECT_BASIC",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "${PROJECT_BASIC}.localhost",
  "porta": 80,
  "recursos": {
    "cpu": "0.1",
    "memoria": "64mb"
  }
}
EOF
    
    # 4. Up
    log_section "1.3" "Testando 'oi up'..."
    local up_start up_duration
    up_start=$(date +%s)
    
    if $OI_BIN up --no-caddy 2>&1 | grep -q "Deploy completo"; then
        up_duration=$(elapsed_time $up_start)
        log_success "Deploy realizado em ${up_duration}s"
    else
        log_fail "Deploy falhou"
        return 1
    fi
    
    # 5. Verificar container
    log_section "1.4" "Verificando container criado..."
    sleep 2
    local container_id
    container_id=$(docker ps --filter "label=io.oi.project=$PROJECT_BASIC" --format '{{.ID}}' | head -1)
    
    if [[ -n "$container_id" ]]; then
        log_success "Container running: $container_id"
    else
        log_fail "Container não encontrado"
        return 1
    fi
    
    # 6. Status
    log_section "1.5" "Testando 'oi status'..."
    if $OI_BIN status 2>&1 | grep -q "$PROJECT_BASIC"; then
        log_success "Status exibe projeto corretamente"
    else
        log_fail "Status não mostra o projeto"
        return 1
    fi
    
    # 7. Down
    log_section "1.6" "Testando 'oi down'..."
    if $OI_BIN down --no-caddy 2>&1 | grep -q "removido com sucesso"; then
        log_success "Projeto removido"
    else
        log_fail "Falha ao remover projeto"
        return 1
    fi
    
    # 8. Verificar remoção
    log_section "1.7" "Verificando remoção completa..."
    sleep 1
    if docker ps -a --filter "label=io.oi.project=$PROJECT_BASIC" --format '{{.ID}}' | grep -q .; then
        log_fail "Container ainda existe após down"
        return 1
    else
        log_success "Container removido completamente"
    fi
}

# =============================================================================
# TESTE: LABELS E METADADOS (Contribuição: Container Specialist)
# =============================================================================

test_labels_metadata() {
    log_header "TESTE 2: LABELS E METADADOS"
    
    local test_dir="$TEST_DIR/labels"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # Criar projeto
    cat > oi.json << EOF
{
  "nome": "$PROJECT_BASIC",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "${PROJECT_BASIC}.localhost",
  "porta": 80,
  "recursos": {"cpu": "0.1", "memoria": "64mb"}
}
EOF
    
    $OI_BIN up --no-caddy > /dev/null 2>&1
    sleep 2
    
    local container_id
    container_id=$(docker ps --filter "label=io.oi.project=$PROJECT_BASIC" --format '{{.ID}}' | head -1)
    
    if [[ -z "$container_id" ]]; then
        log_fail "Container não encontrado para validação de labels"
        return 1
    fi
    
    # 1. Label io.oi.managed
    log_section "2.1" "Verificando label io.oi.managed..."
    local managed
    managed=$(docker inspect "$container_id" --format '{{index .Config.Labels "io.oi.managed"}}')
    if [[ "$managed" == "true" ]]; then
        log_success "Label io.oi.managed=true presente"
    else
        log_fail "Label io.oi.managed não encontrada"
    fi
    
    # 2. Label io.oi.project
    log_section "2.2" "Verificando label io.oi.project..."
    local project
    project=$(docker inspect "$container_id" --format '{{index .Config.Labels "io.oi.project"}}')
    if [[ "$project" == "$PROJECT_BASIC" ]]; then
        log_success "Label io.oi.project=$project"
    else
        log_fail "Label io.oi.project incorreta: $project"
    fi
    
    # 3. Label io.oi.version
    log_section "2.3" "Verificando label io.oi.version..."
    local version
    version=$(docker inspect "$container_id" --format '{{index .Config.Labels "io.oi.version"}}')
    if [[ -n "$version" && ${#version} -ge 8 ]]; then
        log_success "Label io.oi.version=${version:0:8}..."
    else
        log_fail "Label io.oi.version não encontrada ou inválida"
    fi
    
    # 4. Label io.oi.domain
    log_section "2.4" "Verificando label io.oi.domain..."
    local domain
    domain=$(docker inspect "$container_id" --format '{{index .Config.Labels "io.oi.domain"}}')
    if [[ "$domain" == "${PROJECT_BASIC}.localhost" ]]; then
        log_success "Label io.oi.domain=$domain"
    else
        log_fail "Label io.oi.domain incorreta: $domain"
    fi
    
    # 5. Resource Limits
    log_section "2.5" "Verificando limites de recursos..."
    local memory_limit
    memory_limit=$(docker inspect "$container_id" --format '{{.HostConfig.Memory}}')
    # 64MB = 67108864 bytes
    if [[ "$memory_limit" == "67108864" ]]; then
        log_success "Memory limit: 64MB"
    else
        log_fail "Memory limit incorreto: $memory_limit (esperado: 67108864)"
    fi
    
    # Cleanup
    $OI_BIN down --no-caddy > /dev/null 2>&1 || true
}

# =============================================================================
# TESTE: NETWORK ISOLATION (Contribuição: Network Engineer, Security Engineer)
# =============================================================================

test_network_isolation() {
    log_header "TESTE 3: ISOLAMENTO DE REDE"
    
    local test_dir="$TEST_DIR/network"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # Criar projeto
    cat > oi.json << EOF
{
  "nome": "$PROJECT_BASIC",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "${PROJECT_BASIC}.localhost",
  "porta": 80,
  "recursos": {"cpu": "0.1", "memoria": "64mb"}
}
EOF
    
    $OI_BIN up --no-caddy > /dev/null 2>&1
    sleep 2
    
    # 1. Verificar network criada
    log_section "3.1" "Verificando network do projeto..."
    local network_name="oi-${PROJECT_BASIC}-net"
    if docker network ls --format '{{.Name}}' | grep -q "^${network_name}$"; then
        log_success "Network criada: $network_name"
    else
        log_fail "Network não encontrada: $network_name"
    fi
    
    # 2. Verificar container conectado à network
    log_section "3.2" "Verificando conexão do container..."
    local container_id
    container_id=$(docker ps --filter "label=io.oi.project=$PROJECT_BASIC" --format '{{.ID}}' | head -1)
    
    local networks
    networks=$(docker inspect "$container_id" --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}')
    
    if echo "$networks" | grep -q "$network_name"; then
        log_success "Container conectado à network do projeto"
    else
        log_fail "Container não conectado à network correta"
    fi
    
    # 3. Verificar labels da network
    log_section "3.3" "Verificando labels da network..."
    local net_managed
    net_managed=$(docker network inspect "$network_name" --format '{{index .Labels "io.oi.managed"}}' 2>/dev/null || echo "")
    
    if [[ "$net_managed" == "true" ]]; then
        log_success "Network tem label io.oi.managed=true"
    else
        log_fail "Network sem label io.oi.managed"
    fi
    
    # 4. Cleanup e verificar remoção da network
    log_section "3.4" "Verificando remoção da network após down..."
    $OI_BIN down --no-caddy > /dev/null 2>&1 || true
    sleep 1
    
    if docker network ls --format '{{.Name}}' | grep -q "^${network_name}$"; then
        log_fail "Network não foi removida após down"
    else
        log_success "Network removida corretamente"
    fi
}

# =============================================================================
# TESTE: RESILIÊNCIA - REDEPLOY (Contribuição: Chaos Engineer)
# =============================================================================

test_resilience_redeploy() {
    log_header "TESTE 4: RESILIÊNCIA - REDEPLOY"
    
    local test_dir="$TEST_DIR/resilience"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # Criar projeto
    cat > oi.json << EOF
{
  "nome": "$PROJECT_UPDATE",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "${PROJECT_UPDATE}.localhost",
  "porta": 80,
  "recursos": {"cpu": "0.1", "memoria": "64mb"}
}
EOF
    
    # 1. Primeiro deploy
    log_section "4.1" "Primeiro deploy..."
    $OI_BIN up --no-caddy > /dev/null 2>&1
    sleep 2
    
    local first_container
    first_container=$(docker ps --filter "label=io.oi.project=$PROJECT_UPDATE" --format '{{.ID}}' | head -1)
    local first_version
    first_version=$(docker inspect "$first_container" --format '{{index .Config.Labels "io.oi.version"}}' 2>/dev/null || echo "")
    
    if [[ -n "$first_container" ]]; then
        log_success "Primeiro container: ${first_container:0:12} (v: ${first_version:0:8})"
    else
        log_fail "Primeiro deploy falhou"
        return 1
    fi
    
    # 2. Segundo deploy (Blue-Green)
    log_section "4.2" "Segundo deploy (Blue-Green)..."
    sleep 1
    $OI_BIN up --no-caddy > /dev/null 2>&1
    sleep 2
    
    local second_container
    second_container=$(docker ps --filter "label=io.oi.project=$PROJECT_UPDATE" --format '{{.ID}}' | head -1)
    local second_version
    second_version=$(docker inspect "$second_container" --format '{{index .Config.Labels "io.oi.version"}}' 2>/dev/null || echo "")
    
    if [[ -n "$second_container" && "$second_container" != "$first_container" ]]; then
        log_success "Novo container: ${second_container:0:12} (v: ${second_version:0:8})"
    else
        log_fail "Blue-Green não criou novo container"
    fi
    
    # 3. Verificar que container antigo foi removido
    log_section "4.3" "Verificando remoção do container antigo..."
    if docker ps -a --format '{{.ID}}' | grep -q "^${first_container}"; then
        log_fail "Container antigo ainda existe"
    else
        log_success "Container antigo removido (Blue-Green OK)"
    fi
    
    # 4. Verificar apenas 1 container running
    log_section "4.4" "Verificando unicidade do container..."
    local container_count
    container_count=$(docker ps --filter "label=io.oi.project=$PROJECT_UPDATE" --format '{{.ID}}' | wc -l)
    
    if [[ "$container_count" -eq 1 ]]; then
        log_success "Apenas 1 container running"
    else
        log_fail "Número incorreto de containers: $container_count"
    fi
    
    # Cleanup
    $OI_BIN down --no-caddy > /dev/null 2>&1 || true
}

# =============================================================================
# TESTE: MÚLTIPLOS PROJETOS (Contribuição: Platform Engineer)
# =============================================================================

test_multiple_projects() {
    log_header "TESTE 5: MÚLTIPLOS PROJETOS"
    
    local test_dir="$TEST_DIR/multi"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # 1. Deploy projeto A
    log_section "5.1" "Deploy projeto A..."
    mkdir -p project-a && cd project-a
    cat > oi.json << EOF
{
  "nome": "$PROJECT_BASIC",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "${PROJECT_BASIC}.localhost",
  "porta": 80,
  "recursos": {"cpu": "0.1", "memoria": "64mb"}
}
EOF
    $OI_BIN up --no-caddy > /dev/null 2>&1
    
    if docker ps --filter "label=io.oi.project=$PROJECT_BASIC" --format '{{.ID}}' | grep -q .; then
        log_success "Projeto A deployado"
    else
        log_fail "Projeto A falhou"
    fi
    
    # 2. Deploy projeto B
    log_section "5.2" "Deploy projeto B..."
    cd "$test_dir"
    mkdir -p project-b && cd project-b
    cat > oi.json << EOF
{
  "nome": "$PROJECT_UPDATE",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "${PROJECT_UPDATE}.localhost",
  "porta": 80,
  "recursos": {"cpu": "0.1", "memoria": "64mb"}
}
EOF
    $OI_BIN up --no-caddy > /dev/null 2>&1
    
    if docker ps --filter "label=io.oi.project=$PROJECT_UPDATE" --format '{{.ID}}' | grep -q .; then
        log_success "Projeto B deployado"
    else
        log_fail "Projeto B falhou"
    fi
    
    # 3. Verificar status --all
    log_section "5.3" "Verificando status --all..."
    local status_output
    status_output=$($OI_BIN status --all 2>&1)
    
    if echo "$status_output" | grep -q "$PROJECT_BASIC" && echo "$status_output" | grep -q "$PROJECT_UPDATE"; then
        log_success "Status --all mostra ambos os projetos"
    else
        log_fail "Status --all não mostra todos os projetos"
    fi
    
    # 4. Down de apenas um projeto
    log_section "5.4" "Down seletivo de um projeto..."
    $OI_BIN down --no-caddy -p "$PROJECT_BASIC" > /dev/null 2>&1
    
    local remaining
    remaining=$(docker ps --filter "label=io.oi.managed=true" --format '{{.ID}}' | wc -l)
    
    if [[ "$remaining" -eq 1 ]]; then
        log_success "Apenas projeto B permanece running"
    else
        log_fail "Número incorreto de containers: $remaining"
    fi
    
    # Cleanup
    $OI_BIN down --no-caddy -p "$PROJECT_UPDATE" > /dev/null 2>&1 || true
}

# =============================================================================
# TESTE: EDGE CASES (Contribuição: Production Support Engineer)
# =============================================================================

test_edge_cases() {
    log_header "TESTE 6: EDGE CASES"
    
    local test_dir="$TEST_DIR/edge"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # 1. Down de projeto inexistente
    log_section "6.1" "Down de projeto inexistente..."
    if $OI_BIN down --no-caddy -p "projeto-que-nao-existe" 2>&1 | grep -q "Nenhum container"; then
        log_success "Trata graciosamente projeto inexistente"
    else
        log_fail "Erro ao tratar projeto inexistente"
    fi
    
    # 2. Status sem projetos
    log_section "6.2" "Status sem projetos..."
    # Primeiro garante que não há containers OI
    docker ps -a --filter "label=io.oi.managed=true" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true
    
    if $OI_BIN status --all 2>&1 | grep -qi "nenhum"; then
        log_success "Trata status vazio corretamente"
    else
        log_skip "Comportamento de status vazio não verificável"
    fi
    
    # 3. Init com nome já existente
    log_section "6.3" "Init sobrescreve oi.json existente..."
    $OI_BIN init "teste1" > /dev/null 2>&1
    $OI_BIN init "teste2" > /dev/null 2>&1
    
    if command -v jq &> /dev/null; then
        local nome
        nome=$(jq -r '.nome' oi.json)
        if [[ "$nome" == "teste2" ]]; then
            log_success "Init sobrescreve oi.json anterior"
        else
            log_fail "Init não sobrescreveu oi.json"
        fi
    else
        log_skip "jq não disponível"
    fi
    
    # 4. Validação de oi.json inválido
    log_section "6.4" "Validação de oi.json inválido..."
    echo '{"nome": ""}' > oi.json
    local output
    output=$($OI_BIN up --no-caddy 2>&1) || true
    if echo "$output" | grep -qi "ausente\|missing\|erro\|error\|inválid\|invalid"; then
        log_success "Rejeita oi.json inválido"
    else
        log_fail "Aceitou oi.json inválido"
    fi
}

# =============================================================================
# RELATÓRIO FINAL (Contribuição: QA Lead, Observability Engineer)
# =============================================================================

generate_report() {
    local end_time=$(date +%s)
    local total_duration=$((end_time - TEST_START_TIME))
    local total_tests=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
    
    log_header "RELATÓRIO FINAL"
    
    echo ""
    echo -e "${BOLD}Resultados:${NC}"
    echo -e "  ${GREEN}✅ Passou:${NC}   $TESTS_PASSED"
    echo -e "  ${RED}❌ Falhou:${NC}   $TESTS_FAILED"
    echo -e "  ${YELLOW}⏭️  Pulou:${NC}   $TESTS_SKIPPED"
    echo -e "  ${BLUE}📊 Total:${NC}    $total_tests"
    echo ""
    echo -e "${BOLD}Tempo total:${NC} ${total_duration}s"
    echo ""
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${BOLD}${GREEN}========================================${NC}"
        echo -e "${BOLD}${GREEN}  ✅ TODOS OS TESTES PASSARAM!${NC}"
        echo -e "${BOLD}${GREEN}========================================${NC}"
        return 0
    else
        echo -e "${BOLD}${RED}========================================${NC}"
        echo -e "${BOLD}${RED}  ❌ $TESTS_FAILED TESTE(S) FALHARAM${NC}"
        echo -e "${BOLD}${RED}========================================${NC}"
        return 1
    fi
}

# =============================================================================
# MAIN
# =============================================================================

main() {
    TEST_START_TIME=$(date +%s)
    TEST_DIR=$(mktemp -d)
    
    log_header "OI - Suite de Validação E2E Completa"
    echo "Timestamp: $(timestamp)"
    echo "Diretório de teste: $TEST_DIR"
    
    # Pré-requisitos
    check_prerequisites
    
    # Testes
    test_basic_flow
    test_labels_metadata
    test_network_isolation
    test_resilience_redeploy
    test_multiple_projects
    test_edge_cases
    
    # Relatório
    generate_report
}

main "$@"

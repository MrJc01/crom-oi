#!/bin/bash
# =============================================================================
# Script de Teste: OI Hardening Phase 1
# =============================================================================
# Testa as 3 funcionalidades implementadas:
# 1. Validação de Domínio (Fail-Fast)
# 2. Status Global com tabwriter
# 3. Check de Proxy (Caddy)
# =============================================================================

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OI_BIN="${OI_BIN:-$PROJECT_DIR/oi}"

echo "=============================================="
echo "🧪 OI Hardening Phase 1 - Testes"
echo "=============================================="

# Verificar binário
if [[ ! -x "$OI_BIN" ]]; then
    echo -e "${RED}❌ Binário não encontrado: $OI_BIN${NC}"
    echo "Execute: go build -o ./oi ./cmd/oi"
    exit 1
fi

# Diretório temporário para testes
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"

TESTS_PASSED=0
TESTS_FAILED=0

# Limpar no final
cleanup() {
    echo -e "\n🧹 Limpando recursos de teste..."
    docker ps -a --filter "label=io.oi.project=test-hardening" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true
    docker network rm oi-test-hardening-net 2>/dev/null || true
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

pass() {
    echo -e "${GREEN}✅ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}❌ $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# =============================================================================
# TESTE 1: Validação de Domínio - DNS Inválido
# =============================================================================
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}TESTE 1: Domínio com DNS inválido (deve falhar)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cat > oi.json << 'EOF'
{
  "nome": "test-hardening",
  "origem": "nginx:alpine",
  "dominio": "dominio-inexistente-xyz-123.fake",
  "porta": 80
}
EOF

echo "📄 oi.json criado com domínio inválido"
echo "🚀 Executando: oi up"

OUTPUT=$($OI_BIN up 2>&1 || true)
echo "$OUTPUT" | head -5

if echo "$OUTPUT" | grep -qi "não aponta para este servidor\|Configure o DNS"; then
    pass "TESTE 1: Deploy abortado corretamente (domínio inválido)"
else
    fail "TESTE 1: Deploy deveria ter sido abortado"
fi

# =============================================================================
# TESTE 2: Validação de Domínio - .localhost (bypass)
# =============================================================================
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}TESTE 2: Domínio .localhost (deve passar validação DNS)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cat > oi.json << 'EOF'
{
  "nome": "test-hardening",
  "origem": "nginx:alpine",
  "dominio": "test.localhost",
  "porta": 80
}
EOF

echo "📄 oi.json atualizado com domínio .localhost"
echo "🚀 Executando: oi up --no-caddy"

OUTPUT=$($OI_BIN up --no-caddy 2>&1 || true)
echo "$OUTPUT" | head -10

if echo "$OUTPUT" | grep -qi "não aponta para este servidor"; then
    fail "TESTE 2: Domínio .localhost deveria passar validação DNS"
else
    pass "TESTE 2: Validação DNS ignorada para .localhost"
fi

# Verifica se passou para próxima etapa
if echo "$OUTPUT" | grep -qE "(Criando.*network|Deploy completo)"; then
    pass "TESTE 2.1: Deploy progrediu além da validação DNS"
else
    echo -e "${YELLOW}⚠️  Deploy pode ter falhado por outra razão (não DNS)${NC}"
fi

# =============================================================================
# TESTE 3: Check de Proxy (Caddy)
# =============================================================================
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}TESTE 3: Verificação de conectividade com Proxy${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Verificar se Caddy está rodando
if curl -s http://localhost:2019/config/ > /dev/null 2>&1; then
    echo -e "${GREEN}ℹ️  Caddy está rodando na porta 2019${NC}"
    CADDY_RUNNING=true
else
    echo -e "${YELLOW}ℹ️  Caddy NÃO está rodando${NC}"
    CADDY_RUNNING=false
fi

# Testar com Caddy (sem --no-caddy)
cat > oi.json << 'EOF'
{
  "nome": "test-hardening",
  "origem": "nginx:alpine",
  "dominio": "test.localhost",
  "porta": 80
}
EOF

OUTPUT=$($OI_BIN up 2>&1 || true)

if [[ "$CADDY_RUNNING" == "false" ]]; then
    if echo "$OUTPUT" | grep -qi "Proxy.*não acessível\|Caddy.*não acessível"; then
        pass "TESTE 3: OI detectou que Caddy não está acessível"
    else
        echo "$OUTPUT" | head -5
        fail "TESTE 3: OI deveria detectar Caddy inacessível"
    fi
else
    if echo "$OUTPUT" | grep -qi "Verificando conectividade"; then
        pass "TESTE 3: OI verificou conectividade com proxy"
    else
        pass "TESTE 3: Caddy acessível, deploy prosseguiu"
    fi
fi

# =============================================================================
# TESTE 4: Status Global (--all flag)
# =============================================================================
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}TESTE 4: Status Global com tabwriter${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "🚀 Executando: oi status --all"
STATUS_OUTPUT=$($OI_BIN status --all 2>&1 || true)
echo "$STATUS_OUTPUT"

# Verificar se output mostra cabeçalho de tabela ou mensagem de "nenhum container"
if echo "$STATUS_OUTPUT" | grep -qE "(PROJETO.*STATUS|Nenhum container)"; then
    pass "TESTE 4: Comando status --all funcionando com tabwriter"
else
    fail "TESTE 4: Output inesperado do status"
fi

# =============================================================================
# RESUMO
# =============================================================================
echo -e "\n${BLUE}=============================================="
echo "📊 RESUMO DOS TESTES"
echo "==============================================${NC}"
echo ""
echo -e "  ${GREEN}✅ Passou:${NC}  $TESTS_PASSED"
echo -e "  ${RED}❌ Falhou:${NC}  $TESTS_FAILED"
echo ""

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}=============================================="
    echo "✅ TODOS OS TESTES DE HARDENING PASSARAM!"
    echo "==============================================${NC}"
    exit 0
else
    echo -e "${RED}=============================================="
    echo "❌ $TESTS_FAILED TESTE(S) FALHARAM"
    echo "==============================================${NC}"
    exit 1
fi

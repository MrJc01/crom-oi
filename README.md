# OI - Orquestrador de Intenção

> **Deployment through Intent, not Configuration.**

OI é um orquestrador de containers focado em **Intenção** em vez de **Configuração**. O usuário não gerencia infraestrutura. Apenas dá um `oi up`.

## ✨ Filosofia

```
┌──────────────┐      ┌─────────────────┐      ┌─────────────┐
│   oi.json    │ ───▶ │   OI Engine     │ ───▶ │   Docker    │
│  (Intenção)  │      │ (Reconciliador) │      │  (Realidade)│
└──────────────┘      └────────┬────────┘      └─────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │     Caddy       │
                      │ (Proxy + SSL)   │
                      └─────────────────┘
```

O OI lê um arquivo `oi.json` e **garante que a realidade do servidor** (Docker/Rede/SSL) **corresponda exatamente à intenção descrita**.

## 🚀 Quick Start

### 1. Inicialize um projeto
```bash
oi init meu-app
```
Isso cria um arquivo `oi.json` padrão.

### 2. Edite a intenção (Opcional)
```json
{
  "nome": "meu-app",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "meu-app.localhost",
  "porta": 80,
  "recursos": { "cpu": "0.1", "memoria": "64mb" }
}
```

### 3. Deploy (Modo Desenvolvimento)
Para testar localmente sem configurar proxy/SSL, use a flag `--no-caddy`. O OI irá expor a porta automaticamente no `localhost`.

```bash
sudo oi up --no-caddy
```
> Acesse: http://localhost:80 (ou a porta definida no seu json)

### 4. Deploy (Modo Produção)
Para deploy completo com SSL automático e domínio (requer Caddy rodando):

```bash
sudo oi up
```
> Acesse: https://meu-app.localhost

### 5. Verificar Status
```bash
oi status
```

### 6. Remover Projeto (Teardown)
```bash
sudo oi down
```

## 🔧 Comandos

| Comando | Descrição |
|---------|-----------|
| `oi up` | Deploy atômico com Blue-Green (Zero-Downtime) |
| `oi down` | Remove containers, rede e rotas do proxy |
| `oi status` | Mostra estado atual e saúde dos containers |
| `oi init [nome]` | Cria template de `oi.json` |

## 🌟 Features Principais

- **Zero-Downtime Deploy**: Estratégia Blue-Green nativa. O novo container só substitui o antigo após passar no health check.
- **Rollback Automático**: Se o novo deploy falhar, o sistema mantém a versão anterior intacta.
- **Isolamento de Rede**: Cada projeto ganha sua própria rede Docker isolada (`oi-<projeto>-net`).
- **Dev Mode Automático**: Se não houver proxy, publica portas no host para facilitar testes (`--no-caddy`).
- **State via Labels**: Docker labels são a única "tabela de estado". Sem banco de dados para gerenciar.

## 🏗️ Build e Validação

```bash
# Build do binário
make build

# Instalar no sistema (/usr/local/bin)
sudo make install

# Executar suite de testes E2E
sudo ./scripts/validate_e2e.sh
```

## 📦 Requisitos

- **Docker Engine** 20.10+ (API 1.41+)
- **Go** 1.22+ (para compilar)
- **Caddy** 2.x (Opcional, apenas para modo produção com SSL)

## 📄 Licença

MIT

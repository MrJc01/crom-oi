# OI - Orquestrador de Intenção

> **Deployment through Intent, not Configuration.**

OI (Orquestrador de Intenção) é uma ferramenta CLI minimalista (`3MB`) para gerenciar containers com foco no **O QUE** você quer, não no **COMO** fazer. Eliminamos a complexidade de `Dockerfiles` e `docker-compose.yml` para a maioria dos casos de uso.

[![Release](https://img.shields.io/github/v/release/MrJc01/crom-oi?style=flat-square)](https://github.com/MrJc01/crom-oi/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/MrJc01/crom-oi/release.yml?style=flat-square)](https://github.com/MrJc01/crom-oi/actions)
[![License](https://img.shields.io/github/license/MrJc01/crom-oi?style=flat-square)](LICENSE)

## 📥 Instalação

Instale a versão mais recente com um único comando:

```bash
curl -sSL https://raw.githubusercontent.com/MrJc01/crom-oi/main/scripts/install.sh | sudo bash
```

O script detecta automaticamente seu sistema (Linux x86_64 ou ARM64) e instala as dependências necessárias.

---

## 🚀 Guia Rápido

### 1. Inicialize
```bash
oi init meu-app
```

### 2. Defina a Intenção (`oi.json`)
```json
{
  "nome": "meu-app",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "meu-app.localhost",
  "porta": 80,
  "recursos": { "cpu": "0.5", "memoria": "128mb" },
  "dev": {
    "volumes": ["./src:/usr/share/nginx/html"]
  }
}
```

### 3. Deploy
```bash
# Produção (Usa a imagem definida em 'origem')
sudo oi up

# Modo Live (Monta volumes locais para Hot Reload)
sudo oi up --live

# Arquivo Específico
sudo oi up meu-projeto.json

# Todos os arquivos da pasta
sudo oi up --all

# Filtrando arquivos
sudo oi up --filter "servico-*.json"
```

---

## 🔧 Referência de Comandos

### `oi up`
Realiza ou atualiza o deploy da intenção.
- **Uso:** `oi up [arquivo] [flags]`
- **Flags:**
  - `--all`: Processa todos os arquivos `.json` do diretório atual.
  - `--filter`: Filtra arquivos usando glob pattern (ex: `*-prod.json`).
  - `--live`: Ativa o "Modo Live".
  - `--no-caddy`: Desabilita Caddy.

### `oi down` (ou `oi remove`)
Remove recursos.
- **Uso:** `oi down [flags]`
- **Flags:**
  - `--all`: 🚨 **Limpeza Total**. Remove TODOS os containers e redes gerenciados pelo OI.
  - `-p, --project`: Especifica um projeto para remover.

### `oi status`
Mostra o estado dos containers.
- **Uso:** `oi status [flags]`
- **Flags:**
  - `-a, --all`: Mostra todos os containers OI rodando no sistema, não apenas do projeto atual.
  - `-p, --project`: Filtra por projeto.

### `oi logs` (Live Stream)
Acompanha os logs do container em tempo real (como `tail -f`).
- **Uso:** `oi logs [flags]`
- **Flags:**
  - `--tail`: Número de linhas iniciais (default "all").

### `oi log` (Dump)
Despeja todo o log do container e sai. Útil para pipe em arquivos ou grep.
- **Uso:** `oi log > arquivo.txt`

### `oi stop` e `oi start`
Pausa e inicia containers sem removê-los.
- **Uso:** `oi stop [flags]` / `oi start [flags]`
- **Flags:**
  - `--all`: Aplica a ação em **todos** os containers OI.

### `oi info`
Exibe diagnósticos do sistema (Versão, Docker Daemon, Caddy, Redes).

### `oi init`
Cria um esqueleto de arquivo `oi.json`.
- **Uso:** `oi init [nome-do-app] [flags]`
- **Flags:**
  - `-d, --dockerfile`: Lê um `Dockerfile` existente para extrair a porta (`EXPOSE`) e configurar o projeto automaticamente.

### `oi update` (ou `oi upgrade`)
Verifica e instala a última versão estável do OI.
- Mantém o histórico de versões antigas em `~/.oi/versions/`.
- **Uso:** `sudo oi update` (Requer sudo se instalado em `/usr/local/bin`).

---

## ⚙️ Configuração (`oi.json`)

O arquivo `oi.json` é a fonte da verdade.

| Campo | Descrição | Exemplo |
|-------|-----------|---------|
| `nome` / `name` | Nome único do projeto. | `"meu-blog"` |
| `origem` / `origin` | Imagem Docker base. | `"wordpress:latest"` |
| `dominio` / `domain` | Domínio ou subdomínio local. | `"blog.localhost"` |
| `porta` / `port` | Porta interna do container. | `80` |
| `recursos` / `resources` | Limites de hardware. | `{"cpu": "1.0", "memory": "512mb"}` |
| `dev.volumes` | Mapeamento de volumes. | `["./src:/app"]` |

> **Nota:** Você pode usar chaves em **Português** ou **Inglês**. O OI entende ambas! 🇺🇸 🇧🇷

---

## 🌟 Features Principais

- **🛡️ Hardening Nativo**: Validação fail-fast de DNS e checagem de integridade do Proxy.
- **🔄 Zero-Downtime Deploy**: Estratégia Blue-Green. O novo container só entra se passar no health check (Green), senão o antigo (Blue) permanece.
- **🔥 Hot Reload (--live)**: Desenvolvimento local com containers, montando seu código fonte como volume.
- **🔙 Rollback Automático**: Falhou no boot? O OI reverte automaticamente.
- **🔒 Isolamento de Rede**: Cada projeto tem sua rede isolada.
- **🌐 SSL Automático**: Caddy cuida dos certificados.

---

## 🏗️ Desenvolvimento do OI (Contribuindo)

```bash
# Build local
go build -o oi ./cmd/oi

# Instalar binário construído
sudo mv oi /usr/local/bin/oi
```

Licença MIT © 2024

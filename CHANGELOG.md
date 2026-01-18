# Changelog

Todas as mudanças notáveis no projeto OI serão documentadas neste arquivo.

## [v0.3.0] - 2026-01-18
### Adicionado
- **Init Dockerfile**: Novo comando `oi init --dockerfile` que lê um Dockerfile existente, extrai a porta (`EXPOSE`) e gera o `oi.json` automaticamente.

## [v0.2.0] - 2026-01-18
### Adicionado
- **Live Mode**: Suporte a Hot Reload com `oi up --live`. Monta volumes locais definidos em `dev.volumes` no `oi.json`.
- **Logs Streaming**: Comando `oi logs` para acompanhar logs em tempo real (como `docker logs -f`).
- **Logs Dump**: Comando `oi log` para despejar todo o histórico de logs.
- **Info**: Comando `oi info` para diagnósticos do sistema (Docker, Caddy, Versão).
- **Smart Status**: Exibe `💚` para containers "Running" mesmo sem healthcheck explícito.

## [v0.1.0] - 2026-01-18
### Adicionado
- **Global Management**: Comandos `oi stop`, `oi start`, `oi remove` (alias `down`).
- **Bulk Operations**: Flag `--all` para operar em todos os projetos de uma vez.
- **Smart Port Binding**: Suporte a `"porta": 0` no `oi.json` para alocação dinâmica de portas.
- **Hardening**: Fail-fast para DNS e Caddy Healthcheck.

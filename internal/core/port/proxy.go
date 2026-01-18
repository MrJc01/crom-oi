package port

import "context"

// ProxyManager define as operações para gerenciar o reverse proxy
// Abstraído do Caddy para facilitar testes e possível troca de proxy
type ProxyManager interface {
	// AddRoute adiciona ou atualiza uma rota de domínio para upstream
	AddRoute(ctx context.Context, domain string, upstream string, port int) error

	// RemoveRoute remove uma rota de domínio
	RemoveRoute(ctx context.Context, domain string) error

	// HasRoute verifica se uma rota existe
	HasRoute(ctx context.Context, domain string) (bool, error)

	// GetUpstream retorna o upstream atual para um domínio
	GetUpstream(ctx context.Context, domain string) (string, error)

	// Reload força recarregamento da configuração
	Reload(ctx context.Context) error

	// Health verifica se o proxy está saudável
	Health(ctx context.Context) error
}

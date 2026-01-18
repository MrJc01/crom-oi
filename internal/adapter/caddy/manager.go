package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Manager implementa port.ProxyManager usando Caddy Admin API
type Manager struct {
	adminURL   string
	httpClient *http.Client
}

// NewManager cria uma nova instância do Caddy Manager
func NewManager(adminURL string) *Manager {
	if adminURL == "" {
		adminURL = "http://localhost:2019"
	}
	return &Manager{
		adminURL: adminURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// routeConfig representa a configuração de rota do Caddy
type routeConfig struct {
	Match    []matchConfig  `json:"match"`
	Handle   []handleConfig `json:"handle"`
	Terminal bool           `json:"terminal"`
}

type matchConfig struct {
	Host []string `json:"host"`
}

type handleConfig struct {
	Handler   string        `json:"handler"`
	Upstreams []upstream    `json:"upstreams,omitempty"`
	Routes    []interface{} `json:"routes,omitempty"`
}

type upstream struct {
	Dial string `json:"dial"`
}

// AddRoute adiciona ou atualiza uma rota de domínio para upstream
func (m *Manager) AddRoute(ctx context.Context, domain string, upstreamHost string, port int) error {
	route := routeConfig{
		Match: []matchConfig{
			{Host: []string{domain}},
		},
		Handle: []handleConfig{
			{
				Handler: "reverse_proxy",
				Upstreams: []upstream{
					{Dial: fmt.Sprintf("%s:%d", upstreamHost, port)},
				},
			},
		},
		Terminal: true,
	}

	body, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("falha ao serializar rota: %w", err)
	}

	// POST /config/apps/http/servers/srv0/routes/...
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", m.adminURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("falha ao criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao comunicar com Caddy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Caddy retornou erro %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// RemoveRoute remove uma rota de domínio
func (m *Manager) RemoveRoute(ctx context.Context, domain string) error {
	// Primeiro, precisa encontrar o índice da rota
	// GET /config/apps/http/servers/srv0/routes
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", m.adminURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("falha ao criar request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao comunicar com Caddy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil // Não existe, nada a fazer
	}

	var routes []routeConfig
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return fmt.Errorf("falha ao parsear rotas: %w", err)
	}

	// Encontra o índice da rota com o domínio
	for i, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				if host == domain {
					// DELETE /config/apps/http/servers/srv0/routes/<index>
					deleteURL := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes/%d", m.adminURL, i)
					delReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
					delResp, err := m.httpClient.Do(delReq)
					if err != nil {
						return fmt.Errorf("falha ao remover rota: %w", err)
					}
					delResp.Body.Close()
					return nil
				}
			}
		}
	}

	return nil // Rota não encontrada, ok
}

// HasRoute verifica se uma rota existe
func (m *Manager) HasRoute(ctx context.Context, domain string) (bool, error) {
	upstream, err := m.GetUpstream(ctx, domain)
	if err != nil {
		return false, err
	}
	return upstream != "", nil
}

// GetUpstream retorna o upstream atual para um domínio
func (m *Manager) GetUpstream(ctx context.Context, domain string) (string, error) {
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", m.adminURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("falha ao criar request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("falha ao comunicar com Caddy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", nil
	}

	var routes []routeConfig
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return "", fmt.Errorf("falha ao parsear rotas: %w", err)
	}

	for _, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				if host == domain {
					if len(route.Handle) > 0 && len(route.Handle[0].Upstreams) > 0 {
						return route.Handle[0].Upstreams[0].Dial, nil
					}
				}
			}
		}
	}

	return "", nil
}

// Reload força recarregamento da configuração
func (m *Manager) Reload(ctx context.Context) error {
	// Caddy não precisa de reload explícito, as mudanças são aplicadas imediatamente
	return nil
}

// Health verifica se o Caddy está saudável
func (m *Manager) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/config/", m.adminURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("falha ao criar request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Caddy não acessível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Caddy retornou status %d", resp.StatusCode)
	}

	return nil
}

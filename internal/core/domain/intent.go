package domain

import "time"

// Intent representa a intenção declarada no arquivo oi.json
// É a "fonte da verdade" do que o usuário deseja
type Intent struct {
	Nome     string   `json:"nome"`
	Origem   string   `json:"origem"`
	Dominio  string   `json:"dominio"`
	Porta    int      `json:"porta"`
	Recursos Recursos `json:"recursos"`
}

// Recursos define os limites de CPU e memória para o container
type Recursos struct {
	CPU     string `json:"cpu"`
	Memoria string `json:"memoria"`
}

// Validate verifica se a intenção está completa
func (i *Intent) Validate() error {
	if i.Nome == "" {
		return ErrMissingField("nome")
	}
	if i.Origem == "" {
		return ErrMissingField("origem")
	}
	if i.Dominio == "" {
		return ErrMissingField("dominio")
	}
	if i.Porta < 0 || i.Porta > 65535 {
		return ErrInvalidPort
	}
	return nil
}

// ContainerStatus representa o estado atual de um container
type ContainerStatus string

const (
	StatusCreated  ContainerStatus = "created"
	StatusRunning  ContainerStatus = "running"
	StatusStopped  ContainerStatus = "stopped"
	StatusRemoving ContainerStatus = "removing"
)

// HealthStatus representa o status de saúde do container
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthStarting  HealthStatus = "starting"
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// Container representa o estado atual de um container gerenciado pelo OI
type Container struct {
	ID         string
	Name       string
	Project    string
	Version    string
	Image      string
	Status     ContainerStatus
	Health     HealthStatus
	CreatedAt  time.Time
	PublicPort int
}

// IsHealthy retorna true se o container está saudável e pronto para receber tráfego
func (c *Container) IsHealthy() bool {
	return c.Status == StatusRunning && c.Health == HealthHealthy
}

// IsRunning retorna true se o container está em execução
func (c *Container) IsRunning() bool {
	return c.Status == StatusRunning
}

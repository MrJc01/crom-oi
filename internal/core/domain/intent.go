package domain

import "time"

// Intent representa a intenção declarada no arquivo oi.json
// É a "fonte da verdade" do que o usuário deseja
type Intent struct {
	// Portuguese
	Nome     string   `json:"nome,omitempty"`
	Origem   string   `json:"origem,omitempty"`
	Dominio  string   `json:"dominio,omitempty"`
	Porta    int      `json:"porta,omitempty"`
	Recursos Recursos `json:"recursos,omitempty"`

	// English
	Name      string   `json:"name,omitempty"`
	Origin    string   `json:"origin,omitempty"`
	Domain    string   `json:"domain,omitempty"`
	Port      int      `json:"port,omitempty"`
	Resources Recursos `json:"resources,omitempty"`

	Dev DevConfig `json:"dev,omitempty"`
}

// Recursos define os limites de CPU e memória para o container
type Recursos struct {
	// Portuguese
	CPU     string `json:"cpu,omitempty"`
	Memoria string `json:"memoria,omitempty"`

	// English
	Memory string `json:"memory,omitempty"`
}

// Normalize consolida os campos em Inglês para os campos em Português
func (i *Intent) Normalize() {
	if i.Nome == "" {
		i.Nome = i.Name
	}
	if i.Origem == "" {
		i.Origem = i.Origin
	}
	if i.Dominio == "" {
		i.Dominio = i.Domain
	}
	if i.Porta == 0 {
		i.Porta = i.Port
	}

	// Recursos
	if i.Recursos.CPU == "" && i.Resources.CPU != "" {
		i.Recursos.CPU = i.Resources.CPU
	}
	if i.Recursos.Memoria == "" {
		if i.Resources.Memory != "" {
			i.Recursos.Memoria = i.Resources.Memory
		} else if i.Resources.Memoria != "" {
			i.Recursos.Memoria = i.Resources.Memoria
		}
	}
}

// DevConfig define configurações específicas para desenvolvimento (oi up --live)
type DevConfig struct {
	// Volumes para montar (ex: ["./src:/app"])
	Volumes []string `json:"volumes"`
	// Comando alternativo (ex: ["npm", "run", "dev"])
	Command []string `json:"command,omitempty"`
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

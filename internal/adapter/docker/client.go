package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/crom-tech/oi/internal/core/domain"
	"github.com/crom-tech/oi/pkg/labels"
)

// Client implementa port.ContainerRuntime usando Docker SDK
type Client struct {
	cli *client.Client
}

// NewClient cria uma nova instância do Docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar Docker client: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close fecha a conexão com o Docker
func (c *Client) Close() error {
	return c.cli.Close()
}

// Ping verifica se o Docker está acessível
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

// List retorna containers gerenciados pelo OI
func (c *Client) List(ctx context.Context, project string) ([]domain.Container, error) {
	f := filters.NewArgs()
	f.Add("label", labels.ManagedFilter())

	if project != "" {
		f.Add("label", labels.ProjectFilter(project))
	}

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true, // Inclui containers parados
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar containers: %w", err)
	}

	result := make([]domain.Container, 0, len(containers))
	for _, ctr := range containers {
		result = append(result, c.toDomainContainer(ctr))
	}

	return result, nil
}

// Pull baixa uma imagem do registry
func (c *Client) Pull(ctx context.Context, imageName string) error {
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("falha ao baixar imagem %s: %w", imageName, err)
	}
	defer reader.Close()

	// Consome o output (necessário para completar o pull)
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("falha ao completar pull da imagem: %w", err)
	}

	return nil
}

// Create cria um novo container baseado na intenção
func (c *Client) Create(ctx context.Context, intent domain.Intent, version string, publishPort bool) (string, error) {
	containerName := c.containerName(intent.Nome, version)
	networkName := c.networkName(intent.Nome)

	// Configuração do container
	// Se a porta for 0 na intenção, usamos 80 como porta interna padrão do container
	// mas deixamos o bind externo como aleatório
	internalPort := intent.Porta
	if internalPort == 0 {
		internalPort = 80 // Porta padrão interna comum
	}
	exposedPort := nat.Port(fmt.Sprintf("%d/tcp", internalPort))

	config := &container.Config{
		Image: intent.Origem,
		Labels: labels.OILabels(
			intent.Nome,
			version,
			intent.Dominio,
			intent.Porta,
		),
		ExposedPorts: nat.PortSet{
			exposedPort: struct{}{},
		},
	}

	// Configuração do host (recursos e portas)
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			NanoCPUs: c.parseCPU(intent.Recursos.CPU),
			Memory:   c.parseMemory(intent.Recursos.Memoria),
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Se publishPort for true, mapeia a porta no host
	if publishPort {
		hostPort := fmt.Sprintf("%d", intent.Porta)
		if intent.Porta == 0 {
			hostPort = "0" // Docker aloca porta aleatória
		}

		hostConfig.PortBindings = nat.PortMap{
			exposedPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0", // Bind em todas as interfaces para garantir acesso
					HostPort: hostPort,
				},
			},
		}
	}

	// Configuração de rede
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := c.cli.ContainerCreate(
		ctx,
		config,
		hostConfig,
		networkingConfig,
		nil, // platform
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("falha ao criar container: %w", err)
	}

	return resp.ID, nil
}

// Start inicia um container
func (c *Client) Start(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("falha ao iniciar container %s: %w", containerID, err)
	}
	return nil
}

// Stop para um container gracefully
func (c *Client) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	timeoutSec := int(timeout.Seconds())
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSec,
	}); err != nil {
		return fmt.Errorf("falha ao parar container %s: %w", containerID, err)
	}
	return nil
}

// Remove remove um container
func (c *Client) Remove(ctx context.Context, containerID string, force bool) error {
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false, // Preserva volumes para segurança
	}); err != nil {
		return fmt.Errorf("falha ao remover container %s: %w", containerID, err)
	}
	return nil
}

// WaitHealthy aguarda o container ficar healthy ou timeout
func (c *Client) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return domain.ErrHealthCheckFailed{
					ContainerID: containerID,
					Reason:      "timeout esperando container ficar healthy",
				}
			}

			info, err := c.cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("falha ao inspecionar container: %w", err)
			}

			// Se não tem health check configurado, considera running como healthy
			if info.State.Health == nil {
				if info.State.Running {
					return nil
				}
				continue
			}

			switch info.State.Health.Status {
			case "healthy":
				return nil
			case "unhealthy":
				return domain.ErrHealthCheckFailed{
					ContainerID: containerID,
					Reason:      "container reportou unhealthy",
				}
			}
		}
	}
}

// Inspect retorna informações detalhadas de um container
func (c *Client) Inspect(ctx context.Context, containerID string) (*domain.Container, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("falha ao inspecionar container: %w", err)
	}

	ctr := &domain.Container{
		ID:      info.ID,
		Name:    strings.TrimPrefix(info.Name, "/"),
		Project: info.Config.Labels[labels.Project],
		Version: info.Config.Labels[labels.Version],
		Image:   info.Config.Image,
	}

	// Descobrir porta pública mapeada
	for _, bindings := range info.NetworkSettings.Ports {
		if len(bindings) > 0 {
			var p int
			fmt.Sscanf(bindings[0].HostPort, "%d", &p)
			ctr.PublicPort = p
			break
		}
	}

	// Status
	if info.State.Running {
		ctr.Status = domain.StatusRunning
	} else {
		ctr.Status = domain.StatusStopped
	}

	// Health
	if info.State.Health != nil {
		switch info.State.Health.Status {
		case "healthy":
			ctr.Health = domain.HealthHealthy
		case "unhealthy":
			ctr.Health = domain.HealthUnhealthy
		case "starting":
			ctr.Health = domain.HealthStarting
		default:
			ctr.Health = domain.HealthUnknown
		}
	} else {
		// Sem health check: running = healthy
		if info.State.Running {
			ctr.Health = domain.HealthHealthy
		} else {
			ctr.Health = domain.HealthUnknown
		}
	}

	return ctr, nil
}

// EnsureNetwork garante que a network do projeto existe
func (c *Client) EnsureNetwork(ctx context.Context, project string) (string, error) {
	networkName := c.networkName(project)

	// Verifica se já existe
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	})
	if err != nil {
		return "", fmt.Errorf("falha ao listar networks: %w", err)
	}

	if len(networks) > 0 {
		return networks[0].ID, nil
	}

	// Cria a network
	resp, err := c.cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			labels.Managed: "true",
			labels.Project: project,
		},
	})
	if err != nil {
		return "", fmt.Errorf("falha ao criar network: %w", err)
	}

	return resp.ID, nil
}

// RemoveNetwork remove a network de um projeto
func (c *Client) RemoveNetwork(ctx context.Context, project string) error {
	networkName := c.networkName(project)
	if err := c.cli.NetworkRemove(ctx, networkName); err != nil {
		return fmt.Errorf("falha ao remover network: %w", err)
	}
	return nil
}

// containerName gera o nome do container
func (c *Client) containerName(project, version string) string {
	return fmt.Sprintf("oi-%s-%s", project, version[:8])
}

// networkName gera o nome da network
func (c *Client) networkName(project string) string {
	return fmt.Sprintf("oi-%s-net", project)
}

// toDomainContainer converte types.Container para domain.Container
func (c *Client) toDomainContainer(ctr types.Container) domain.Container {
	name := ""
	if len(ctr.Names) > 0 {
		name = strings.TrimPrefix(ctr.Names[0], "/")
	}

	status := domain.StatusStopped
	if strings.HasPrefix(ctr.State, "running") {
		status = domain.StatusRunning
	}

	return domain.Container{
		ID:        ctr.ID,
		Name:      name,
		Project:   ctr.Labels[labels.Project],
		Version:   ctr.Labels[labels.Version],
		Image:     ctr.Image,
		Status:    status,
		CreatedAt: time.Unix(ctr.Created, 0),
	}
}

// parseCPU converte string de CPU para NanoCPUs
// Exemplo: "0.5" -> 500000000
func (c *Client) parseCPU(cpu string) int64 {
	if cpu == "" {
		return 0
	}
	var value float64
	fmt.Sscanf(cpu, "%f", &value)
	return int64(value * 1e9)
}

// parseMemory converte string de memória para bytes
// Exemplo: "512mb" -> 536870912
func (c *Client) parseMemory(mem string) int64 {
	if mem == "" {
		return 0
	}

	mem = strings.ToLower(mem)
	var value int64
	var unit string

	fmt.Sscanf(mem, "%d%s", &value, &unit)

	switch unit {
	case "kb", "k":
		return value * 1024
	case "mb", "m":
		return value * 1024 * 1024
	case "gb", "g":
		return value * 1024 * 1024 * 1024
	default:
		return value
	}
}

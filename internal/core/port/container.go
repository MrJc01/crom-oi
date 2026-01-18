package port

import (
	"context"
	"time"

	"github.com/crom-tech/oi/internal/core/domain"
)

// ContainerRuntime define as operações necessárias para gerenciar containers
// Abstraído do Docker SDK para facilitar testes e possível troca de runtime
type ContainerRuntime interface {
	// List retorna todos os containers gerenciados pelo OI para um projeto específico
	// Se project for vazio, retorna todos os containers OI
	List(ctx context.Context, project string) ([]domain.Container, error)

	// Pull baixa a imagem do registry
	Pull(ctx context.Context, image string) error

	// Create cria um novo container baseado na intenção
	// Create cria um novo container para a intenção
	Create(ctx context.Context, intent domain.Intent, version string, publishPort bool) (string, error)

	// Start inicia um container parado
	Start(ctx context.Context, containerID string) error

	// Stop para um container com graceful shutdown
	Stop(ctx context.Context, containerID string, timeout time.Duration) error

	// Remove remove um container
	Remove(ctx context.Context, containerID string, force bool) error

	// WaitHealthy aguarda o container ficar healthy ou timeout
	WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error

	// Inspect retorna informações detalhadas de um container
	Inspect(ctx context.Context, containerID string) (*domain.Container, error)

	// EnsureNetwork garante que a network do projeto existe
	EnsureNetwork(ctx context.Context, project string) (string, error)

	// RemoveNetwork remove a network de um projeto se não houver containers
	RemoveNetwork(ctx context.Context, project string) error
}

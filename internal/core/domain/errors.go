package domain

import "fmt"

// Erros de domínio
var (
	ErrInvalidPort = fmt.Errorf("porta inválida: deve estar entre 1 e 65535")
)

// ErrMissingField retorna erro para campo obrigatório ausente
func ErrMissingField(field string) error {
	return fmt.Errorf("campo obrigatório ausente: %s", field)
}

// ErrContainerNotFound indica que um container não foi encontrado
type ErrContainerNotFound struct {
	ID string
}

func (e ErrContainerNotFound) Error() string {
	return fmt.Sprintf("container não encontrado: %s", e.ID)
}

// ErrHealthCheckFailed indica falha no health check
type ErrHealthCheckFailed struct {
	ContainerID string
	Reason      string
}

func (e ErrHealthCheckFailed) Error() string {
	return fmt.Sprintf("health check falhou para container %s: %s", e.ContainerID, e.Reason)
}

// ErrDeployFailed indica falha no deploy
type ErrDeployFailed struct {
	Project string
	Reason  string
}

func (e ErrDeployFailed) Error() string {
	return fmt.Sprintf("deploy falhou para projeto %s: %s", e.Project, e.Reason)
}

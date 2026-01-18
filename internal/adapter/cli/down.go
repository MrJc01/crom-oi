package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/crom-tech/oi/internal/adapter/caddy"
	"github.com/crom-tech/oi/internal/adapter/docker"
	"github.com/crom-tech/oi/internal/config"
	"github.com/crom-tech/oi/internal/core/port"
	"github.com/crom-tech/oi/internal/core/service"
)

// NewDownCommand cria o comando "oi down"
func NewDownCommand() *cobra.Command {
	var path string
	var project string
	var noCaddy bool

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Remove todos os containers e recursos de um projeto",
		Long: `Para e remove todos os containers gerenciados pelo OI para um projeto.
Também remove a network isolada e as rotas do proxy.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determina o nome do projeto
			projectName := project
			if projectName == "" {
				// Tenta carregar do oi.json
				intent, err := config.LoadIntent(path)
				if err != nil {
					return fmt.Errorf("❌ Especifique --project ou tenha um oi.json válido")
				}
				projectName = intent.Nome
			}

			// Cria Docker client
			dockerClient, err := docker.NewClient()
			if err != nil {
				return fmt.Errorf("❌ Erro ao conectar com Docker: %w", err)
			}
			defer dockerClient.Close()

			// Cria Caddy manager (opcional)
			var proxyManager port.ProxyManager // Interface nil por padrão
			if !noCaddy {
				caddyManager := caddy.NewManager("")
				if err := caddyManager.Health(cmd.Context()); err == nil {
					proxyManager = caddyManager
				}
			}

			// Cria orchestrator
			orchestrator := service.NewOrchestrator(dockerClient, proxyManager)

			// Executa down
			return orchestrator.Down(cmd.Context(), projectName)
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json ou diretório")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Nome do projeto (sobrescreve oi.json)")
	cmd.Flags().BoolVar(&noCaddy, "no-caddy", false, "Desabilita integração com Caddy")

	return cmd
}

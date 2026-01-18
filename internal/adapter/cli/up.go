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

// NewUpCommand cria o comando "oi up"
func NewUpCommand() *cobra.Command {
	var path string
	var noCaddy bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Faz deploy do projeto baseado no oi.json",
		Long: `Lê o arquivo oi.json e garante que a realidade do servidor 
(Docker/Rede/SSL) corresponda exatamente à intenção descrita.

Usa Blue-Green deployment para zero-downtime.
Se o deploy falhar, mantém a versão anterior funcional.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Carrega intenção
			intent, err := config.LoadIntent(path)
			if err != nil {
				return fmt.Errorf("❌ Erro ao carregar oi.json: %w", err)
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
				// Testa conexão com Caddy
				if err := caddyManager.Health(cmd.Context()); err != nil {
					fmt.Printf("⚠️  Caddy não disponível, pulando configuração de proxy\n")
					// proxyManager permanece nil
				} else {
					proxyManager = caddyManager
				}
			}

			// Cria orchestrator
			orchestrator := service.NewOrchestrator(dockerClient, proxyManager)

			// Executa deploy
			return orchestrator.Up(cmd.Context(), *intent)
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json ou diretório contendo")
	cmd.Flags().BoolVar(&noCaddy, "no-caddy", false, "Desabilita integração com Caddy")

	return cmd
}

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crom-tech/oi/internal/adapter/docker"
	"github.com/crom-tech/oi/internal/config"
	"github.com/crom-tech/oi/internal/core/service"
)

// NewLogsCommand cria o comando "oi logs" (streaming)
func NewLogsCommand() *cobra.Command {
	var path string
	var project string
	var tail string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream de logs do container (ao vivo)",
		Long:  `Exibe e acompanha os logs do container do projeto. Similar ao 'docker logs -f'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(cmd.Context(), path, project, true, tail)
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Nome do projeto")
	cmd.Flags().StringVar(&tail, "tail", "all", "Número de linhas para mostrar")

	return cmd
}

// NewLogCommand cria o comando "oi log" (dump)
func NewLogCommand() *cobra.Command {
	var path string
	var project string
	var tail string

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Exibe logs do container (dump completo)",
		Long:  `Despeja todo o log do container e sai. Útil para grep ou análise rápida.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(cmd.Context(), path, project, false, tail)
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Nome do projeto")
	cmd.Flags().StringVar(&tail, "tail", "all", "Número de linhas para mostrar")

	return cmd
}

func runLogs(ctx context.Context, path, project string, follow bool, tail string) error {
	projectName := project
	if projectName == "" {
		intent, err := config.LoadIntent(path)
		if err != nil {
			return fmt.Errorf("❌ Especifique --project ou tenha um oi.json válido")
		}
		projectName = intent.Nome
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("❌ Erro ao conectar com Docker: %w", err)
	}
	defer dockerClient.Close()

	orchestrator := service.NewOrchestrator(dockerClient, nil)
	return orchestrator.Logs(ctx, projectName, os.Stdout, os.Stderr, follow, tail)
}

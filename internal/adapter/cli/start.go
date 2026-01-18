package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/crom-tech/oi/internal/adapter/docker"
	"github.com/crom-tech/oi/internal/config"
	"github.com/crom-tech/oi/internal/core/service"
)

// NewStartCommand cria o comando "oi start"
func NewStartCommand() *cobra.Command {
	var path string
	var project string
	var all bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Inicia containers parados",
		Long:  `Reinicia containers que foram parados com oi stop.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := ""
			if all {
				projectName = ""
			} else {
				projectName = project
				if projectName == "" {
					intent, err := config.LoadIntent(path)
					if err != nil {
						return fmt.Errorf("❌ Especifique --project, --all ou tenha um oi.json válido")
					}
					projectName = intent.Nome
				}
			}

			dockerClient, err := docker.NewClient()
			if err != nil {
				return fmt.Errorf("❌ Erro ao conectar com Docker: %w", err)
			}
			defer dockerClient.Close()

			orchestrator := service.NewOrchestrator(dockerClient, nil)
			return orchestrator.Start(cmd.Context(), projectName)
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Nome do projeto")
	cmd.Flags().BoolVar(&all, "all", false, "Inicia TODOS os containers OI")

	return cmd
}

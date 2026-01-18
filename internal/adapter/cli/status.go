package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/crom-tech/oi/internal/adapter/docker"
	"github.com/crom-tech/oi/internal/config"
	"github.com/crom-tech/oi/internal/core/service"
)

// NewStatusCommand cria o comando "oi status"
func NewStatusCommand() *cobra.Command {
	var path string
	var project string
	var all bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Mostra o estado atual dos containers",
		Long:  `Exibe informações sobre containers gerenciados pelo OI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determina o nome do projeto
			projectName := project
			if projectName == "" && !all {
				// Tenta carregar do oi.json
				intent, err := config.LoadIntent(path)
				if err == nil {
					projectName = intent.Nome
				}
			}

			// Cria Docker client
			dockerClient, err := docker.NewClient()
			if err != nil {
				return fmt.Errorf("❌ Erro ao conectar com Docker: %w", err)
			}
			defer dockerClient.Close()

			// Cria orchestrator (sem proxy para status)
			orchestrator := service.NewOrchestrator(dockerClient, nil)

			// Lista containers
			var filterProject string
			if !all {
				filterProject = projectName
			}
			containers, err := orchestrator.Status(cmd.Context(), filterProject)
			if err != nil {
				return fmt.Errorf("❌ Erro ao listar containers: %w", err)
			}

			if len(containers) == 0 {
				if all {
					fmt.Println("📭 Nenhum container OI em execução")
				} else {
					fmt.Printf("📭 Nenhum container encontrado para projeto '%s'\n", projectName)
				}
				return nil
			}

			// Formata tabela
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJETO\tNOME\tSTATUS\tHEALTH\tVERSÃO")
			fmt.Fprintln(w, "-------\t----\t------\t------\t------")

			for _, c := range containers {
				statusIcon := "⏸️"
				if c.Status == "running" {
					statusIcon = "▶️"
				}

				healthIcon := "❓"
				switch c.Health {
				case "healthy":
					healthIcon = "💚"
				case "unhealthy":
					healthIcon = "❤️"
				case "starting":
					healthIcon = "💛"
				}

				version := c.Version
				if len(version) > 8 {
					version = version[:8]
				}

				fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\t%s\n",
					c.Project,
					c.Name,
					statusIcon, c.Status,
					healthIcon,
					version,
				)
			}

			w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json ou diretório")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Nome do projeto")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Mostra todos os containers OI")

	return cmd
}

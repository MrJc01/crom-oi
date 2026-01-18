package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crom-tech/oi/internal/adapter/cli"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "oi",
		Short: "OI - Orquestrador de Intenção",
		Long: `OI é um orquestrador de containers focado em "Intenção" em vez de "Configuração".

O usuário não gerencia infraestrutura. Apenas dá um 'oi up'.
O OI lê o arquivo oi.json e garante que a realidade do servidor 
(Docker/Rede/SSL) corresponda exatamente à intenção descrita.`,
		Version: version,
	}

	// Adiciona comandos
	rootCmd.AddCommand(cli.NewUpCommand())
	rootCmd.AddCommand(cli.NewDownCommand())
	rootCmd.AddCommand(cli.NewStatusCommand())
	rootCmd.AddCommand(cli.NewStopCommand())
	rootCmd.AddCommand(cli.NewStartCommand())
	rootCmd.AddCommand(newInitCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newInitCommand cria o comando "oi init" para gerar um oi.json exemplo
func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [nome]",
		Short: "Cria um arquivo oi.json de exemplo",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nome := "meu-app"
			if len(args) > 0 {
				nome = args[0]
			}

			template := fmt.Sprintf(`{
  "nome": "%s",
  "origem": "docker.io/library/nginx:alpine",
  "dominio": "%s.localhost",
  "porta": 80,
  "recursos": {
    "cpu": "0.5",
    "memoria": "256mb"
  }
}
`, nome, nome)

			if err := os.WriteFile("oi.json", []byte(template), 0644); err != nil {
				return fmt.Errorf("❌ Erro ao criar oi.json: %w", err)
			}

			fmt.Printf("✅ Arquivo oi.json criado!\n")
			fmt.Printf("📝 Edite o arquivo e execute 'oi up' para fazer deploy.\n")
			return nil
		},
	}
}

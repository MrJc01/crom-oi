package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	rootCmd.AddCommand(cli.NewLogsCommand())
	rootCmd.AddCommand(cli.NewLogCommand())
	rootCmd.AddCommand(cli.NewInfoCommand(version))
	rootCmd.AddCommand(cli.NewUpdateCommand(version))
	rootCmd.AddCommand(newInitCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newInitCommand cria o comando "oi init" para gerar um oi.json exemplo
func newInitCommand() *cobra.Command {
	var dockerfile string

	cmd := &cobra.Command{
		Use:   "init [nome]",
		Short: "Cria um arquivo oi.json de exemplo",
		Long:  `Gera um arquivo de configuração oi.json. Pode ler um Dockerfile existente para extrair a porta (EXPOSE).`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nome := "meu-app"
			origem := "docker.io/library/nginx:alpine"
			porta := 80

			// Se Dockerfile especificado, tenta extrair informações
			if dockerfile != "" {
				fmt.Printf("🐳 Lendo Dockerfile '%s'...\n", dockerfile)

				// 1. Nome e Origem baseados no diretório atual
				cwd, err := os.Getwd()
				if err == nil {
					dirName := filepath.Base(cwd)
					nome = dirName
					origem = fmt.Sprintf("%s:latest", dirName)
				}

				// 2. Extrair porta do EXPOSE
				file, err := os.Open(dockerfile)
				if err != nil {
					return fmt.Errorf("❌ Erro ao ler Dockerfile: %w", err)
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					upperLine := strings.ToUpper(line)
					if strings.HasPrefix(upperLine, "EXPOSE") {
						parts := strings.Fields(line)
						if len(parts) >= 2 {
							var p int
							if _, err := fmt.Sscanf(parts[1], "%d", &p); err == nil {
								porta = p
								fmt.Printf("   ✅ Porta %d detectada\n", porta)
							}
						}
					}
				}
			}

			// Argumento CLI tem prioridade para o nome
			if len(args) > 0 {
				nome = args[0]
			}

			template := fmt.Sprintf(`{
  "nome": "%s",
  "origem": "%s",
  "dominio": "%s.localhost",
  "porta": %d,
  "recursos": {
    "cpu": "0.5",
    "memoria": "256mb"
  }
}
`, nome, origem, nome, porta)

			if err := os.WriteFile("oi.json", []byte(template), 0644); err != nil {
				return fmt.Errorf("❌ Erro ao criar oi.json: %w", err)
			}

			fmt.Printf("✅ Arquivo oi.json criado!\n")
			fmt.Printf("📝 Edite o arquivo e execute 'oi up' para fazer deploy.\n")
			return nil
		},
	}

	cmd.Flags().StringVarP(&dockerfile, "dockerfile", "d", "", "Caminho para um Dockerfile existente para importar config")

	return cmd
}

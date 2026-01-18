package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/crom-tech/oi/internal/adapter/caddy"
	"github.com/crom-tech/oi/internal/adapter/docker"
)

// NewInfoCommand cria o comando "oi info"
func NewInfoCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Exibe informações do sistema e ambiente",
		Long:  `Mostra detalhes sobre a instalação do OI, versões de dependências (Docker, Caddy) e saúde do sistema.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("📦 OI - Orquestrador de Intenção\n")
			fmt.Printf("   Versão: %s\n", version)
			fmt.Printf("   OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Println()

			// Check Docker
			fmt.Printf("🐳 Docker:\n")
			dockerClient, err := docker.NewClient()
			if err != nil {
				fmt.Printf("   ❌ Erro ao conectar: %v\n", err)
			} else {
				if err := dockerClient.Ping(cmd.Context()); err != nil {
					fmt.Printf("   ❌ Daemon não acessível: %v\n", err)
				} else {
					fmt.Printf("   ✅ Daemon acessível\n")

					// List managed networks
					nets, err := dockerClient.ListNetworks(cmd.Context())
					if err == nil {
						fmt.Printf("   🌐 Redes Gerenciadas: %d\n", len(nets))
					}
				}
				dockerClient.Close()
			}
			fmt.Println()

			// Check Caddy
			fmt.Printf("🔒 Caddy Proxy:\n")
			caddyManager := caddy.NewManager("")
			if err := caddyManager.Health(cmd.Context()); err != nil {
				fmt.Printf("   ⚠️  Caddy não detectado ou inacessível via API (:2019)\n")
				fmt.Printf("       (Isso é normal se você usa --no-caddy)\n")
			} else {
				fmt.Printf("   ✅ API acessível\n")
			}
			fmt.Println()

			// Check Config File
			if _, err := os.Stat("oi.json"); err == nil {
				fmt.Printf("📄 Arquivo oi.json detectado no diretório atual.\n")
			} else {
				fmt.Printf("📄 Nenhum oi.json no diretório atual.\n")
			}

			return nil
		},
	}
}

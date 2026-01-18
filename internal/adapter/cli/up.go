package cli

import (
	"fmt"
	"path/filepath"

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
	var live bool
	var all bool
	var filter string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Faz deploy do projeto baseado no oi.json",
		Long: `Lê o arquivo oi.json e garante que a realidade do servidor 
(Docker/Rede/SSL) corresponda exatamente à intenção descrita.

Usa Blue-Green deployment para zero-downtime.
Se o deploy falhar, mantém a versão anterior funcional.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var targetFiles []string

			// 1. Determina arquivos alvo
			if all {
				matches, err := filepath.Glob("*.json")
				if err != nil {
					return fmt.Errorf("erro ao buscar arquivos json: %w", err)
				}
				targetFiles = matches
			} else if filter != "" {
				matches, err := filepath.Glob(filter)
				if err != nil {
					return fmt.Errorf("erro ao filtrar arquivos (%s): %w", filter, err)
				}
				targetFiles = matches
			} else if len(args) > 0 {
				targetFiles = args
			} else {
				targetFiles = []string{path}
			}

			if len(targetFiles) == 0 {
				return fmt.Errorf("❌ Nenhum arquivo de configuração encontrado")
			}

			fmt.Printf("🎯 Processando %d arquivo(s)...\n", len(targetFiles))

			// Cria clientes (reutilizados para todos os deploys)
			dockerClient, err := docker.NewClient()
			if err != nil {
				return fmt.Errorf("❌ Erro ao conectar com Docker: %w", err)
			}
			defer dockerClient.Close()

			var proxyManager port.ProxyManager
			if !noCaddy {
				caddyManager := caddy.NewManager("")
				if err := caddyManager.Health(cmd.Context()); err != nil {
					fmt.Printf("⚠️  Caddy não disponível, pulando configuração de proxy\n")
				} else {
					proxyManager = caddyManager
				}
			}

			orchestrator := service.NewOrchestrator(dockerClient, proxyManager)

			// 2. Loop de execução
			var errs []error
			for _, p := range targetFiles {
				fmt.Printf("\n📂 Lendo configuração: %s\n", p)

				intent, err := config.LoadIntent(p)
				if err != nil {
					fmt.Printf("❌ Falha ao carregar %s: %v\n", p, err)
					errs = append(errs, err)
					continue
				}

				if err := orchestrator.Up(cmd.Context(), *intent, live); err != nil {
					fmt.Printf("❌ Falha no deploy de %s: %v\n", intent.Nome, err)
					errs = append(errs, err)
				} else {
					fmt.Printf("✅ Deploy de %s concluído!\n", intent.Nome)
				}
			}

			if len(errs) > 0 {
				return fmt.Errorf("ocorreram %d erros durante o processamento", len(errs))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "file", "f", ".", "Caminho para oi.json ou diretório contendo")
	cmd.Flags().BoolVar(&noCaddy, "no-caddy", false, "Desabilita integração com Caddy")
	cmd.Flags().BoolVar(&live, "live", false, "Habilita modo de desenvolvimento com volumes")
	cmd.Flags().BoolVar(&all, "all", false, "Processa todos os arquivos .json no diretório atual")
	cmd.Flags().StringVar(&filter, "filter", "", "Filtra arquivos por padrão glob (ex: 'data/oi-*.json')")

	return cmd
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/crom-tech/oi/internal/core/domain"
	"github.com/crom-tech/oi/internal/core/port"
)

// Orchestrator coordena o deploy de projetos usando Blue-Green strategy
type Orchestrator struct {
	runtime port.ContainerRuntime
	proxy   port.ProxyManager
}

// NewOrchestrator cria uma nova instância do Orchestrator
func NewOrchestrator(runtime port.ContainerRuntime, proxy port.ProxyManager) *Orchestrator {
	return &Orchestrator{
		runtime: runtime,
		proxy:   proxy,
	}
}

// Up realiza o deploy da intenção usando Blue-Green strategy
// Se falhar, mantém a versão anterior funcional (Zero-Downtime)
func (o *Orchestrator) Up(ctx context.Context, intent domain.Intent, live bool) error {
	// 0. Validação Fail-Fast: DNS
	if err := o.verifyDomain(intent.Dominio); err != nil {
		return err
	}

	// 0.1. Validação Fail-Fast: Proxy acessível
	if o.proxy != nil {
		fmt.Printf("🔍 Verificando conectividade com proxy...\n")
		if err := o.proxy.Health(ctx); err != nil {
			return fmt.Errorf("❌ Proxy (Caddy) não acessível. Verifique se está rodando: %w", err)
		}
	}

	// 1. Gerar version hash
	version := o.generateVersion(intent)

	fmt.Printf("🚀 Iniciando deploy de '%s' (versão %s)\n", intent.Nome, version[:8])

	// 2. Garantir network do projeto
	fmt.Printf("🌐 Criando/verificando network...\n")
	if _, err := o.runtime.EnsureNetwork(ctx, intent.Nome); err != nil {
		return fmt.Errorf("falha ao criar network: %w", err)
	}

	// 3. Listar containers atuais do projeto
	current, err := o.runtime.List(ctx, intent.Nome)
	if err != nil {
		return fmt.Errorf("falha ao listar containers: %w", err)
	}

	// 4. Baixar imagem
	// Se for live, talvez queremos garantir pull? Sim, imagem base ainda precisa.
	fmt.Printf("📦 Baixando imagem '%s'...\n", intent.Origem)
	if err := o.runtime.Pull(ctx, intent.Origem); err != nil {
		return fmt.Errorf("falha ao baixar imagem: %w", err)
	}

	// 5. Criar novo container (Blue-Green)
	fmt.Printf("🐳 Criando container...\n")

	// Se não tem proxy, publica a porta diretamente no host para acesso local
	publishPort := (o.proxy == nil)

	newID, err := o.runtime.Create(ctx, intent, version, publishPort, live)
	if err != nil {
		return fmt.Errorf("falha ao criar container: %w", err)
	}

	// 6. Iniciar container
	fmt.Printf("▶️  Iniciando container...\n")
	if err := o.runtime.Start(ctx, newID); err != nil {
		o.runtime.Remove(ctx, newID, true) // Cleanup do container criado
		return fmt.Errorf("falha ao iniciar container: %w", err)
	}

	// 7. Aguardar healthy (60 segundos de timeout)
	fmt.Printf("💓 Aguardando health check (max 60s)...\n")
	if err := o.runtime.WaitHealthy(ctx, newID, 60*time.Second); err != nil {
		fmt.Printf("❌ Health check falhou, rollback...\n")
		o.runtime.Stop(ctx, newID, 10*time.Second)
		o.runtime.Remove(ctx, newID, true)
		return domain.ErrDeployFailed{
			Project: intent.Nome,
			Reason:  fmt.Sprintf("health check falhou: %v", err),
		}
	}

	// 8. Obter informações do container para proxy
	container, err := o.runtime.Inspect(ctx, newID)
	if err != nil {
		return fmt.Errorf("falha ao inspecionar container: %w", err)
	}

	// 9. Atualizar proxy para novo container
	// 9. Atualizar proxy para novo container
	// 9. Atualizar proxy para novo container
	if o.proxy != nil {
		fmt.Printf("🔀 Configurando proxy para %s...\n", intent.Dominio)

		// Se porta for 0 (dinâmica), o container usa 80 internamente por padrão
		proxyPort := intent.Porta
		if proxyPort == 0 {
			proxyPort = 80
		}

		if err := o.proxy.AddRoute(ctx, intent.Dominio, container.Name, proxyPort); err != nil {
			// Não faz rollback aqui pois o container está healthy
			fmt.Printf("⚠️  Aviso: falha ao configurar proxy: %v\n", err)
		}
	}

	// 10. Remover containers antigos (graceful)
	if len(current) > 0 {
		fmt.Printf("🧹 Removendo %d container(s) antigo(s)...\n", len(current))
		for _, c := range current {
			if c.ID != newID {
				o.runtime.Stop(ctx, c.ID, 30*time.Second)
				o.runtime.Remove(ctx, c.ID, false)
			}
		}
	}

	// 11. Mensagem de sucesso com opções de acesso
	fmt.Printf("\n✅ Deploy completo!\n")

	// Porta para exibição (usar a real do container)
	displayPort := container.PublicPort
	if displayPort == 0 {
		displayPort = intent.Porta // Fallback
	}

	if strings.HasSuffix(intent.Dominio, ".localhost") {
		fmt.Printf("\n📡 Acesso local disponível:\n")
		fmt.Printf("   • http://127.0.0.1:%d\n", displayPort)
		fmt.Printf("   • http://localhost:%d\n", displayPort)
		fmt.Printf("   • http://%s:%d (requer /etc/hosts)\n", intent.Dominio, displayPort)
		if o.proxy != nil {
			fmt.Printf("   • https://%s (via Caddy, se configurado)\n", intent.Dominio)
		}
	} else {
		fmt.Printf("   Acesse: https://%s\n", intent.Dominio)
	}
	fmt.Println()
	return nil
}

// Down remove todos os containers e recursos de um projeto (ou todos se project == "")
func (o *Orchestrator) Down(ctx context.Context, project string) error {
	label := project
	if label == "" {
		label = "TODOS OS PROJETOS"
	}
	fmt.Printf("🛑 Parando recursos de '%s'...\n", label)

	// 1. Listar containers do projeto (ou todos)
	containers, err := o.runtime.List(ctx, project)
	if err != nil {
		return fmt.Errorf("falha ao listar containers: %w", err)
	}

	if len(containers) == 0 && project != "" {
		fmt.Printf("⚠️  Nenhum container encontrado para '%s'\n", project)
		// Se for projeto específico, tenta remover network mesmo assim
	}

	// 2. Parar e remover cada container
	for _, c := range containers {
		fmt.Printf("🐳 Parando container %s...\n", c.Name)
		o.runtime.Stop(ctx, c.ID, 30*time.Second)
		o.runtime.Remove(ctx, c.ID, false)
	}

	// 3. Remover rotas do proxy
	if o.proxy != nil {
		for _, c := range containers {
			if domain := c.Project; domain != "" {
				o.proxy.RemoveRoute(ctx, domain)
			}
		}
	}

	// 4. Remover networks
	// Se project == "", listar todas as networks gerenciadas e remover
	if project == "" {
		fmt.Printf("🌐 Removendo todas as networks OI...\n")
		projects, err := o.runtime.ListNetworks(ctx)
		if err != nil {
			fmt.Printf("⚠️  Aviso: falha ao listar networks: %v\n", err)
		} else {
			for _, p := range projects {
				if err := o.runtime.RemoveNetwork(ctx, p); err != nil {
					fmt.Printf("⚠️  Aviso: falha ao remover network do projeto %s: %v\n", p, err)
				}
			}
		}
	} else {
		// Projeto específico
		fmt.Printf("🌐 Removendo network...\n")
		if err := o.runtime.RemoveNetwork(ctx, project); err != nil {
			// Ignora erro se network não existir
		}
	}

	fmt.Printf("✅ Recursos de '%s' removidos com sucesso!\n", label)
	return nil
}

// Stop para containers de um projeto (ou todos)
func (o *Orchestrator) Stop(ctx context.Context, project string) error {
	containers, err := o.runtime.List(ctx, project)
	if err != nil {
		return fmt.Errorf("falha ao listar containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("⚠️  Nenhum container encontrado.")
		return nil
	}

	for _, c := range containers {
		if c.Status == domain.StatusRunning {
			fmt.Printf("🛑 Parando %s...\n", c.Name)
			if err := o.runtime.Stop(ctx, c.ID, 30*time.Second); err != nil {
				fmt.Printf("⚠️  Falha ao parar %s: %v\n", c.Name, err)
			}
		}
	}
	fmt.Println("✅ Containers parados.")
	return nil
}

// Start inicia containers de um projeto (ou todos)
func (o *Orchestrator) Start(ctx context.Context, project string) error {
	containers, err := o.runtime.List(ctx, project)
	if err != nil {
		return fmt.Errorf("falha ao listar containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("⚠️  Nenhum container encontrado.")
		return nil
	}

	for _, c := range containers {
		if c.Status != domain.StatusRunning {
			fmt.Printf("▶️  Iniciando %s...\n", c.Name)
			if err := o.runtime.Start(ctx, c.ID); err != nil {
				fmt.Printf("⚠️  Falha ao iniciar %s: %v\n", c.Name, err)
			}
		}
	}
	fmt.Println("✅ Containers iniciados.")
	return nil
}

// Status retorna o estado atual de um projeto
func (o *Orchestrator) Status(ctx context.Context, project string) ([]domain.Container, error) {
	return o.runtime.List(ctx, project)
}

// verifyDomain valida se o domínio está configurado corretamente
// Evita falhas silenciosas na emissão de SSL pelo Caddy
func (o *Orchestrator) verifyDomain(domain string) error {
	// Bypass para desenvolvimento local
	if strings.HasSuffix(domain, ".localhost") {
		return nil
	}

	// Lookup DNS para verificar se domínio aponta para algum servidor
	_, err := net.LookupHost(domain)
	if err != nil {
		return fmt.Errorf("❌ Domínio '%s' não aponta para este servidor. Configure o DNS antes de fazer deploy: %w", domain, err)
	}

	return nil
}

// generateVersion gera um hash único para a versão
func (o *Orchestrator) generateVersion(intent domain.Intent) string {
	data := fmt.Sprintf("%s-%s-%s-%d-%s",
		intent.Nome,
		intent.Origem,
		intent.Dominio,
		intent.Porta,
		time.Now().Format(time.RFC3339),
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Logs obtém o stream de logs do container principal do projeto
func (o *Orchestrator) Logs(ctx context.Context, project string, stdout, stderr io.Writer, follow bool, tail string) error {
	containers, err := o.runtime.List(ctx, project)
	if err != nil {
		return fmt.Errorf("falha ao listar containers: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("nenhum container encontrado para o projeto '%s'", project)
	}

	// Tenta encontrar um container rodando
	var targetID string
	var targetName string

	for _, c := range containers {
		if c.Status == domain.StatusRunning {
			targetID = c.ID
			targetName = c.Name
			break
		}
	}

	// Se nenhum estiver rodando, pega o primeiro da lista (mais recente geralmente)
	if targetID == "" {
		targetID = containers[0].ID
		targetName = containers[0].Name
	}

	fmt.Printf("📜 Exibindo logs de '%s'...\n", targetName)
	return o.runtime.Logs(ctx, targetID, stdout, stderr, follow, tail)
}

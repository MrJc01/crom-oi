package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

type release struct {
	TagName string `json:"tag_name"` // ex: v0.3.0
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// NewUpdateCommand cria o comando "oi update"
func NewUpdateCommand(currentVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"upgrade"},
		Short:   "Atualiza o OI para a versão mais recente",
		Long:    `Verifica, baixa e instala a última versão estável do OI. Mantém um backup das versões em ~/.oi/versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("🔍 Verificando atualizações (atual: %s)...\n", currentVersion)

			// 1. Check Latest Version
			latest, err := getLatestRelease()
			if err != nil {
				return fmt.Errorf("falha ao verificar versão: %w", err)
			}

			if latest.TagName == currentVersion {
				fmt.Printf("✅ Você já está na versão mais recente (%s).\n", currentVersion)
				return nil
			}

			fmt.Printf("🚀 Nova versão encontrada: %s\n", latest.TagName)

			// 2. Determine Asset URL
			targetName := fmt.Sprintf("oi-%s-%s", runtime.GOOS, runtime.GOARCH)
			downloadURL := ""
			for _, asset := range latest.Assets {
				if asset.Name == targetName {
					downloadURL = asset.BrowserDownloadURL
					break
				}
			}

			if downloadURL == "" {
				return fmt.Errorf("nenhum binário encontrado para %s/%s na versão %s", runtime.GOOS, runtime.GOARCH, latest.TagName)
			}

			// 3. Prepare Directories
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("falha ao obter diretório atual: %w", err)
			}

			versionDir := filepath.Join(cwd, "versions", latest.TagName)
			if err := os.MkdirAll(versionDir, 0755); err != nil {
				return fmt.Errorf("falha ao criar diretório de versão: %w", err)
			}

			destPath := filepath.Join(versionDir, "oi")

			// 4. Download
			fmt.Printf("⬇️  Baixando %s...\n", downloadURL)
			if err := downloadFile(downloadURL, destPath); err != nil {
				return fmt.Errorf("falha ao baixar atualização: %w", err)
			}

			if err := os.Chmod(destPath, 0755); err != nil {
				return fmt.Errorf("falha ao dar permissão de execução: %w", err)
			}
			fmt.Printf("📦 Versão arquivada em: %s\n", destPath)

			// 5. Update Current Binary
			exePath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("falha ao localizar binário atual: %w", err)
			}

			// Resolve symlinks se necessário
			exePath, err = filepath.EvalSymlinks(exePath)
			if err != nil {
				return fmt.Errorf("falha ao resolver symlinks: %w", err)
			}

			// Verifica permissão de escrita
			if err := checkWritePermission(filepath.Dir(exePath)); err != nil {
				fmt.Printf("⚠️  Não é possível escrever em %s (%v)\n", exePath, err)
				fmt.Printf("👉 Execute: sudo oi update\n")
				// Se não consegue instalar, pelo menos baixou
				return fmt.Errorf("permissão negada para atualizar binário do sistema")
			}

			fmt.Printf("🔄 Atualizando %s...\n", exePath)

			// Move backup da atual (opcional, mas seguro)
			backupPath := exePath + ".backup"
			_ = os.Rename(exePath, backupPath)

			// Copia nova versão para o local do executável
			if err := copyFile(destPath, exePath); err != nil {
				// Tenta restaurar backup
				_ = os.Rename(backupPath, exePath)
				return fmt.Errorf("falha ao instalar nova versão: %w", err)
			}
			if err := os.Chmod(exePath, 0755); err != nil {
				return fmt.Errorf("falha ao setar permissões: %w", err)
			}

			fmt.Printf("✅ OI atualizado com sucesso para %s!\n", latest.TagName)
			return nil
		},
	}
	return cmd
}

func getLatestRelease() (*release, error) {
	url := "https://api.github.com/repos/MrJc01/crom-oi/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api retornou status %d", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func downloadFile(url, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func checkWritePermission(dir string) error {
	// Tenta criar um arquivo temporário
	f, err := os.CreateTemp(dir, "oi-check")
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(f.Name())
	return nil
}

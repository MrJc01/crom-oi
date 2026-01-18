package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crom-tech/oi/internal/core/domain"
)

const (
	// DefaultFileName é o nome padrão do arquivo de intenção
	DefaultFileName = "oi.json"
)

// LoadIntent carrega e valida a intenção de um arquivo oi.json
func LoadIntent(path string) (*domain.Intent, error) {
	// Se path for diretório, procura oi.json dentro dele
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao acessar %s: %w", path, err)
	}

	if info.IsDir() {
		path = filepath.Join(path, DefaultFileName)
	}

	// Lê o arquivo
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler %s: %w", path, err)
	}

	// Parse JSON
	var intent domain.Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, fmt.Errorf("erro ao parsear JSON: %w", err)
	}

	// Normaliza campos (Inglês -> Português)
	intent.Normalize()

	// Valida campos obrigatórios
	if err := intent.Validate(); err != nil {
		return nil, err
	}

	return &intent, nil
}

// SaveIntent salva uma intenção em um arquivo oi.json
func SaveIntent(path string, intent *domain.Intent) error {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar intenção: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("erro ao salvar %s: %w", path, err)
	}

	return nil
}

// ExistsIntent verifica se existe um arquivo oi.json no caminho
func ExistsIntent(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		_, err = os.Stat(filepath.Join(path, DefaultFileName))
		return err == nil
	}

	return true
}

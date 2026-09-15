// Package config carrega a configuração do backend a partir de variáveis de
// ambiente — sem arquivo de configuração separado, conforme o escopo da v1.
package config

import (
	"fmt"
	"os"
)

// Config reúne toda a configuração necessária para subir o backend.
type Config struct {
	DatabaseURL string
	JWTSecret   string
	Porta       string
}

// Load lê a configuração das variáveis de ambiente do processo, retornando
// erro se alguma variável obrigatória estiver ausente.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Porta:       os.Getenv("PORT"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: variável de ambiente DATABASE_URL é obrigatória")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("config: variável de ambiente JWT_SECRET é obrigatória")
	}
	if cfg.Porta == "" {
		cfg.Porta = "8080"
	}

	return cfg, nil
}

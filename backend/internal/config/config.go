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
	SMTP        SMTP
}

// SMTP configura o envio de e-mail da recuperação de senha (FR-025). É
// opcional: sem host, a recuperação responde 503 em vez de falhar o boot.
type SMTP struct {
	Host      string
	Porta     string
	Usuario   string
	Senha     string
	Remetente string
}

// SMTPConfigurado informa se há servidor SMTP para enviar e-mails.
func (c Config) SMTPConfigurado() bool {
	return c.SMTP.Host != ""
}

// Load lê a configuração das variáveis de ambiente do processo, retornando
// erro se alguma variável obrigatória estiver ausente.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Porta:       os.Getenv("PORT"),
		SMTP: SMTP{
			Host:      os.Getenv("SMTP_HOST"),
			Porta:     os.Getenv("SMTP_PORTA"),
			Usuario:   os.Getenv("SMTP_USUARIO"),
			Senha:     os.Getenv("SMTP_SENHA"),
			Remetente: os.Getenv("SMTP_REMETENTE"),
		},
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
	if cfg.SMTP.Porta == "" {
		cfg.SMTP.Porta = "25"
	}
	if cfg.SMTP.Remetente == "" {
		cfg.SMTP.Remetente = "nao-responda@radiopx.local"
	}

	return cfg, nil
}

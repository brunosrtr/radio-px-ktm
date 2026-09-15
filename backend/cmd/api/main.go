package main

import (
	"context"
	"log"
	"net/http"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuração inválida: %v", err)
	}

	pool, err := storage.Conectar(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("erro ao conectar ao Postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.AplicarMigracoes(ctx); err != nil {
		log.Fatalf("erro ao aplicar migrações: %v", err)
	}

	roteador := transporte.NovoRoteador(cfg, pool)

	log.Printf("backend ouvindo na porta %s", cfg.Porta)
	if err := http.ListenAndServe(":"+cfg.Porta, roteador); err != nil {
		log.Fatalf("erro ao subir o servidor HTTP: %v", err)
	}
}

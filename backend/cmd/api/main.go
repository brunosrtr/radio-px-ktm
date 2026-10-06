package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/redelocal"

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

	if os.Getenv("LOCAL_DISCOVERY") == "true" {
		porta, err := strconv.Atoi(cfg.Porta)
		if err != nil || porta < 1 || porta > 65535 {
			log.Fatal("porta local inválida")
		}
		caminho := os.Getenv("LOCAL_ID_FILE")
		if caminho == "" {
			caminho = ".radio-px-server-id"
		}
		id, err := redelocal.Identidade(caminho)
		if err != nil {
			log.Fatalf("não foi possível guardar identidade local: %v", err)
		}
		local := &redelocal.Servidor{ID: id, Nome: "Rádio PX KTM", Porta: porta}
		roteador.Mount("/local", http.StripPrefix("/local", local.Handler()))
		go local.Anunciar(ctx)
		log.Printf("conectar celulares: http://localhost:%s/painel/conectar.html", cfg.Porta)
	}

	log.Printf("backend ouvindo na porta %s", cfg.Porta)
	if err := http.ListenAndServe(":"+cfg.Porta, roteador); err != nil {
		log.Fatalf("erro ao subir o servidor HTTP: %v", err)
	}
}

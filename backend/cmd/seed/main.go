// Comando seed cria uma empresa e três usuários de teste (um admin, dois
// motoristas) para exercitar o app e o painel manualmente em ambiente local,
// sem depender de telas de gestão ainda não implementadas.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/empresa"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

var contas = []struct {
	nome, login, senha, papel string
}{
	{"Admin Teste", "admin", "admin123", usuario.PapelAdmin},
	{"Motorista Um", "motorista1", "motorista123", usuario.PapelMotorista},
	{"Motorista Dois", "motorista2", "motorista123", usuario.PapelMotorista},
}

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

	empresas := empresa.NovoRepositorio(pool)
	usuarios := usuario.NovoRepositorio(pool)

	e, err := empresas.Criar(ctx, "Transportadora Teste Ltda", "00000000000100")
	if err != nil {
		log.Fatalf("erro ao criar empresa de teste: %v", err)
	}
	fmt.Printf("empresa criada: %s (id=%s)\n", e.RazaoSocial, e.ID)

	for _, c := range contas {
		hash, err := auth.HashSenha(c.senha)
		if err != nil {
			log.Fatalf("erro ao gerar hash de senha para %s: %v", c.login, err)
		}

		u, err := usuarios.Criar(ctx, usuario.Usuario{
			EmpresaID: e.ID,
			Nome:      c.nome,
			Login:     c.login,
			SenhaHash: hash,
			Papel:     c.papel,
		})
		if err != nil {
			log.Fatalf("erro ao criar usuário %s: %v", c.login, err)
		}
		fmt.Printf("usuário criado: login=%s senha=%s papel=%s id=%s\n", c.login, c.senha, c.papel, u.ID)
	}
}

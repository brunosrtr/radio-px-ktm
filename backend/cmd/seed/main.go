// Comando seed cria uma empresa e três usuários de teste (um admin, dois
// motoristas) para exercitar o app e o painel manualmente em ambiente local,
// sem depender de telas de gestão ainda não implementadas.
//
// Com -demo, cria em vez disso os dados da demonstração da Rede Orbit
// (roteiro-demo.md da 003-rede-orbit): contas admin_ktm e demo, três Orbits
// virtuais, quatro pórticos simulados em volta de -centro e dois canais
// públicos. Pode rodar várias vezes; -zerar volta a demo ao estado inicial.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/demo"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/empresa"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

var contas = []struct {
	nome, apelido, login, senha, papel string
}{
	{"Admin Teste", "Admin", "admin", "admin123", usuario.PapelAdmin},
	{"Motorista Um", "Motorista Um", "motorista1", "motorista123", usuario.PapelMotorista},
	{"Motorista Dois", "Motorista Dois", "motorista2", "motorista123", usuario.PapelMotorista},
}

func main() {
	modoDemo := flag.Bool("demo", false, "cria os dados da demonstração da Rede Orbit")
	centro := flag.String("centro", demo.CentroPadrao, "com -demo: lat,lng do local da apresentação (os pórticos simulados ficam em volta)")
	zerar := flag.Bool("zerar", false, "com -demo: apaga leituras, pórticos e contadores da demo antes de semear")
	flag.Parse()
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

	if *modoDemo {
		if err := semearDemo(ctx, pool, *centro, *zerar); err != nil {
			log.Fatalf("erro ao criar dados da demonstração: %v", err)
		}
		return
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
			Apelido:   c.apelido,
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

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/orbit"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// Contas de demonstração (valores de teste, só para ambiente local).
var contasDemo = []struct {
	nome, apelido, email, senha, papel string
}{
	{"Administração KTM", "KTM", "admin.ktm@demo.radiopx.local", "demo-ktm-2026", usuario.PapelAdminKTM},
	{"Celular Orbit virtual", "Orbit demo", "demo@demo.radiopx.local", "demo-orbit-2026", usuario.PapelDemo},
}

var orbitsDemo = []struct{ id, rotulo string }{
	{"ORBIT-0A1B2C", "Caminhão demo 1 · ABC1D23"},
	{"ORBIT-0A1B2D", "Caminhão demo 2 · KTM2A26"},
	{"ORBIT-0A1B2E", "Caminhão demo 3 · PXR4B07"},
}

// Pontos fixos fictícios ao longo da BR-116 (Régis Bittencourt), saindo de
// São Paulo — servem só para a comparação de cobertura na demonstração.
var pontosDemo = []orbit.PontoFixo{
	{Nome: "Pórtico simulado km 279 · Taboão da Serra", Latitude: -23.6280, Longitude: -46.7870, RaioMetros: 400, Simulado: true},
	{Nome: "Pórtico simulado km 290 · Embu das Artes", Latitude: -23.6650, Longitude: -46.8610, RaioMetros: 400, Simulado: true},
	{Nome: "Pórtico simulado km 300 · Itapecerica", Latitude: -23.7120, Longitude: -46.9180, RaioMetros: 400, Simulado: true},
	{Nome: "Pórtico simulado km 336 · São Lourenço", Latitude: -23.8530, Longitude: -47.0600, RaioMetros: 400, Simulado: true},
}

func semearDemo(ctx context.Context, pool *storage.Pool) error {
	usuarios := usuario.NovoRepositorio(pool)
	orbits := orbit.NovoRepositorio(pool)

	for _, c := range contasDemo {
		hash, err := auth.HashSenha(c.senha)
		if err != nil {
			return err
		}
		var id string
		err = pool.QueryRow(ctx, `
			insert into usuario (nome, apelido, email, senha_hash, papel)
			values ($1, $2, $3, $4, $5)
			on conflict (email) where email is not null do update set
				nome = excluded.nome, apelido = excluded.apelido,
				senha_hash = excluded.senha_hash, papel = excluded.papel,
				ativo = true, excluido_em = null
			returning id
		`, c.nome, c.apelido, c.email, hash, c.papel).Scan(&id)
		if err != nil {
			return fmt.Errorf("conta %s: %w", c.email, err)
		}
		for _, tipo := range []string{usuario.ConsentimentoTermos, usuario.ConsentimentoPontoTransmissao} {
			if err := usuarios.DefinirConsentimento(ctx, id, tipo, true); err != nil {
				return err
			}
		}
		fmt.Printf("conta %-9s e-mail=%s senha=%s\n", c.papel, c.email, c.senha)
	}

	for _, o := range orbitsDemo {
		if err := orbits.DefinirOrbit(ctx, o.id, true, o.rotulo); err != nil {
			return err
		}
		fmt.Printf("Orbit virtual %s (%s)\n", o.id, o.rotulo)
	}

	existentes, err := orbits.PontosFixos(ctx)
	if err != nil {
		return err
	}
	jaTem := map[string]orbit.PontoFixo{}
	for _, p := range existentes {
		jaTem[p.Nome] = p
	}
	var primeiro orbit.PontoFixo
	for i, p := range pontosDemo {
		criado, ok := jaTem[p.Nome]
		if !ok {
			novo, err := orbits.CriarPontoFixo(ctx, p)
			if err != nil {
				return err
			}
			criado = *novo
		}
		if i == 0 {
			primeiro = criado
		}
		fmt.Printf("ponto fixo %s\n", p.Nome)
	}

	// "Como é hoje": o primeiro Orbit só foi visto horas atrás, num pórtico.
	var temLeitura bool
	if err := pool.QueryRow(ctx, `select ultima_leitura_em is not null from orbit where id = $1`, orbitsDemo[0].id).Scan(&temLeitura); err != nil {
		return err
	}
	if !temLeitura {
		_, err := orbits.IngerirLote(ctx, orbit.OrigemFixo, primeiro.ID, "", []orbit.LeituraRecebida{{
			OrbitID: orbitsDemo[0].id, Latitude: primeiro.Latitude, Longitude: primeiro.Longitude,
			DetectadoEm: time.Now().Add(-3 * time.Hour),
		}})
		if err != nil {
			return err
		}
		fmt.Printf("%s: última leitura há 3 h no %s\n", orbitsDemo[0].id, primeiro.Nome)
	}
	fmt.Println("painel: http://localhost:8080/painel/rede.html")
	return nil
}

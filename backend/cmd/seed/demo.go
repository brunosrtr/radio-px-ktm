package main

import (
	"context"
	"fmt"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/demo"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/orbit"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

func semearDemo(ctx context.Context, pool *storage.Pool, centro string, zerar bool) error {
	latC, lngC, err := demo.LerCentro(centro)
	if err != nil {
		return err
	}
	usuarios := usuario.NovoRepositorio(pool)
	orbits := orbit.NovoRepositorio(pool)

	if zerar {
		if err := zerarDemo(ctx, pool); err != nil {
			return err
		}
		fmt.Println("demo zerada: leituras, pórticos simulados, canais da demo e contadores apagados")
	}

	ids := map[string]string{}
	for _, c := range demo.Contas {
		hash, err := auth.HashSenha(c.Senha)
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
		`, c.Nome, c.Apelido, c.Email, hash, c.Papel).Scan(&id)
		if err != nil {
			return fmt.Errorf("conta %s: %w", c.Email, err)
		}
		for _, tipo := range []string{usuario.ConsentimentoTermos, usuario.ConsentimentoPontoTransmissao} {
			if err := usuarios.DefinirConsentimento(ctx, id, tipo, true); err != nil {
				return err
			}
		}
		ids[c.Papel] = id
		fmt.Printf("conta %-9s e-mail=%s senha=%s\n", c.Papel, c.Email, c.Senha)
	}

	for _, o := range demo.Orbits {
		if err := orbits.DefinirOrbit(ctx, o.ID, true, o.Rotulo); err != nil {
			return err
		}
		fmt.Printf("Orbit virtual %s (%s)\n", o.ID, o.Rotulo)
	}

	porticos, err := garantirPorticos(ctx, orbits, latC, lngC)
	if err != nil {
		return err
	}

	if err := garantirCanais(ctx, pool, ids[usuario.PapelDemo], latC, lngC); err != nil {
		return err
	}

	// "Como é hoje": o caminhão 1 só foi visto horas atrás, no primeiro
	// pórtico.
	var temLeitura bool
	if err := pool.QueryRow(ctx, `select ultima_leitura_em is not null from orbit where id = $1`, demo.Orbits[0].ID).Scan(&temLeitura); err != nil {
		return err
	}
	if !temLeitura {
		primeiro := porticos[0]
		if _, err := orbits.IngerirLote(ctx, orbit.OrigemFixo, primeiro.ID, "", []orbit.LeituraRecebida{{
			OrbitID: demo.Orbits[0].ID, Latitude: primeiro.Latitude, Longitude: primeiro.Longitude,
			DetectadoEm: time.Now().Add(-3 * time.Hour),
		}}); err != nil {
			return err
		}
		fmt.Printf("%s: última leitura há 3 h no %s\n", demo.Orbits[0].ID, primeiro.Nome)
	}
	fmt.Println("painel: http://<ip-do-notebook>:8080/painel/rede.html")
	return nil
}

// zerarDemo devolve a demo ao estado inicial para um novo ensaio.
func zerarDemo(ctx context.Context, pool *storage.Pool) error {
	ids := make([]string, 0, len(demo.Orbits))
	for _, o := range demo.Orbits {
		ids = append(ids, o.ID)
	}
	emails := make([]string, 0, len(demo.Contas))
	for _, c := range demo.Contas {
		emails = append(emails, c.Email)
	}
	comandos := []struct {
		sql  string
		args []any
	}{
		{`delete from leitura_orbit where orbit_id = any($1)`, []any{ids}},
		{`update orbit set ultima_latitude = null, ultima_longitude = null, ultima_leitura_em = null where id = any($1)`, []any{ids}},
		{`delete from ponto_fixo where simulado`, nil},
		{`delete from contribuicao_diaria where usuario_id in (select id from usuario where email = any($1))`, []any{emails}},
		// Os canais da demo são recriados em volta do novo centro.
		{`update canal set ativo = false, codigo_convite = null
			where ativo and criador_id in (select id from usuario where email = any($1))`, []any{emails}},
	}
	for _, c := range comandos {
		if _, err := pool.Exec(ctx, c.sql, c.args...); err != nil {
			return fmt.Errorf("zerar demo: %w", err)
		}
	}
	return nil
}

// garantirPorticos cria os 4 pórticos simulados em volta do centro, se
// ainda não houver pórticos simulados.
func garantirPorticos(ctx context.Context, orbits *orbit.Repositorio, latC, lngC float64) ([]orbit.PontoFixo, error) {
	existentes, err := orbits.PontosFixos(ctx)
	if err != nil {
		return nil, err
	}
	var simulados []orbit.PontoFixo
	for _, p := range existentes {
		if p.Simulado {
			simulados = append(simulados, p)
		}
	}
	if len(simulados) > 0 {
		fmt.Printf("%d pórticos simulados já existem (use -zerar para recriar em outro centro)\n", len(simulados))
		return simulados, nil
	}
	for i, km := range demo.PorticosKm {
		lat, lng := demo.Deslocar(latC, lngC, km)
		criado, err := orbits.CriarPontoFixo(ctx, orbit.PontoFixo{
			Nome:       fmt.Sprintf("Pórtico simulado %c (%+.0f km)", 'A'+i, km),
			Latitude:   lat,
			Longitude:  lng,
			RaioMetros: 400,
			Simulado:   true,
		})
		if err != nil {
			return nil, err
		}
		simulados = append(simulados, *criado)
		fmt.Printf("ponto fixo %s\n", criado.Nome)
	}
	return simulados, nil
}

// garantirCanais cria os canais públicos da demo: um sem área (entra de
// qualquer lugar) e um com área em volta do centro.
func garantirCanais(ctx context.Context, pool *storage.Pool, criadorID string, latC, lngC float64) error {
	raio := 40_000
	canais := []struct {
		nome, descricao string
		area            bool
	}{
		{"Rádio KTM · Demo", "Canal aberto da demonstração", false},
		{"Trecho da demo", "Só para quem está perto da demonstração", true},
	}
	repo := canal.NovoRepositorio(pool)
	for _, c := range canais {
		var existe bool
		if err := pool.QueryRow(ctx, `
			select exists(select 1 from canal where nome = $1 and criador_id = $2 and ativo)
		`, c.nome, criadorID).Scan(&existe); err != nil {
			return err
		}
		if existe {
			continue
		}
		novo := canal.CanalDB{
			CriadorID: criadorID, Nome: c.nome, Descricao: c.descricao, TipoAcesso: "privado",
			Visibilidade: "publico", LimiteParticipantes: 20,
		}
		if c.area {
			lat, lng := latC, lngC
			novo.GeocercaAtiva, novo.CentroLatitude, novo.CentroLongitude, novo.RaioMetros = true, &lat, &lng, &raio
		}
		if _, err := repo.CriarCanal(ctx, novo); err != nil {
			return fmt.Errorf("canal %s: %w", c.nome, err)
		}
		fmt.Printf("canal público %q\n", c.nome)
	}
	return nil
}

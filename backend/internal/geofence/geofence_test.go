package geofence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func TestEntradaRecusadaForaDoRaio(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()

	empresaID := testutil.CriarEmpresa(t, pool)
	motoristaID := testutil.CriarUsuario(t, pool, empresaID, "motorista")

	repo := canal.NovoRepositorio(pool)
	servico := canal.NovoServico(repo, canal.NovoGerenciador(), posicao.NovoRepositorio(pool))

	raio := 500
	lat, lon := -28.4497, -52.2003
	canalDB, err := servico.Criar(ctx, empresaID, canal.DTOCanal{
		Nome: "Canal Geocerca", TipoAcesso: "privado", LimiteParticipantes: 10,
		GeocercaAtiva: true, CentroLatitude: &lat, CentroLongitude: &lon, RaioMetros: &raio,
	})
	if err != nil {
		t.Fatalf("erro ao criar canal com geocerca: %v", err)
	}

	if _, err := servico.ObterParaEntrada(ctx, canalDB.ID, empresaID, motoristaID); !errors.Is(err, canal.ErrForaDaArea) {
		t.Fatalf("sem posição registrada, esperava ErrForaDaArea, veio: %v", err)
	}

	inserirPosicaoAtual(t, pool, motoristaID, lat+1, lon+1)
	if _, err := servico.ObterParaEntrada(ctx, canalDB.ID, empresaID, motoristaID); !errors.Is(err, canal.ErrForaDaArea) {
		t.Fatalf("longe do centro, esperava ErrForaDaArea, veio: %v", err)
	}

	inserirPosicaoAtual(t, pool, motoristaID, lat, lon)
	if _, err := servico.ObterParaEntrada(ctx, canalDB.ID, empresaID, motoristaID); err != nil {
		t.Fatalf("dentro do raio, esperava entrada aceita, veio erro: %v", err)
	}
}

func TestVerificadorRemoveMotoristaAoSairDoRaio(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()

	empresaID := testutil.CriarEmpresa(t, pool)
	motoristaID := testutil.CriarUsuario(t, pool, empresaID, "motorista")

	repo := canal.NovoRepositorio(pool)
	gerenciador := canal.NovoGerenciador()
	servico := canal.NovoServico(repo, gerenciador, posicao.NovoRepositorio(pool))

	raio := 500
	lat, lon := -28.4497, -52.2003
	canalDB, err := servico.Criar(ctx, empresaID, canal.DTOCanal{
		Nome: "Canal Geocerca", TipoAcesso: "privado", LimiteParticipantes: 10,
		GeocercaAtiva: true, CentroLatitude: &lat, CentroLongitude: &lon, RaioMetros: &raio,
	})
	if err != nil {
		t.Fatalf("erro ao criar canal: %v", err)
	}

	inserirPosicaoAtual(t, pool, motoristaID, lat, lon)

	hub := gerenciador.ObterOuCriar(canalDB.ID, canalDB.LimiteParticipantes)
	membro, _, err := hub.Entrar(motoristaID, "Motorista Teste")
	if err != nil {
		t.Fatalf("erro ao entrar no hub: %v", err)
	}
	if err := repo.RegistrarEntrada(ctx, canalDB.ID, motoristaID); err != nil {
		t.Fatalf("erro ao registrar entrada: %v", err)
	}

	verificador := geofence.NovoVerificador(canal.NovoAdaptadorGeocerca(gerenciador, repo))

	// Posição ainda dentro do raio: não deve remover.
	verificador.VerificarPosicao(ctx, motoristaID, lat, lon)
	if !hub.TemMembro(motoristaID) {
		t.Fatal("não deveria ter sido removido enquanto dentro do raio")
	}

	// Posição fora do raio: deve remover, notificar e registrar o motivo.
	verificador.VerificarPosicao(ctx, motoristaID, lat+1, lon+1)

	if hub.TemMembro(motoristaID) {
		t.Fatal("deveria ter sido removido do canal ao sair do raio")
	}

	select {
	case evento := <-membro.Eventos:
		if evento.Tipo != "removido_canal" || evento.Dados["motivo"] != "geocerca" {
			t.Fatalf("esperava removido_canal com motivo geocerca, veio: %+v", evento)
		}
	default:
		t.Fatal("esperava evento removido_canal no canal do membro")
	}

	var motivo string
	err = pool.QueryRow(ctx, `
		select motivo_saida from participacao_canal
		where canal_id = $1 and usuario_id = $2 and saiu_em is not null
	`, canalDB.ID, motoristaID).Scan(&motivo)
	if err != nil {
		t.Fatalf("erro ao verificar participacao_canal: %v", err)
	}
	if motivo != "geocerca" {
		t.Fatalf("esperava motivo_saida = geocerca, veio %q", motivo)
	}
}

func inserirPosicaoAtual(t *testing.T, pool *storage.Pool, usuarioID string, lat, lon float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into posicao_atual (usuario_id, latitude, longitude, capturado_em)
		values ($1, $2, $3, now())
		on conflict (usuario_id) do update set
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			capturado_em = excluded.capturado_em
	`, usuarioID, lat, lon)
	if err != nil {
		t.Fatalf("erro ao inserir posição atual de teste: %v", err)
	}
}

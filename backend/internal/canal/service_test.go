package canal

import (
	"context"
	"errors"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func TestCriarCanalValidaLimiteEGeocerca(t *testing.T) {
	pool := testutil.AbrirPool(t)
	empresaID := testutil.CriarEmpresa(t, pool)

	servico := NovoServico(NovoRepositorio(pool), NovoGerenciador())
	ctx := context.Background()

	raio := 1000
	lat, lon := -28.45, -52.20

	casos := []struct {
		nome     string
		dto      DTOCanal
		querErro bool
	}{
		{
			nome:     "limite_participantes fora do conjunto permitido",
			dto:      DTOCanal{Nome: "Canal A", TipoAcesso: "privado", LimiteParticipantes: 7},
			querErro: true,
		},
		{
			nome:     "geocerca ativa sem centro nem raio",
			dto:      DTOCanal{Nome: "Canal B", TipoAcesso: "privado", LimiteParticipantes: 10, GeocercaAtiva: true},
			querErro: true,
		},
		{
			nome: "geocerca ativa com centro e raio completos",
			dto: DTOCanal{
				Nome: "Canal C", TipoAcesso: "privado", LimiteParticipantes: 10,
				GeocercaAtiva: true, CentroLatitude: &lat, CentroLongitude: &lon, RaioMetros: &raio,
			},
			querErro: false,
		},
		{
			nome:     "canal simples válido",
			dto:      DTOCanal{Nome: "Canal D", TipoAcesso: "privado", LimiteParticipantes: 20},
			querErro: false,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			_, err := servico.Criar(ctx, empresaID, c.dto)
			if c.querErro && !errors.Is(err, ErrCanalInvalido) {
				t.Fatalf("esperava ErrCanalInvalido, veio: %v", err)
			}
			if !c.querErro && err != nil {
				t.Fatalf("não esperava erro, veio: %v", err)
			}
		})
	}
}

func TestListarParaMotoristaFiltraPorAutorizacaoDeEmpresa(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()

	empresaA := testutil.CriarEmpresa(t, pool)
	empresaB := testutil.CriarEmpresa(t, pool)
	empresaC := testutil.CriarEmpresa(t, pool)
	motoristaA := testutil.CriarUsuario(t, pool, empresaA, "motorista")

	servico := NovoServico(NovoRepositorio(pool), NovoGerenciador())

	canalPrivadoA, err := servico.Criar(ctx, empresaA, DTOCanal{
		Nome: "Privado A", TipoAcesso: "privado", LimiteParticipantes: 10,
	})
	if err != nil {
		t.Fatalf("erro ao criar canal privado de A: %v", err)
	}

	canalCompartilhadoB, err := servico.Criar(ctx, empresaB, DTOCanal{
		Nome: "Compartilhado B", TipoAcesso: "compartilhado", LimiteParticipantes: 10,
	})
	if err != nil {
		t.Fatalf("erro ao criar canal compartilhado de B: %v", err)
	}
	if err := servico.LiberarEmpresa(ctx, canalCompartilhadoB.ID, empresaB, empresaA); err != nil {
		t.Fatalf("erro ao liberar empresa A no canal de B: %v", err)
	}

	canalPrivadoC, err := servico.Criar(ctx, empresaC, DTOCanal{
		Nome: "Privado C", TipoAcesso: "privado", LimiteParticipantes: 10,
	})
	if err != nil {
		t.Fatalf("erro ao criar canal privado de C: %v", err)
	}

	listados, err := servico.ListarParaMotorista(ctx, empresaA, motoristaA)
	if err != nil {
		t.Fatalf("erro ao listar canais para o motorista: %v", err)
	}

	presentes := make(map[string]bool, len(listados))
	for _, c := range listados {
		presentes[c.ID] = true
	}

	if !presentes[canalPrivadoA.ID] {
		t.Errorf("esperava que o canal privado da própria empresa (%s) aparecesse na listagem", canalPrivadoA.ID)
	}
	if !presentes[canalCompartilhadoB.ID] {
		t.Errorf("esperava que o canal compartilhado liberado (%s) aparecesse na listagem", canalCompartilhadoB.ID)
	}
	if presentes[canalPrivadoC.ID] {
		t.Errorf("canal privado de empresa não autorizada (%s) não deveria aparecer na listagem", canalPrivadoC.ID)
	}
}

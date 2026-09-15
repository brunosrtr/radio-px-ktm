package posicao

import (
	"context"
	"fmt"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
)

// Servico implementa a ingestão de lote de posições (RF-024, RF-025,
// RNF-015).
type Servico struct {
	repositorio *Repositorio
	verificador *geofence.Verificador
}

// NovoServico cria o Servico de posições. verificador pode ser nil quando
// não há necessidade de reavaliar geocercas (ex.: testes isolados do
// histórico de posição).
func NovoServico(repositorio *Repositorio, verificador *geofence.Verificador) *Servico {
	return &Servico{repositorio: repositorio, verificador: verificador}
}

// IngerirLote valida e insere um lote de posições, e reavalia a geocerca dos
// canais em que o motorista está conectado com base no ponto mais recente do
// lote (US3).
func (s *Servico) IngerirLote(ctx context.Context, usuarioID string, pontos []Ponto) error {
	if len(pontos) == 0 {
		return fmt.Errorf("%w: lote vazio", ErrLoteInvalido)
	}

	maisRecente := pontos[0]
	for _, p := range pontos {
		if p.CapturadoEm.IsZero() {
			return fmt.Errorf("%w: capturado_em é obrigatório em todos os pontos", ErrLoteInvalido)
		}
		if p.CapturadoEm.After(maisRecente.CapturadoEm) {
			maisRecente = p
		}
	}

	if err := s.repositorio.InserirLote(ctx, usuarioID, pontos); err != nil {
		return err
	}

	if s.verificador != nil {
		s.verificador.VerificarPosicao(ctx, usuarioID, maisRecente.Latitude, maisRecente.Longitude)
	}

	return nil
}

// PosicoesAtuaisPorEmpresa retorna a posição atual dos motoristas de
// empresaAlvoID, mas só quando ela é a mesma empresa do solicitante — nunca
// a de outra empresa (FR-026).
func (s *Servico) PosicoesAtuaisPorEmpresa(ctx context.Context, empresaSolicitanteID, empresaAlvoID string) ([]MotoristaPosicaoAtual, error) {
	if empresaSolicitanteID != empresaAlvoID {
		return nil, ErrSemPermissao
	}
	return s.repositorio.PosicoesAtuaisPorEmpresa(ctx, empresaAlvoID)
}

// TrajetoDoMotorista retorna o trajeto de um motorista no período
// informado, desde que ele pertença à mesma empresa do administrador
// autenticado (FR-027).
func (s *Servico) TrajetoDoMotorista(ctx context.Context, empresaAdminID, motoristaID string, de, ate time.Time) ([]PontoTrajeto, error) {
	pertence, err := s.repositorio.MotoristaPertenceAEmpresa(ctx, motoristaID, empresaAdminID)
	if err != nil {
		return nil, err
	}
	if !pertence {
		return nil, ErrSemPermissao
	}
	return s.repositorio.Trajeto(ctx, motoristaID, de, ate)
}

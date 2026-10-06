package canal

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
)

// ErrCanalInvalido indica que o corpo de criação/edição de canal não passa
// nas validações (limite de participantes ou geocerca incompleta).
var ErrCanalInvalido = errors.New("canal: dados inválidos")

// ErrSemPermissao indica que a empresa do usuário não tem acesso ao canal.
var ErrSemPermissao = errors.New("canal: sem permissão")

// ErrForaDaArea indica que a última posição conhecida do motorista está fora
// do raio de geocerca do canal (ou que ele não tem posição registrada).
var ErrForaDaArea = errors.New("canal: fora da área do canal")

var limitesValidos = map[int]bool{5: true, 10: true, 15: true, 20: true}

// DTOCanal reúne os campos editáveis de um canal, usados tanto na criação
// quanto na edição (contracts/rest-api.md).
type DTOCanal struct {
	Nome                string
	Descricao           string
	TipoAcesso          string
	LimiteParticipantes int
	GeocercaAtiva       bool
	CentroLatitude      *float64
	CentroLongitude     *float64
	RaioMetros          *int
}

// CanalListado é um canal como visto por um motorista em GET /canais.
type CanalListado struct {
	CanalDB
	ParticipantesAtual int
	Silenciado         bool
}

// Servico implementa as regras de negócio de gestão de canais e controle de
// acesso por empresa (FR-009 a FR-013, FR-019, FR-022, RNF12).
type Servico struct {
	repositorio *Repositorio
	gerenciador *Gerenciador
	posicoes    *posicao.Repositorio
}

// NovoServico cria o Servico de canais sobre o repositório, o gerenciador do
// hub em memória e o repositório de posições (para a checagem de geocerca)
// informados.
func NovoServico(repositorio *Repositorio, gerenciador *Gerenciador, posicoes *posicao.Repositorio) *Servico {
	return &Servico{repositorio: repositorio, gerenciador: gerenciador, posicoes: posicoes}
}

func validarDTO(dto DTOCanal) error {
	if dto.Nome == "" {
		return fmt.Errorf("%w: nome é obrigatório", ErrCanalInvalido)
	}
	if !limitesValidos[dto.LimiteParticipantes] {
		return fmt.Errorf("%w: limite_participantes deve ser 5, 10, 15 ou 20", ErrCanalInvalido)
	}
	if dto.TipoAcesso != "privado" && dto.TipoAcesso != "compartilhado" {
		return fmt.Errorf("%w: tipo_acesso deve ser 'privado' ou 'compartilhado'", ErrCanalInvalido)
	}
	if dto.GeocercaAtiva && (dto.CentroLatitude == nil || dto.CentroLongitude == nil || dto.RaioMetros == nil) {
		return fmt.Errorf("%w: geocerca ativa exige centro_latitude, centro_longitude e raio_metros", ErrCanalInvalido)
	}
	if dto.GeocercaAtiva {
		lat, lon := *dto.CentroLatitude, *dto.CentroLongitude
		if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 || math.IsNaN(lon) || math.IsInf(lon, 0) || lon < -180 || lon > 180 || *dto.RaioMetros < 1 || *dto.RaioMetros > 20000000 {
			return fmt.Errorf("%w: centro geográfico ou raio inválido", ErrCanalInvalido)
		}
	}
	return nil
}

// Criar valida e persiste um novo canal da empresa do administrador
// autenticado.
func (s *Servico) Criar(ctx context.Context, empresaID string, dto DTOCanal) (*CanalDB, error) {
	if err := validarDTO(dto); err != nil {
		return nil, err
	}
	return s.repositorio.CriarCanal(ctx, CanalDB{
		EmpresaID:           empresaID,
		Nome:                dto.Nome,
		Descricao:           dto.Descricao,
		TipoAcesso:          dto.TipoAcesso,
		LimiteParticipantes: dto.LimiteParticipantes,
		GeocercaAtiva:       dto.GeocercaAtiva,
		CentroLatitude:      dto.CentroLatitude,
		CentroLongitude:     dto.CentroLongitude,
		RaioMetros:          dto.RaioMetros,
	})
}

// Editar aplica as alterações de dto a um canal existente, checando que ele
// pertence à empresa do administrador autenticado.
func (s *Servico) Editar(ctx context.Context, canalID, empresaID string, dto DTOCanal) (*CanalDB, error) {
	if err := validarDTO(dto); err != nil {
		return nil, err
	}

	atual, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return nil, err
	}
	if atual.EmpresaID != empresaID {
		return nil, ErrSemPermissao
	}

	atualizado, err := s.repositorio.AtualizarCanal(ctx, CanalDB{
		ID:                  canalID,
		Nome:                dto.Nome,
		Descricao:           dto.Descricao,
		TipoAcesso:          dto.TipoAcesso,
		LimiteParticipantes: dto.LimiteParticipantes,
		GeocercaAtiva:       dto.GeocercaAtiva,
		CentroLatitude:      dto.CentroLatitude,
		CentroLongitude:     dto.CentroLongitude,
		RaioMetros:          dto.RaioMetros,
	})
	if err != nil {
		return nil, err
	}

	if hub, ok := s.gerenciador.Obter(canalID); ok {
		hub.DefinirLimiteParticipantes(atualizado.LimiteParticipantes)
		// Uma redução do raio vale também para motoristas já conectados.
		if atualizado.GeocercaAtiva {
			for _, membro := range hub.EstadoParaCentral().Participantes {
				ponto, err := s.posicoes.ObterAtual(ctx, membro.UsuarioID)
				if err != nil || !geofence.DentroDoRaio(ponto.Latitude, ponto.Longitude, *atualizado.CentroLatitude, *atualizado.CentroLongitude, *atualizado.RaioMetros) {
					hub.Remover(membro.UsuarioID, Evento{Tipo: "removido_canal", Dados: map[string]any{"canal_id": canalID, "motivo": "geocerca"}})
					if err := s.repositorio.RegistrarSaida(ctx, canalID, membro.UsuarioID, "geocerca"); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return atualizado, nil
}

// LiberarEmpresa concede a uma empresa parceira acesso a um canal
// compartilhado, checando que o canal pertence à empresa do administrador.
func (s *Servico) LiberarEmpresa(ctx context.Context, canalID, empresaDonaID, empresaParceiraID string) error {
	canal, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return err
	}
	if canal.EmpresaID != empresaDonaID {
		return ErrSemPermissao
	}
	return s.repositorio.LiberarEmpresa(ctx, canalID, empresaParceiraID)
}

// RevogarEmpresa remove o acesso de uma empresa parceira a um canal.
func (s *Servico) RevogarEmpresa(ctx context.Context, canalID, empresaDonaID, empresaParceiraID string) error {
	canal, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return err
	}
	if canal.EmpresaID != empresaDonaID {
		return ErrSemPermissao
	}
	return s.repositorio.RevogarEmpresa(ctx, canalID, empresaParceiraID)
}

// DefinirPreferencia silencia ou reativa um canal para o usuário, sem tirá-lo
// dele, e reflete a mudança na conexão ativa (se houver) no hub em memória.
func (s *Servico) DefinirPreferencia(ctx context.Context, usuarioID, canalID string, silenciado bool) error {
	if err := s.repositorio.DefinirPreferencia(ctx, usuarioID, canalID, silenciado); err != nil {
		return err
	}
	if hub, ok := s.gerenciador.Obter(canalID); ok {
		hub.DefinirSilenciado(usuarioID, silenciado)
	}
	return nil
}

// ObterParaEntrada busca um canal, confirma que a empresa informada tem
// autorização de acesso e, se a geocerca estiver ativa, que a última posição
// conhecida do motorista está dentro do raio (FR-014/FR-015) — usado pelo
// handler WebSocket em entrar_canal.
func (s *Servico) ObterParaEntrada(ctx context.Context, canalID, empresaID, usuarioID string) (*CanalDB, error) {
	c, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return nil, err
	}

	autorizado := c.EmpresaID == empresaID
	if !autorizado {
		autorizado, err = s.repositorio.EstaAutorizada(ctx, canalID, empresaID)
		if err != nil {
			return nil, err
		}
	}
	if !autorizado {
		return nil, ErrSemPermissao
	}

	if c.GeocercaAtiva {
		pos, err := s.posicoes.ObterAtual(ctx, usuarioID)
		if err != nil {
			return nil, ErrForaDaArea
		}
		if !geofence.DentroDoRaio(pos.Latitude, pos.Longitude, *c.CentroLatitude, *c.CentroLongitude, *c.RaioMetros) {
			return nil, ErrForaDaArea
		}
	}

	return c, nil
}

// ListarParaMotorista retorna os canais autorizados para a empresa do
// motorista, com o total de participantes atuais (do hub em memória), se o
// próprio motorista o silenciou (RNF12), e já excluindo canais com geocerca
// ativa fora do alcance da última posição conhecida do motorista (FR-014,
// FR-015, FR-029).
func (s *Servico) ListarParaMotorista(ctx context.Context, empresaID, usuarioID string) ([]CanalListado, error) {
	canais, err := s.repositorio.ListarAutorizados(ctx, empresaID)
	if err != nil {
		return nil, err
	}

	preferencias, err := s.repositorio.PreferenciasSilenciadas(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	posicaoAtual, err := s.posicoes.ObterAtual(ctx, usuarioID)
	if err != nil && !errors.Is(err, posicao.ErrNaoEncontrada) {
		return nil, err
	}

	listados := make([]CanalListado, 0, len(canais))
	for _, c := range canais {
		if c.GeocercaAtiva {
			if posicaoAtual == nil {
				continue
			}
			if !geofence.DentroDoRaio(posicaoAtual.Latitude, posicaoAtual.Longitude, *c.CentroLatitude, *c.CentroLongitude, *c.RaioMetros) {
				continue
			}
		}

		participantesAtual := 0
		if hub, ok := s.gerenciador.Obter(c.ID); ok {
			participantesAtual = hub.Participantes()
		}
		listados = append(listados, CanalListado{
			CanalDB:            c,
			ParticipantesAtual: participantesAtual,
			Silenciado:         preferencias[c.ID],
		})
	}
	return listados, nil
}

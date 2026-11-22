package canal

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
)

// Autor é quem pede uma ação sobre canais. AdminEmpresa = administrador de
// transportadora (canais de empresa da 001); os demais criam canais da
// comunidade.
type Autor struct {
	UsuarioID    string
	EmpresaID    string
	AdminEmpresa bool
}

// ErrNaoPodeRemoverSiMesmo impede o criador de se banir do próprio canal.
var ErrNaoPodeRemoverSiMesmo = errors.New("canal: o criador não pode se remover")

// DistanciaPerto é o quanto além da borda da área um canal ainda conta como
// "perto de mim" na aba Explorar.
const DistanciaPerto = 50_000.0

// podeGerenciar: o criador de um canal da comunidade ou um admin da empresa
// dona do canal.
func podeGerenciar(c *CanalDB, autor Autor) bool {
	if c.Comunidade() {
		return c.CriadorID == autor.UsuarioID
	}
	return autor.AdminEmpresa && autor.EmpresaID != "" && c.EmpresaID == autor.EmpresaID
}

func validarVisibilidade(v string) error {
	if v != "publico" && v != "privado" {
		return fmt.Errorf("%w: visibilidade deve ser 'publico' ou 'privado'", ErrCanalInvalido)
	}
	return nil
}

// CriarComo cria um canal em nome do autor: de empresa, se ele administra
// uma (regra da 001), ou da comunidade (FR-029). Canais privados da
// comunidade já nascem com código de convite.
func (s *Servico) CriarComo(ctx context.Context, autor Autor, dto DTOCanal) (*CanalDB, error) {
	if autor.AdminEmpresa && autor.EmpresaID != "" {
		if err := validarDTO(dto); err != nil {
			return nil, err
		}
		return s.repositorio.CriarCanal(ctx, s.canalDoDTO(dto, autor.EmpresaID, autor.UsuarioID))
	}

	dto.TipoAcesso = "privado"
	if err := validarDTO(dto); err != nil {
		return nil, err
	}
	if err := validarVisibilidade(dto.Visibilidade); err != nil {
		return nil, err
	}
	for range 5 {
		novo := s.canalDoDTO(dto, "", autor.UsuarioID)
		novo.Visibilidade = dto.Visibilidade
		if dto.Visibilidade == "privado" {
			codigo, err := GerarCodigoConvite()
			if err != nil {
				return nil, err
			}
			novo.CodigoConvite = codigo
		}
		criado, err := s.repositorio.CriarCanal(ctx, novo)
		if errors.Is(err, ErrConviteRepetido) {
			continue
		}
		return criado, err
	}
	return nil, ErrConviteRepetido
}

func (s *Servico) canalDoDTO(dto DTOCanal, empresaID, criadorID string) CanalDB {
	return CanalDB{
		EmpresaID:           empresaID,
		CriadorID:           criadorID,
		Nome:                dto.Nome,
		Descricao:           dto.Descricao,
		TipoAcesso:          dto.TipoAcesso,
		LimiteParticipantes: dto.LimiteParticipantes,
		GeocercaAtiva:       dto.GeocercaAtiva,
		CentroLatitude:      dto.CentroLatitude,
		CentroLongitude:     dto.CentroLongitude,
		RaioMetros:          dto.RaioMetros,
	}
}

// EditarComo altera um canal; só o criador (comunidade) ou um admin da
// empresa dona (FR-031). Tornar um canal privado gera convite; torná-lo
// público apaga o convite.
func (s *Servico) EditarComo(ctx context.Context, canalID string, autor Autor, dto DTOCanal) (*CanalDB, error) {
	atual, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return nil, err
	}
	if !podeGerenciar(atual, autor) {
		return nil, ErrSemPermissao
	}
	if !atual.Comunidade() {
		return s.Editar(ctx, canalID, autor.EmpresaID, dto)
	}
	dto.TipoAcesso = "privado"
	if dto.Visibilidade == "" {
		dto.Visibilidade = atual.Visibilidade
	}
	if err := validarDTO(dto); err != nil {
		return nil, err
	}
	if err := validarVisibilidade(dto.Visibilidade); err != nil {
		return nil, err
	}
	codigo := ""
	if dto.Visibilidade == "privado" {
		codigo = atual.CodigoConvite
		if codigo == "" {
			if codigo, err = GerarCodigoConvite(); err != nil {
				return nil, err
			}
		}
	}
	editado := s.canalDoDTO(dto, "", atual.CriadorID)
	editado.ID = canalID
	editado.Visibilidade = dto.Visibilidade
	editado.CodigoConvite = codigo
	atualizado, err := s.repositorio.AtualizarCanal(ctx, editado)
	if err != nil {
		return nil, err
	}
	if hub, ok := s.gerenciador.Obter(canalID); ok {
		hub.DefinirLimiteParticipantes(atualizado.LimiteParticipantes)
	}
	return atualizado, nil
}

// Apagar desativa o canal e desconecta quem estiver nele, avisando o motivo.
// Nenhum áudio é preservado: o hub descarta fila e buffers (Princípio I).
func (s *Servico) Apagar(ctx context.Context, canalID string, autor Autor) error {
	c, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return err
	}
	if !podeGerenciar(c, autor) {
		return ErrSemPermissao
	}
	if err := s.repositorio.Apagar(ctx, canalID); err != nil {
		return err
	}
	if hub, ok := s.gerenciador.Obter(canalID); ok {
		for _, membro := range hub.EstadoParaCentral().Participantes {
			hub.Remover(membro.UsuarioID, Evento{Tipo: "removido_canal", Dados: map[string]any{"canal_id": canalID, "motivo": "canal_apagado"}})
			if err := s.repositorio.RegistrarSaida(ctx, canalID, membro.UsuarioID, "encerramento"); err != nil {
				return err
			}
		}
	}
	return nil
}

// NovoConvite troca o código do canal privado; o antigo deixa de funcionar,
// mas quem já entrou continua com acesso (FR-031).
func (s *Servico) NovoConvite(ctx context.Context, canalID string, autor Autor) (string, error) {
	c, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return "", err
	}
	if !c.Comunidade() || !podeGerenciar(c, autor) {
		return "", ErrSemPermissao
	}
	if c.Publico() {
		return "", fmt.Errorf("%w: canal público não usa convite", ErrCanalInvalido)
	}
	for range 5 {
		codigo, err := GerarCodigoConvite()
		if err != nil {
			return "", err
		}
		err = s.repositorio.DefinirConvite(ctx, canalID, codigo)
		if errors.Is(err, ErrConviteRepetido) {
			continue
		}
		return codigo, err
	}
	return "", ErrConviteRepetido
}

// EntrarPorConvite libera o canal privado do código para o usuário.
func (s *Servico) EntrarPorConvite(ctx context.Context, codigoDigitado, usuarioID string) (*CanalDB, error) {
	codigo := NormalizarCodigoConvite(codigoDigitado)
	if codigo == "" {
		return nil, ErrNaoEncontrado
	}
	c, err := s.repositorio.ObterPorConvite(ctx, codigo)
	if err != nil {
		return nil, err
	}
	situacao, err := s.repositorio.SituacaoAcesso(ctx, c.ID, usuarioID)
	if err != nil {
		return nil, err
	}
	if situacao == SituacaoBanido {
		return nil, ErrBanido
	}
	if c.CriadorID != usuarioID {
		if err := s.repositorio.LiberarAcesso(ctx, c.ID, usuarioID); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Remover tira um participante do canal e impede a volta (FR-031): ele sai
// na hora, recebe o aviso e não entra mais nem por convite.
func (s *Servico) Remover(ctx context.Context, canalID string, autor Autor, alvoID string) error {
	c, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return err
	}
	if !podeGerenciar(c, autor) {
		return ErrSemPermissao
	}
	if alvoID == autor.UsuarioID || alvoID == c.CriadorID {
		return ErrNaoPodeRemoverSiMesmo
	}
	if err := s.repositorio.Banir(ctx, canalID, alvoID); err != nil {
		return err
	}
	if hub, ok := s.gerenciador.Obter(canalID); ok {
		hub.Remover(alvoID, Evento{Tipo: "removido_canal", Dados: map[string]any{"canal_id": canalID, "motivo": "removido"}})
	}
	return s.repositorio.RegistrarSaida(ctx, canalID, alvoID, "removido")
}

// Participantes lista quem está conectado agora, para o criador escolher quem
// remover.
func (s *Servico) Participantes(ctx context.Context, canalID string, autor Autor) ([]ParticipanteConectado, error) {
	c, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return nil, err
	}
	if !podeGerenciar(c, autor) {
		return nil, ErrSemPermissao
	}
	if hub, ok := s.gerenciador.Obter(canalID); ok {
		return hub.EstadoParaCentral().Participantes, nil
	}
	return []ParticipanteConectado{}, nil
}

// Favoritar marca ou desmarca um canal que o usuário pode ver (FR-032).
func (s *Servico) Favoritar(ctx context.Context, canalID string, autor Autor, favorito bool) error {
	if _, err := s.Detalhar(ctx, canalID, autor); err != nil {
		return err
	}
	return s.repositorio.DefinirFavorito(ctx, autor.UsuarioID, canalID, favorito)
}

// DetalheCanal é o canal visto por um usuário que pode vê-lo.
type DetalheCanal struct {
	CanalDB
	SouCriador         bool
	PodeGerenciar      bool
	Favorito           bool
	ParticipantesAtual int
}

// Detalhar devolve o canal se o usuário pode vê-lo (público, criador,
// convite ou empresa), sem considerar geocerca ou lotação.
func (s *Servico) Detalhar(ctx context.Context, canalID string, autor Autor) (*DetalheCanal, error) {
	c, err := s.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return nil, err
	}
	acesso, err := s.montarAcesso(ctx, c, autor.EmpresaID, autor.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := PodeEntrar(acesso); err != nil {
		return nil, err
	}
	favorito, err := s.repositorio.EhFavorito(ctx, autor.UsuarioID, canalID)
	if err != nil {
		return nil, err
	}
	return &DetalheCanal{
		CanalDB:            *c,
		SouCriador:         acesso.EhCriador,
		PodeGerenciar:      podeGerenciar(c, autor),
		Favorito:           favorito,
		ParticipantesAtual: s.participantesAgora(canalID),
	}, nil
}

// CanalExplorado é um canal público na aba Explorar.
type CanalExplorado struct {
	CanalDB
	Favorito           bool
	ParticipantesAtual int
	// DistanciaMetros até a borda da área do canal (0 = você está dentro);
	// nil quando o canal não tem área ou a posição não foi informada.
	DistanciaMetros *float64
}

// Explorar lista canais públicos da comunidade (FR-030). Com busca, filtra
// pelo nome. Com perto = true e posição, só os canais com área a até
// DistanciaPerto, do mais próximo ao mais distante; senão, os mais cheios
// primeiro.
func (s *Servico) Explorar(ctx context.Context, usuarioID, busca string, lat, lng *float64, perto bool) ([]CanalExplorado, error) {
	canais, err := s.repositorio.ListarPublicos(ctx, usuarioID, busca)
	if err != nil {
		return nil, err
	}
	resultado := make([]CanalExplorado, 0, len(canais))
	for _, c := range canais {
		e := CanalExplorado{CanalDB: c.CanalDB, Favorito: c.Favorito, ParticipantesAtual: s.participantesAgora(c.ID)}
		if c.GeocercaAtiva && lat != nil && lng != nil {
			d := math.Max(0, geofence.Distancia(*lat, *lng, *c.CentroLatitude, *c.CentroLongitude)-float64(*c.RaioMetros))
			e.DistanciaMetros = &d
		}
		if perto && (e.DistanciaMetros == nil || *e.DistanciaMetros > DistanciaPerto) {
			continue
		}
		resultado = append(resultado, e)
	}
	sort.SliceStable(resultado, func(i, j int) bool {
		if perto {
			return *resultado[i].DistanciaMetros < *resultado[j].DistanciaMetros
		}
		return resultado[i].ParticipantesAtual > resultado[j].ParticipantesAtual
	})
	return resultado, nil
}

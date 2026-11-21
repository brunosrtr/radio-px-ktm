package orbit

import (
	"context"
	"log"
	"time"
)

// Servico orquestra a ingestão de leituras, os pontos ativos e o
// encaminhamento para as integrações (painel e, quando houver, a KTM).
type Servico struct {
	repositorio  *Repositorio
	pontosAtivos *PontosAtivos
	integracoes  []Integracao
	segredo      string
	agora        func() time.Time
}

// NovoServico cria o serviço. segredo assina o ponto_id rotativo.
func NovoServico(repositorio *Repositorio, pontosAtivos *PontosAtivos, segredo string, integracoes ...Integracao) *Servico {
	return &Servico{repositorio: repositorio, pontosAtivos: pontosAtivos, integracoes: integracoes, segredo: segredo, agora: time.Now}
}

// Repositorio expõe o repositório para consultas do painel.
func (s *Servico) Repositorio() *Repositorio { return s.repositorio }

// PontosAtivos expõe o registro de pontos ativos.
func (s *Servico) PontosAtivos() *PontosAtivos { return s.pontosAtivos }

// IngerirLeituras valida e grava um lote vindo do celular do usuário. O
// usuário vira apenas um ponto_id do dia; ele só é usado de novo para somar
// a própria contribuição, numa tabela que o painel não lê.
func (s *Servico) IngerirLeituras(ctx context.Context, usuarioID string, leituras []LeituraRecebida) (ResultadoIngestao, error) {
	agora := s.agora()
	if err := ValidarLote(leituras, agora); err != nil {
		return ResultadoIngestao{}, err
	}
	pontoID := PontoID(s.segredo, usuarioID, agora)
	resultado, err := s.repositorio.IngerirLote(ctx, OrigemCelular, pontoID, usuarioID, leituras)
	if err != nil {
		return resultado, err
	}
	s.pontosAtivos.Registrar(pontoID)
	s.encaminhar(resultado)
	return resultado, nil
}

// RegistrarSinal marca o ponto de transmissão do usuário como ativo
// (POST /ponto/sinal, FR-018).
func (s *Servico) RegistrarSinal(usuarioID string) {
	s.pontosAtivos.Registrar(PontoID(s.segredo, usuarioID, s.agora()))
}

// SimularPassagem gera uma leitura de origem "fixo" do Orbit no ponto fixo
// informado — usado na demonstração para mostrar como é a cobertura hoje.
func (s *Servico) SimularPassagem(ctx context.Context, pontoFixoID, orbitID string) (ResultadoIngestao, error) {
	ponto, err := s.repositorio.ObterPontoFixo(ctx, pontoFixoID)
	if err != nil {
		return ResultadoIngestao{}, err
	}
	leitura := []LeituraRecebida{{OrbitID: orbitID, Latitude: ponto.Latitude, Longitude: ponto.Longitude, DetectadoEm: s.agora()}}
	if err := ValidarLote(leitura, s.agora()); err != nil {
		return ResultadoIngestao{}, err
	}
	resultado, err := s.repositorio.IngerirLote(ctx, OrigemFixo, ponto.ID, "", leitura)
	if err != nil {
		return resultado, err
	}
	s.encaminhar(resultado)
	return resultado, nil
}

// Contribuicao devolve os contadores do usuário.
func (s *Servico) Contribuicao(ctx context.Context, usuarioID string) (hoje, semana, total int, err error) {
	return s.repositorio.Contribuicao(ctx, usuarioID, s.agora())
}

// encaminhar entrega as leituras novas às integrações em segundo plano. Uma
// falha aqui nunca derruba o lote: as leituras já estão gravadas.
func (s *Servico) encaminhar(resultado ResultadoIngestao) {
	if len(resultado.Novas) == 0 {
		return
	}
	for _, integracao := range s.integracoes {
		go func(i Integracao) {
			ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelar()
			if err := i.Encaminhar(ctx, resultado.Novas, resultado.OrbitsAtualizados); err != nil {
				log.Printf("orbit: falha ao encaminhar %d leituras: %v", len(resultado.Novas), err)
			}
		}(integracao)
	}
}

package usuario

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Tipos de consentimento (FR-028, Princípio VII).
const (
	ConsentimentoTermos           = "termos"
	ConsentimentoPontoTransmissao = "ponto_transmissao"
)

// VersaoPontoTransmissaoAtual é a versão do texto que explica o que o
// celular faz como ponto de transmissão.
const VersaoPontoTransmissaoAtual = "2026-11"

// ErrConsentimentoInvalido indica um tipo de consentimento desconhecido ou
// uma tentativa de revogar os termos (para isso, exclui-se a conta).
var ErrConsentimentoInvalido = errors.New("usuario: consentimento inválido")

// Consentimento é um aceite vigente.
type Consentimento struct {
	Tipo     string    `json:"tipo"`
	Versao   string    `json:"versao"`
	AceitoEm time.Time `json:"aceito_em"`
}

func versaoAtual(tipo string) (string, error) {
	switch tipo {
	case ConsentimentoTermos:
		return VersaoTermosAtual, nil
	case ConsentimentoPontoTransmissao:
		return VersaoPontoTransmissaoAtual, nil
	}
	return "", ErrConsentimentoInvalido
}

// DefinirConsentimento registra (aceito = true) ou revoga (aceito = false) o
// consentimento do tipo informado, na versão vigente do texto. Revogar os
// termos não é permitido: quem não concorda mais exclui a conta (FR-026).
func (r *Repositorio) DefinirConsentimento(ctx context.Context, usuarioID, tipo string, aceito bool) error {
	versao, err := versaoAtual(tipo)
	if err != nil {
		return err
	}
	if !aceito && tipo == ConsentimentoTermos {
		return ErrConsentimentoInvalido
	}

	if aceito {
		_, err = r.pool.Exec(ctx, `
			insert into consentimento (usuario_id, tipo, versao) values ($1, $2, $3)
			on conflict (usuario_id, tipo, versao) do update
				set aceito_em = now(), revogado_em = null
		`, usuarioID, tipo, versao)
	} else {
		_, err = r.pool.Exec(ctx, `
			update consentimento set revogado_em = now()
			where usuario_id = $1 and tipo = $2 and revogado_em is null
		`, usuarioID, tipo)
	}
	if err != nil {
		return fmt.Errorf("usuario: erro ao definir consentimento %s: %w", tipo, err)
	}
	return nil
}

// ConsentimentosVigentes devolve, por tipo, o aceite mais recente não
// revogado do usuário.
func (r *Repositorio) ConsentimentosVigentes(ctx context.Context, usuarioID string) ([]Consentimento, error) {
	linhas, err := r.pool.Query(ctx, `
		select distinct on (tipo) tipo, versao, aceito_em
		from consentimento
		where usuario_id = $1 and revogado_em is null
		order by tipo, aceito_em desc
	`, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao buscar consentimentos: %w", err)
	}
	defer linhas.Close()

	vigentes := []Consentimento{}
	for linhas.Next() {
		var c Consentimento
		if err := linhas.Scan(&c.Tipo, &c.Versao, &c.AceitoEm); err != nil {
			return nil, fmt.Errorf("usuario: erro ao ler consentimento: %w", err)
		}
		vigentes = append(vigentes, c)
	}
	return vigentes, linhas.Err()
}

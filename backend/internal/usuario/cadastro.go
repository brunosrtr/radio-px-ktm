package usuario

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrCadastroInvalido     = errors.New("cadastro inválido")
	ErrCadastroDuplicado    = errors.New("CPF ou placa já cadastrado")
	ErrVeiculoNaoAutorizado = errors.New("selecione um veículo da sua empresa")
	formatoCPF              = regexp.MustCompile(`^[0-9.\-\s]+$`)
	formatoPlaca            = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z0-9][0-9]{2}$`)
)

type Veiculo struct {
	ID        string `json:"id"`
	Descricao string `json:"descricao"`
	Placa     string `json:"placa"`
}
type DadosVeiculo struct {
	Descricao string `json:"descricao"`
	Placa     string `json:"placa"`
}
type DadosMotorista struct {
	Nome             string        `json:"nome"`
	Sobrenome        string        `json:"sobrenome"`
	CPF              string        `json:"cpf"`
	Senha            string        `json:"senha"`
	ConfirmacaoSenha string        `json:"confirmacao_senha"`
	VeiculoID        string        `json:"veiculo_id"`
	NovoVeiculo      *DadosVeiculo `json:"novo_veiculo"`
}
type MotoristaCadastrado struct {
	ID           string   `json:"id"`
	Nome         string   `json:"nome"`
	PrimeiroNome string   `json:"primeiro_nome"`
	Sobrenome    string   `json:"sobrenome"`
	CPF          string   `json:"cpf"`
	Login        string   `json:"login"`
	Veiculo      *Veiculo `json:"veiculo"`
}

// NormalizarCPF aceita dígitos ou a apresentação usual com pontos e hífen.
func NormalizarCPF(valor string) string {
	if !formatoCPF.MatchString(valor) {
		return ""
	}
	var resultado strings.Builder
	for _, c := range valor {
		if c >= '0' && c <= '9' {
			resultado.WriteRune(c)
		}
	}
	return resultado.String()
}
func NormalizarLogin(valor string) string {
	valor = strings.TrimSpace(valor)
	if cpf := NormalizarCPF(valor); len(cpf) == 11 {
		return cpf
	}
	return valor
}
func cpfValido(cpf string) bool {
	if len(cpf) != 11 || cpf == strings.Repeat(cpf[:1], 11) {
		return false
	}
	for n := 9; n <= 10; n++ {
		soma := 0
		for i := 0; i < n; i++ {
			soma += int(cpf[i]-'0') * (n + 1 - i)
		}
		digito := (soma * 10) % 11
		if digito == 10 {
			digito = 0
		}
		if int(cpf[n]-'0') != digito {
			return false
		}
	}
	return true
}
func validarVeiculo(d DadosVeiculo) (DadosVeiculo, error) {
	d.Descricao = strings.TrimSpace(d.Descricao)
	d.Placa = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(d.Placa), "-", ""), " ", ""))
	if d.Descricao == "" || utf8.RuneCountInString(d.Descricao) > 100 {
		return d, fmt.Errorf("%w: informe o nome/modelo do veículo (até 100 caracteres)", ErrCadastroInvalido)
	}
	if !formatoPlaca.MatchString(d.Placa) {
		return d, fmt.Errorf("%w: informe uma placa válida, como ABC1234 ou ABC1D23", ErrCadastroInvalido)
	}
	return d, nil
}
func erroCadastro(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrCadastroDuplicado
	}
	return err
}
func (r *Repositorio) ListarVeiculos(ctx context.Context, empresa string) ([]Veiculo, error) {
	linhas, err := r.pool.Query(ctx, `select id,descricao,placa from veiculo where empresa_id=$1 order by descricao,placa`, empresa)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	veiculos := []Veiculo{}
	for linhas.Next() {
		var v Veiculo
		if err := linhas.Scan(&v.ID, &v.Descricao, &v.Placa); err != nil {
			return nil, err
		}
		veiculos = append(veiculos, v)
	}
	return veiculos, linhas.Err()
}
func (r *Repositorio) CriarVeiculo(ctx context.Context, empresa string, d DadosVeiculo) (*Veiculo, error) {
	d, err := validarVeiculo(d)
	if err != nil {
		return nil, err
	}
	v := Veiculo{Descricao: d.Descricao, Placa: d.Placa}
	err = r.pool.QueryRow(ctx, `insert into veiculo(empresa_id,descricao,placa) values($1,$2,$3) returning id`, empresa, d.Descricao, d.Placa).Scan(&v.ID)
	if err != nil {
		return nil, erroCadastro(err)
	}
	return &v, nil
}
func (r *Repositorio) CriarMotorista(ctx context.Context, empresa string, d DadosMotorista) (*MotoristaCadastrado, error) {
	d.Nome = strings.TrimSpace(d.Nome)
	d.Sobrenome = strings.TrimSpace(d.Sobrenome)
	d.CPF = NormalizarCPF(d.CPF)
	if d.Nome == "" || d.Sobrenome == "" || utf8.RuneCountInString(d.Nome) > 80 || utf8.RuneCountInString(d.Sobrenome) > 120 {
		return nil, fmt.Errorf("%w: informe nome e sobrenome", ErrCadastroInvalido)
	}
	if !cpfValido(d.CPF) {
		return nil, fmt.Errorf("%w: CPF inválido", ErrCadastroInvalido)
	}
	if utf8.RuneCountInString(d.Senha) < 8 || len(d.Senha) > 72 {
		return nil, fmt.Errorf("%w: a senha deve ter pelo menos 8 caracteres e no máximo 72 bytes", ErrCadastroInvalido)
	}
	if d.Senha != d.ConfirmacaoSenha {
		return nil, fmt.Errorf("%w: as senhas não coincidem", ErrCadastroInvalido)
	}
	if (d.VeiculoID == "") == (d.NovoVeiculo == nil) {
		return nil, fmt.Errorf("%w: selecione um veículo ou cadastre um novo", ErrCadastroInvalido)
	}
	if d.NovoVeiculo != nil {
		validado, err := validarVeiculo(*d.NovoVeiculo)
		if err != nil {
			return nil, err
		}
		d.NovoVeiculo = &validado
	}
	hash, err := auth.HashSenha(d.Senha)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	v := &Veiculo{}
	if d.NovoVeiculo != nil {
		v.Descricao = d.NovoVeiculo.Descricao
		v.Placa = d.NovoVeiculo.Placa
		err = tx.QueryRow(ctx, `insert into veiculo(empresa_id,descricao,placa) values($1,$2,$3) returning id`, empresa, v.Descricao, v.Placa).Scan(&v.ID)
	} else {
		err = tx.QueryRow(ctx, `select id,descricao,placa from veiculo where empresa_id=$1 and id::text=$2`, empresa, d.VeiculoID).Scan(&v.ID, &v.Descricao, &v.Placa)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVeiculoNaoAutorizado
		}
	}
	if err != nil {
		return nil, erroCadastro(err)
	}
	m := MotoristaCadastrado{Nome: d.Nome + " " + d.Sobrenome, PrimeiroNome: d.Nome, Sobrenome: d.Sobrenome, CPF: d.CPF, Login: d.CPF, Veiculo: v}
	err = tx.QueryRow(ctx, `insert into usuario(empresa_id,nome,primeiro_nome,sobrenome,cpf,login,senha_hash,papel,veiculo_id) values($1,$2,$3,$4,$5,$6,$7,'motorista',$8) returning id`, empresa, m.Nome, m.PrimeiroNome, m.Sobrenome, m.CPF, m.Login, hash, v.ID).Scan(&m.ID)
	if err != nil {
		return nil, erroCadastro(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &m, nil
}
func (r *Repositorio) ListarMotoristas(ctx context.Context, empresa string) ([]MotoristaCadastrado, error) {
	linhas, err := r.pool.Query(ctx, `select u.id,u.nome,coalesce(u.primeiro_nome,''),coalesce(u.sobrenome,''),coalesce(u.cpf,''),u.login,v.id,v.descricao,v.placa from usuario u left join veiculo v on v.id=u.veiculo_id and v.empresa_id=u.empresa_id where u.empresa_id=$1 and u.papel='motorista' and u.ativo order by u.nome`, empresa)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	lista := []MotoristaCadastrado{}
	for linhas.Next() {
		var m MotoristaCadastrado
		var id, descricao, placa *string
		if err := linhas.Scan(&m.ID, &m.Nome, &m.PrimeiroNome, &m.Sobrenome, &m.CPF, &m.Login, &id, &descricao, &placa); err != nil {
			return nil, err
		}
		if id != nil {
			m.Veiculo = &Veiculo{ID: *id, Descricao: *descricao, Placa: *placa}
		}
		lista = append(lista, m)
	}
	return lista, linhas.Err()
}

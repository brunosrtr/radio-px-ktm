package transporte_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/email"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

type clienteTeste struct {
	t   *testing.T
	url string
}

func (c clienteTeste) chamar(metodo, rota, token string, dados any, esperado int) map[string]any {
	c.t.Helper()
	var corpo []byte
	if dados != nil {
		corpo, _ = json.Marshal(dados)
	}
	req, _ := http.NewRequest(metodo, c.url+rota, bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	conteudo, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != esperado {
		c.t.Fatalf("%s %s: status %d, esperado %d; %s", metodo, rota, resp.StatusCode, esperado, conteudo)
	}
	resultado := map[string]any{}
	if len(conteudo) > 0 && conteudo[0] == '{' {
		if err := json.Unmarshal(conteudo, &resultado); err != nil {
			c.t.Fatal(err)
		}
	}
	return resultado
}

func emailNovo() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "contrato-" + hex.EncodeToString(b) + "@exemplo.com"
}

func TestContratoContasAbertas(t *testing.T) {
	pool := testutil.AbrirPool(t)
	enviador := &email.EnviadorMemoria{}
	servidor := httptest.NewServer(transporte.NovoRoteadorCom(
		config.Config{JWTSecret: "contas-teste"}, pool,
		transporte.Dependencias{Enviador: enviador, LimitadorContas: transporte.NovoLimitador(1000, time.Minute)},
	))
	defer servidor.Close()
	c := clienteTeste{t: t, url: servidor.URL}

	endereco := emailNovo()
	cadastro := map[string]any{
		"nome": "Maria Souza", "apelido": "Maria da Serra", "email": endereco,
		"senha": "senha-forte-1", "aceitou_termos_versao": usuario.VersaoTermosAtual,
	}

	criada := c.chamar("POST", "/auth/cadastro", "", cadastro, http.StatusCreated)
	token, _ := criada["token"].(string)
	conta, _ := criada["usuario"].(map[string]any)
	if token == "" || conta["papel"] != "usuario" || conta["apelido"] != "Maria da Serra" || conta["tem_empresa"] != false {
		t.Fatalf("resposta do cadastro fora do contrato: %v", criada)
	}

	t.Run("cadastro duplicado e inválido", func(t *testing.T) {
		if r := c.chamar("POST", "/auth/cadastro", "", cadastro, http.StatusConflict); r["codigo"] != "email_em_uso" {
			t.Fatalf("esperava email_em_uso: %v", r)
		}
		invalido := map[string]any{"nome": "X", "apelido": "X", "email": emailNovo(), "senha": "curta", "aceitou_termos_versao": usuario.VersaoTermosAtual}
		if r := c.chamar("POST", "/auth/cadastro", "", invalido, http.StatusUnprocessableEntity); r["codigo"] != "dados_invalidos" {
			t.Fatalf("esperava dados_invalidos: %v", r)
		}
	})

	t.Run("me da conta do público", func(t *testing.T) {
		me := c.chamar("GET", "/me", token, nil, http.StatusOK)
		if me["email"] != endereco || me["empresa_id"] != nil || me["tem_empresa"] != false {
			t.Fatalf("GET /me fora do contrato: %v", me)
		}
		consentimentos, _ := me["consentimentos"].([]any)
		if len(consentimentos) != 1 {
			t.Fatalf("esperava 1 consentimento (termos): %v", me["consentimentos"])
		}
	})

	t.Run("login por e-mail, por campo antigo e com senha errada", func(t *testing.T) {
		c.chamar("POST", "/auth/login", "", map[string]any{"identificador": endereco, "senha": "senha-forte-1"}, http.StatusOK)
		c.chamar("POST", "/auth/login", "", map[string]any{"login": endereco, "senha": "senha-forte-1"}, http.StatusOK)
		if r := c.chamar("POST", "/auth/login", "", map[string]any{"identificador": endereco, "senha": "errada-123"}, http.StatusUnauthorized); r["codigo"] != "credenciais_invalidas" {
			t.Fatalf("esperava credenciais_invalidas: %v", r)
		}
		c.chamar("POST", "/auth/login", "", map[string]any{"identificador": emailNovo(), "senha": "senha-forte-1"}, http.StatusUnauthorized)
	})

	t.Run("login de motorista de empresa continua funcionando", func(t *testing.T) {
		empresa := testutil.CriarEmpresa(t, pool)
		motorista := testutil.CriarUsuario(t, pool, empresa, usuario.PapelMotorista)
		var login string
		hash, _ := authHash("senha-motorista")
		if err := pool.QueryRow(t.Context(), `update usuario set senha_hash = $2 where id = $1 returning login`, motorista, hash).Scan(&login); err != nil {
			t.Fatal(err)
		}
		r := c.chamar("POST", "/auth/login", "", map[string]any{"login": login, "senha": "senha-motorista"}, http.StatusOK)
		if u, _ := r["usuario"].(map[string]any); u["tem_empresa"] != true {
			t.Fatalf("motorista deveria ter empresa: %v", r)
		}
	})

	t.Run("perfil e consentimento", func(t *testing.T) {
		me := c.chamar("PATCH", "/me", token, map[string]any{"apelido": "Maria Estradeira"}, http.StatusOK)
		if me["apelido"] != "Maria Estradeira" {
			t.Fatalf("apelido não atualizado: %v", me)
		}
		c.chamar("PATCH", "/me", token, map[string]any{"apelido": "M"}, http.StatusUnprocessableEntity)
		c.chamar("PUT", "/me/consentimentos/ponto_transmissao", token, map[string]any{"aceito": true}, http.StatusNoContent)
		c.chamar("PUT", "/me/consentimentos/termos", token, map[string]any{"aceito": false}, http.StatusUnprocessableEntity)
		c.chamar("PUT", "/me/consentimentos/ponto_transmissao", token, map[string]any{}, http.StatusBadRequest)
	})

	t.Run("recuperação de senha", func(t *testing.T) {
		c.chamar("POST", "/auth/recuperar-senha", "", map[string]any{"email": emailNovo()}, http.StatusAccepted)
		c.chamar("POST", "/auth/recuperar-senha", "", map[string]any{"email": endereco}, http.StatusAccepted)

		codigo := esperarCodigo(t, enviador, endereco)
		if r := c.chamar("POST", "/auth/redefinir-senha", "", map[string]any{"email": endereco, "codigo": "12345", "nova_senha": "nova-senha-123"}, http.StatusBadRequest); r["codigo"] != "codigo_invalido" {
			t.Fatalf("esperava codigo_invalido: %v", r)
		}
		c.chamar("POST", "/auth/redefinir-senha", "", map[string]any{"email": endereco, "codigo": codigo, "nova_senha": "nova-senha-123"}, http.StatusNoContent)
		c.chamar("POST", "/auth/login", "", map[string]any{"identificador": endereco, "senha": "nova-senha-123"}, http.StatusOK)
		c.chamar("POST", "/auth/login", "", map[string]any{"identificador": endereco, "senha": "senha-forte-1"}, http.StatusUnauthorized)
	})

	t.Run("exclusão da conta", func(t *testing.T) {
		c.chamar("DELETE", "/me", "", nil, http.StatusUnauthorized)
		c.chamar("DELETE", "/me", token, nil, http.StatusNoContent)
		c.chamar("GET", "/me", token, nil, http.StatusNotFound)
		c.chamar("POST", "/auth/login", "", map[string]any{"identificador": endereco, "senha": "nova-senha-123"}, http.StatusUnauthorized)
	})

	t.Run("termos e privacidade são públicos", func(t *testing.T) {
		for _, rota := range []string{"/termos", "/privacidade"} {
			resp, err := http.Get(servidor.URL + rota)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s: status %d", rota, resp.StatusCode)
			}
		}
	})
}

func TestRecuperacaoSemSMTPEhIndisponivel(t *testing.T) {
	pool := testutil.AbrirPool(t)
	servidor := httptest.NewServer(transporte.NovoRoteadorCom(config.Config{JWTSecret: "x"}, pool, transporte.Dependencias{}))
	defer servidor.Close()
	c := clienteTeste{t: t, url: servidor.URL}
	if r := c.chamar("POST", "/auth/recuperar-senha", "", map[string]any{"email": emailNovo()}, http.StatusServiceUnavailable); r["codigo"] != "recuperacao_indisponivel" {
		t.Fatalf("esperava recuperacao_indisponivel: %v", r)
	}
}

func TestCadastroLimitadoPorIP(t *testing.T) {
	pool := testutil.AbrirPool(t)
	servidor := httptest.NewServer(transporte.NovoRoteadorCom(
		config.Config{JWTSecret: "x"}, pool,
		transporte.Dependencias{LimitadorContas: transporte.NovoLimitador(2, time.Hour)},
	))
	defer servidor.Close()
	c := clienteTeste{t: t, url: servidor.URL}
	invalido := map[string]any{"nome": "", "email": "x"}
	c.chamar("POST", "/auth/cadastro", "", invalido, http.StatusUnprocessableEntity)
	c.chamar("POST", "/auth/cadastro", "", invalido, http.StatusUnprocessableEntity)
	if r := c.chamar("POST", "/auth/cadastro", "", invalido, http.StatusTooManyRequests); r["codigo"] != "muitas_tentativas" {
		t.Fatalf("esperava muitas_tentativas: %v", r)
	}
}

var codigoNoCorpo = regexp.MustCompile(`\b\d{6}\b`)

// esperarCodigo aguarda o e-mail enviado em segundo plano pela rota de
// recuperação e extrai o código.
func esperarCodigo(t *testing.T, enviador *email.EnviadorMemoria, para string) string {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for time.Now().Before(limite) {
		if m, ok := enviador.Ultima(para); ok {
			if codigo := codigoNoCorpo.FindString(m.Corpo); codigo != "" {
				return codigo
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("nenhum código enviado para %s", para)
	return ""
}

func authHash(senha string) (string, error) { return auth.HashSenha(senha) }

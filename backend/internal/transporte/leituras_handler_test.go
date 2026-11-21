package transporte_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

func orbitDeTeste() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "ORBIT-" + strings.ToUpper(hex.EncodeToString(b))
}

type eventoRede struct {
	Tipo  string          `json:"tipo"`
	Dados json.RawMessage `json:"dados"`
}

func lerEventoRede(t *testing.T, conn *websocket.Conn) eventoRede {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	_, dados, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("erro ao ler evento da Rede Orbit: %v", err)
	}
	var e eventoRede
	if err := json.Unmarshal(dados, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestContratoRedeOrbit(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	const segredo = "rede-teste"
	servidor := httptest.NewServer(transporte.NovoRoteador(config.Config{JWTSecret: segredo}, pool))
	defer servidor.Close()
	c := clienteTeste{t: t, url: servidor.URL}

	// Ponto de transmissão (conta do público) com e-mail conhecido, para
	// conferir que nada dele aparece no painel.
	emailPonto := emailNovo()
	var pontoID string
	if err := pool.QueryRow(ctx, `
		insert into usuario (nome, apelido, email, senha_hash, papel) values ('Pessoa Ponto', 'ApelidoSecreto', $1, 'h', 'usuario') returning id
	`, emailPonto).Scan(&pontoID); err != nil {
		t.Fatal(err)
	}
	tokenPonto, _ := auth.GerarToken(segredo, pontoID, "", auth.PapelUsuario)

	var adminID string
	if err := pool.QueryRow(ctx, `
		insert into usuario (nome, apelido, email, senha_hash, papel) values ('KTM', 'KTM', $1, 'h', 'admin_ktm') returning id
	`, emailNovo()).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	tokenAdmin, _ := auth.GerarToken(segredo, adminID, "", auth.PapelAdminKTM)

	orbitID := orbitDeTeste()
	lote := map[string]any{"leituras": []map[string]any{{
		"orbit_id": orbitID, "latitude": -23.55, "longitude": -46.63, "precisao_metros": 8.5, "rssi": -67,
		"detectado_em": time.Now().Add(-10 * time.Second).UTC().Format(time.RFC3339Nano),
	}}}

	t.Run("sem consentimento não envia leituras", func(t *testing.T) {
		if r := c.chamar("POST", "/leituras", tokenPonto, lote, http.StatusForbidden); r["codigo"] != "sem_consentimento" {
			t.Fatalf("esperava sem_consentimento: %v", r)
		}
		c.chamar("POST", "/ponto/sinal", tokenPonto, nil, http.StatusForbidden)
		c.chamar("POST", "/leituras", "", lote, http.StatusUnauthorized)
	})

	c.chamar("PUT", "/me/consentimentos/"+usuario.ConsentimentoPontoTransmissao, tokenPonto, map[string]any{"aceito": true}, http.StatusNoContent)

	t.Run("painel é restrito a admin_ktm", func(t *testing.T) {
		c.chamar("GET", "/rede/orbits", tokenPonto, nil, http.StatusForbidden)
		c.chamar("GET", "/rede/pontos-fixos", "", nil, http.StatusUnauthorized)
		url := "ws" + servidor.URL[len("http"):] + "/ws/rede?token=" + tokenPonto
		dialCtx, cancelar := context.WithTimeout(ctx, 3*time.Second)
		defer cancelar()
		if conn, _, err := websocket.Dial(dialCtx, url, nil); err == nil {
			conn.CloseNow()
			t.Fatal("WebSocket do painel deveria recusar quem não é admin_ktm")
		}
	})

	// Painel conectado antes da leitura, para receber o evento ao vivo.
	dialCtx, cancelar := context.WithTimeout(ctx, 3*time.Second)
	defer cancelar()
	painel, _, err := websocket.Dial(dialCtx, "ws"+servidor.URL[len("http"):]+"/ws/rede?token="+tokenAdmin, nil)
	if err != nil {
		t.Fatalf("erro ao conectar o painel: %v", err)
	}
	defer painel.CloseNow()
	if e := lerEventoRede(t, painel); e.Tipo != "inicial" {
		t.Fatalf("primeiro evento deveria ser inicial, veio %q", e.Tipo)
	}

	t.Run("lote aceito, reenviado e inválido", func(t *testing.T) {
		r := c.chamar("POST", "/leituras", tokenPonto, lote, http.StatusOK)
		if r["aceitas"] != float64(1) || r["duplicadas"] != float64(0) {
			t.Fatalf("resposta fora do contrato: %v", r)
		}
		if len(r) != 2 {
			t.Fatalf("a resposta não deveria trazer dados do Orbit (FR-011): %v", r)
		}
		r = c.chamar("POST", "/leituras", tokenPonto, lote, http.StatusOK)
		if r["aceitas"] != float64(0) || r["duplicadas"] != float64(1) {
			t.Fatalf("reenvio deveria ser duplicado: %v", r)
		}
		invalido := map[string]any{"leituras": []map[string]any{{"orbit_id": "xyz", "latitude": -23.5, "longitude": -46.6, "detectado_em": time.Now().UTC().Format(time.RFC3339)}}}
		if r := c.chamar("POST", "/leituras", tokenPonto, invalido, http.StatusUnprocessableEntity); r["codigo"] != "lote_invalido" {
			t.Fatalf("esperava lote_invalido: %v", r)
		}
	})

	t.Run("painel recebe a leitura ao vivo sem dados de quem leu", func(t *testing.T) {
		vistos := map[string]string{}
		limite := time.Now().Add(5 * time.Second)
		for (vistos["leitura"] == "" || vistos["orbit_atualizado"] == "") && time.Now().Before(limite) {
			e := lerEventoRede(t, painel)
			vistos[e.Tipo] = string(e.Dados)
		}
		leitura := vistos["leitura"]
		if !strings.Contains(leitura, orbitID) || !strings.Contains(leitura, `"origem":"celular"`) {
			t.Fatalf("evento de leitura inesperado: %s", leitura)
		}
		if vistos["orbit_atualizado"] == "" {
			t.Fatal("esperava evento orbit_atualizado")
		}
		for tipo, dados := range vistos {
			for _, proibido := range []string{pontoID, emailPonto, "ApelidoSecreto", "Pessoa Ponto", "usuario"} {
				if strings.Contains(dados, proibido) {
					t.Fatalf("evento %s expõe dado de quem leu (%q): %s", tipo, proibido, dados)
				}
			}
		}
	})

	t.Run("contribuição e sinal", func(t *testing.T) {
		c.chamar("POST", "/ponto/sinal", tokenPonto, nil, http.StatusNoContent)
		r := c.chamar("GET", "/me/contribuicao", tokenPonto, nil, http.StatusOK)
		if r["total"] != float64(1) || r["ponto_ativo"] != true {
			t.Fatalf("contribuição fora do contrato: %v", r)
		}
	})

	t.Run("rotas do painel", func(t *testing.T) {
		orbits := c.chamar("GET", "/rede/orbits", tokenAdmin, nil, http.StatusOK)
		if !strings.Contains(mustJSON(orbits), orbitID) {
			t.Fatalf("Orbit lido deveria aparecer em /rede/orbits")
		}
		historico := c.chamar("GET", "/rede/orbits/"+orbitID+"/leituras", tokenAdmin, nil, http.StatusOK)
		if lista, _ := historico["leituras"].([]any); len(lista) != 1 {
			t.Fatalf("esperava 1 leitura no histórico: %v", historico)
		}
		c.chamar("GET", "/rede/orbits/"+orbitDeTeste()+"/leituras", tokenAdmin, nil, http.StatusNotFound)

		c.chamar("POST", "/rede/pontos-fixos", tokenAdmin, map[string]any{"nome": "x", "latitude": -23.6, "longitude": -46.8, "raio_metros": 1}, http.StatusUnprocessableEntity)
		ponto := c.chamar("POST", "/rede/pontos-fixos", tokenAdmin, map[string]any{"nome": "Pórtico de teste", "latitude": -23.6, "longitude": -46.8, "raio_metros": 300}, http.StatusCreated)
		pontoFixoID, _ := ponto["id"].(string)
		r := c.chamar("POST", "/rede/pontos-fixos/"+pontoFixoID+"/simular", tokenAdmin, map[string]any{"orbit_id": orbitID}, http.StatusOK)
		if r["aceitas"] != float64(1) {
			t.Fatalf("simulação deveria gerar uma leitura: %v", r)
		}
		var origem string
		if err := pool.QueryRow(ctx, `select origem from leitura_orbit where orbit_id = $1 order by detectado_em desc limit 1`, orbitID).Scan(&origem); err != nil || origem != "fixo" {
			t.Fatalf("a leitura simulada deveria ter origem fixo: %q %v", origem, err)
		}
		c.chamar("DELETE", "/rede/pontos-fixos/"+pontoFixoID, tokenAdmin, nil, http.StatusNoContent)
		c.chamar("POST", "/rede/pontos-fixos/"+pontoFixoID+"/simular", tokenAdmin, map[string]any{"orbit_id": orbitID}, http.StatusNotFound)
	})
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

package ws_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
)

const segredoTeste = "segredo-de-teste-ws"

type mensagem struct {
	Tipo  string         `json:"tipo"`
	Dados map[string]any `json:"dados"`
}

// TestProtocoloWebSocketTransmissaoDeVoz cobre a sequência completa do
// contrato (contracts/websocket-protocol.md): entrar_canal → solicitar_slot
// → slot_concedido → frames binários → inicio_reproducao/fim_reproducao.
func TestProtocoloWebSocketTransmissaoDeVoz(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()

	empresaID := novoUUID()
	motoristaAID := novoUUID()
	motoristaBID := novoUUID()
	canalID := novoUUID()

	if _, err := pool.Exec(ctx, `insert into empresa (id, razao_social, cnpj) values ($1, 'Empresa Teste', $2)`,
		empresaID, cnpjUnico()); err != nil {
		t.Fatalf("erro ao inserir empresa de teste: %v", err)
	}
	for _, u := range []struct{ id, nome, login string }{
		{motoristaAID, "Motorista A", "motorista-a-" + motoristaAID},
		{motoristaBID, "Motorista B", "motorista-b-" + motoristaBID},
	} {
		if _, err := pool.Exec(ctx, `
			insert into usuario (id, empresa_id, nome, login, senha_hash, papel)
			values ($1, $2, $3, $4, 'hash', 'motorista')
		`, u.id, empresaID, u.nome, u.login); err != nil {
			t.Fatalf("erro ao inserir usuário de teste %s: %v", u.nome, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		insert into canal (id, empresa_id, nome) values ($1, $2, 'Canal de Teste')
	`, canalID, empresaID); err != nil {
		t.Fatalf("erro ao inserir canal de teste: %v", err)
	}

	cfg := config.Config{DatabaseURL: "usado-apenas-pelo-pool-de-teste", JWTSecret: segredoTeste, Porta: "0"}
	servidor := httptest.NewServer(transporte.NovoRoteador(cfg, pool))
	defer servidor.Close()

	connA := conectar(t, servidor.URL, motoristaAID, empresaID)
	defer connA.CloseNow()
	connB := conectar(t, servidor.URL, motoristaBID, empresaID)
	defer connB.CloseNow()

	entrarNoCanal(t, connA, canalID)
	entrarNoCanal(t, connB, canalID)

	enviarTexto(t, connA, "solicitar_slot", map[string]any{"canal_id": canalID})
	slotConcedido := lerMensagem(t, connA)
	if slotConcedido.Tipo != "slot_concedido" {
		t.Fatalf("esperava slot_concedido, veio %q", slotConcedido.Tipo)
	}
	transmissaoID, _ := slotConcedido.Dados["transmissao_id"].(string)
	if transmissaoID == "" {
		t.Fatal("slot_concedido não trouxe transmissao_id")
	}

	if err := connA.Write(context.Background(), websocket.MessageBinary, []byte("chunk-opus-1")); err != nil {
		t.Fatalf("erro ao enviar frame binário: %v", err)
	}
	enviarTexto(t, connA, "finalizar_transmissao", map[string]any{"transmissao_id": transmissaoID})

	inicio := lerMensagem(t, connB)
	if inicio.Tipo != "inicio_reproducao" {
		t.Fatalf("esperava inicio_reproducao em B, veio %q", inicio.Tipo)
	}
	if inicio.Dados["remetente_nome"] != "Motorista A" {
		t.Fatalf("esperava remetente_nome = Motorista A, veio %v", inicio.Dados["remetente_nome"])
	}

	tipoFrame, dadosFrame := lerFrame(t, connB)
	if tipoFrame != websocket.MessageBinary || string(dadosFrame) != "chunk-opus-1" {
		t.Fatalf("esperava frame binário 'chunk-opus-1', veio tipo=%v dados=%q", tipoFrame, dadosFrame)
	}

	fim := lerMensagem(t, connB)
	if fim.Tipo != "fim_reproducao" {
		t.Fatalf("esperava fim_reproducao em B, veio %q", fim.Tipo)
	}
}

func entrarNoCanal(t *testing.T, conn *websocket.Conn, canalID string) {
	t.Helper()
	enviarTexto(t, conn, "entrar_canal", map[string]any{"canal_id": canalID})
	m := lerMensagem(t, conn)
	if m.Tipo != "canal_entrado" {
		t.Fatalf("esperava canal_entrado, veio %q (dados: %v)", m.Tipo, m.Dados)
	}
}

func conectar(t *testing.T, servidorURL, usuarioID, empresaID string) *websocket.Conn {
	t.Helper()
	token, err := auth.GerarToken(segredoTeste, usuarioID, empresaID, auth.PapelMotorista)
	if err != nil {
		t.Fatalf("erro ao gerar token de teste: %v", err)
	}

	url := "ws" + servidorURL[len("http"):] + "/ws?token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("erro ao conectar ao WebSocket: %v", err)
	}
	return conn
}

func enviarTexto(t *testing.T, conn *websocket.Conn, tipo string, dados map[string]any) {
	t.Helper()
	corpo, err := json.Marshal(mensagem{Tipo: tipo, Dados: dados})
	if err != nil {
		t.Fatalf("erro ao serializar mensagem %q: %v", tipo, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, corpo); err != nil {
		t.Fatalf("erro ao enviar mensagem %q: %v", tipo, err)
	}
}

func lerMensagem(t *testing.T, conn *websocket.Conn) mensagem {
	t.Helper()
	tipo, dados := lerFrame(t, conn)
	if tipo != websocket.MessageText {
		t.Fatalf("esperava frame de texto, veio tipo=%v", tipo)
	}
	var m mensagem
	if err := json.Unmarshal(dados, &m); err != nil {
		t.Fatalf("erro ao decodificar mensagem: %v (corpo: %s)", err, dados)
	}
	return m
}

func lerFrame(t *testing.T, conn *websocket.Conn) (websocket.MessageType, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tipo, dados, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("erro ao ler frame: %v", err)
	}
	return tipo, dados
}

func novoUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func cnpjUnico() string {
	b := make([]byte, 7)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:14]
}

package ws_test

import (
	"context"
	"encoding/json"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCentralRestringeAcessoEMantemFilaDeVoz(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	empresa := testutil.CriarEmpresa(t, pool)
	outraEmpresa := testutil.CriarEmpresa(t, pool)
	motorista := testutil.CriarUsuario(t, pool, empresa, "motorista")
	admin := testutil.CriarUsuario(t, pool, empresa, "admin")
	outroAdmin := testutil.CriarUsuario(t, pool, outraEmpresa, "admin")
	servico := canal.NovoServico(canal.NovoRepositorio(pool), canal.NovoGerenciador(), posicao.NovoRepositorio(pool))
	c, err := servico.Criar(ctx, empresa, canal.DTOCanal{Nome: "Central", TipoAcesso: "privado", LimiteParticipantes: 5})
	if err != nil {
		t.Fatal(err)
	}
	servidor := httptest.NewServer(transporte.NovoRoteador(config.Config{JWTSecret: segredoTeste}, pool))
	defer servidor.Close()
	adminToken, _ := auth.GerarToken(segredoTeste, admin, empresa, auth.PapelAdmin)
	motoristaToken, _ := auth.GerarToken(segredoTeste, motorista, empresa, auth.PapelMotorista)
	outroToken, _ := auth.GerarToken(segredoTeste, outroAdmin, outraEmpresa, auth.PapelAdmin)
	for _, caso := range []struct {
		rota, token string
		status      int
	}{{"/central/canais", "", 401}, {"/central/canais", motoristaToken, 403}, {"/ws/central?canal_id=" + c.ID, motoristaToken, 403}, {"/ws/central?canal_id=" + c.ID, outroToken, 403}} {
		req, _ := http.NewRequest("GET", servidor.URL+caso.rota, nil)
		if caso.token != "" {
			req.Header.Set("Authorization", "Bearer "+caso.token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != caso.status {
			t.Fatalf("%s: status %d, esperado %d", caso.rota, resp.StatusCode, caso.status)
		}
	}
	remetente := conectar(t, servidor.URL, motorista, empresa)
	defer remetente.CloseNow()
	entrarNoCanal(t, remetente, c.ID)
	tempo, cancelar := context.WithTimeout(ctx, 3*time.Second)
	defer cancelar()
	central, _, err := websocket.Dial(tempo, "ws"+servidor.URL[len("http"):]+"/ws/central?canal_id="+c.ID+"&token="+adminToken, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer central.CloseNow()
	req, _ := http.NewRequest("GET", servidor.URL+"/central/canais", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Canais []struct {
			ID            string `json:"id"`
			Participantes []struct {
				UsuarioID string `json:"usuario_id"`
			} `json:"participantes"`
		} `json:"canais"`
	}
	err = json.NewDecoder(resp.Body).Decode(&snapshot)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Canais) != 1 || len(snapshot.Canais[0].Participantes) != 1 || snapshot.Canais[0].Participantes[0].UsuarioID != motorista {
		t.Fatalf("snapshot incorreto: %+v", snapshot)
	}
	for i := 0; i < 2; i++ {
		enviarTexto(t, remetente, "solicitar_slot", map[string]any{"canal_id": c.ID})
		slot := lerMensagem(t, remetente)
		if slot.Tipo != "slot_concedido" {
			t.Fatal(slot)
		}
		payload := []byte{byte(i + 1)}
		if err := remetente.Write(ctx, websocket.MessageBinary, payload); err != nil {
			t.Fatal(err)
		}
		enviarTexto(t, remetente, "finalizar_transmissao", map[string]any{"transmissao_id": slot.Dados["transmissao_id"]})
		inicio := lerMensagem(t, central)
		if inicio.Tipo != "inicio_reproducao" || inicio.Dados["transmissao_id"] != slot.Dados["transmissao_id"] {
			t.Fatal(inicio)
		}
		tipo, audio := lerFrame(t, central)
		if tipo != websocket.MessageBinary || len(audio) != 1 || audio[0] != payload[0] {
			t.Fatal("áudio não entregue em ordem")
		}
		if fim := lerMensagem(t, central); fim.Tipo != "fim_reproducao" {
			t.Fatal(fim)
		}
		lerMensagem(t, remetente)
	}
}

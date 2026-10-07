package transporte_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
)

func TestCadastroAdministrativoCriaContaLoginEProtegeEmpresa(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	empresa := testutil.CriarEmpresa(t, pool)
	outra := testutil.CriarEmpresa(t, pool)
	t.Cleanup(func() {
		for _, empresaDoTeste := range []string{empresa, outra} {
			for _, tabela := range []string{"usuario", "veiculo", "empresa"} {
				coluna := "empresa_id"
				if tabela == "empresa" {
					coluna = "id"
				}
				if _, err := pool.Exec(ctx, "delete from "+tabela+" where "+coluna+"=$1", empresaDoTeste); err != nil {
					t.Errorf("limpeza do banco de testes: %v", err)
				}
			}
		}
	})
	admin := testutil.CriarUsuario(t, pool, empresa, "admin")
	adminOutro := testutil.CriarUsuario(t, pool, outra, "admin")
	motorista := testutil.CriarUsuario(t, pool, empresa, "motorista")
	const segredo = "cadastros-teste"
	token, _ := auth.GerarToken(segredo, admin, empresa, auth.PapelAdmin)
	outroToken, _ := auth.GerarToken(segredo, adminOutro, outra, auth.PapelAdmin)
	motoristaToken, _ := auth.GerarToken(segredo, motorista, empresa, auth.PapelMotorista)
	servidor := httptest.NewServer(transporte.NovoRoteador(config.Config{JWTSecret: segredo}, pool))
	defer servidor.Close()
	requisicao := func(metodo, rota, jwt string, dados any, esperado int) map[string]any {
		t.Helper()
		var body []byte
		if dados != nil {
			body, _ = json.Marshal(dados)
		}
		req, _ := http.NewRequest(metodo, servidor.URL+rota, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if jwt != "" {
			req.Header.Set("Authorization", "Bearer "+jwt)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		conteudo, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != esperado {
			t.Fatalf("%s %s: status %d, esperado %d; %s", metodo, rota, resp.StatusCode, esperado, conteudo)
		}
		var resultado map[string]any
		if err = json.Unmarshal(conteudo, &resultado); err != nil {
			t.Fatal(err)
		}
		return resultado
	}
	for _, rota := range []string{"/admin/veiculos", "/admin/motoristas"} {
		requisicao("GET", rota, "", nil, 401)
		requisicao("GET", rota, motoristaToken, nil, 403)
		requisicao("POST", rota, motoristaToken, map[string]any{}, 403)
	}
	// CPFs usados apenas como dados sintéticos dos testes de validação.
	dados := map[string]any{"nome": "Ana", "sobrenome": "Da Estrada", "cpf": "529.982.247-25", "senha": "senha-teste-123", "confirmacao_senha": "senha-teste-123", "novo_veiculo": map[string]any{"descricao": "Volvo FH", "placa": "abc-1234"}}
	criada := requisicao("POST", "/admin/motoristas", token, dados, 201)
	if criada["nome"] != "Ana Da Estrada" || criada["login"] != "52998224725" {
		t.Fatal("nome completo ou login incorreto")
	}
	for _, campo := range []string{"senha", "senha_hash", "confirmacao_senha"} {
		if _, ok := criada[campo]; ok {
			t.Fatal("resposta expôs senha")
		}
	}
	var hash string
	if err := pool.QueryRow(ctx, `select senha_hash from usuario where id=$1`, criada["id"]).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == dados["senha"] || !auth.VerificarSenha(hash, dados["senha"].(string)) {
		t.Fatal("senha não armazenada como bcrypt")
	}
	login := requisicao("POST", "/auth/login", "", map[string]any{"login": "529.982.247-25", "senha": "senha-teste-123"}, 200)
	if login["usuario"].(map[string]any)["nome"] != "Ana Da Estrada" {
		t.Fatal("nome não aparece no login do app")
	}
	veiculo := criada["veiculo"].(map[string]any)
	if veiculo["placa"] != "ABC1234" {
		t.Fatal("placa não normalizada")
	}
	// Duplicar o CPF não deixa um veículo órfão na empresa.
	dados["novo_veiculo"] = map[string]any{"descricao": "Não deve existir", "placa": "DEF2G34"}
	requisicao("POST", "/admin/motoristas", token, dados, 409)
	var quantidade int
	pool.QueryRow(ctx, `select count(*) from veiculo where empresa_id=$1`, empresa).Scan(&quantidade)
	if quantidade != 1 {
		t.Fatal("cadastro inválido deixou veículo órfão")
	}
	externo := requisicao("POST", "/admin/veiculos", outroToken, map[string]any{"descricao": "Outro veículo", "placa": "GHI1234"}, 201)
	delete(dados, "novo_veiculo")
	dados["veiculo_id"] = externo["id"]
	dados["cpf"] = "11144477735"
	requisicao("POST", "/admin/motoristas", token, dados, 422)
	dados["veiculo_id"] = veiculo["id"]
	dados["confirmacao_senha"] = "outra-senha"
	requisicao("POST", "/admin/motoristas", token, dados, 422)
	dados["confirmacao_senha"] = dados["senha"]
	dados["cpf"] = "11111111111"
	requisicao("POST", "/admin/motoristas", token, dados, 422)
	dados["cpf"] = "11144477735"
	dados["papel"] = "admin"
	requisicao("POST", "/admin/motoristas", token, dados, 400)
	delete(dados, "papel")
	dados["nome"] = "Beto"
	requisicao("POST", "/admin/motoristas", token, dados, 201)
	lista := requisicao("GET", "/admin/motoristas", token, nil, 200)
	if len(lista["motoristas"].([]any)) != 3 {
		t.Fatal("cadastros não apareceram na lista")
	}
	isolada := requisicao("GET", "/admin/motoristas", outroToken, nil, 200)
	if len(isolada["motoristas"].([]any)) != 0 {
		t.Fatal("outra empresa acessou os caminhoneiros")
	}
	veiculos := requisicao("GET", "/admin/veiculos", token, nil, 200)
	if len(veiculos["veiculos"].([]any)) != 1 {
		t.Fatal("veículos de outra empresa apareceram")
	}
	requisicao("POST", "/auth/login", "", map[string]any{"login": "52998224725", "senha": "senha-incorreta"}, 401)
	if strings.Contains(hash, "senha-teste") {
		t.Fatal("senha armazenada em texto")
	}
}

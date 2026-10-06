package usuario

import "testing"

func TestCPFNormalizacaoEDigitos(t *testing.T) {
	for _, caso := range []struct {
		cpf    string
		valido bool
	}{{"529.982.247-25", true}, {"11144477735", true}, {"00000000000", false}, {"52998224726", false}, {"letras52998224725", false}, {"", false}} {
		if cpfValido(NormalizarCPF(caso.cpf)) != caso.valido {
			t.Fatalf("CPF: resultado incorreto para %q", caso.cpf)
		}
	}
	if NormalizarLogin(" motorista1 ") != "motorista1" {
		t.Fatal("login existente alterado")
	}
	if NormalizarLogin("529.982.247-25") != "52998224725" {
		t.Fatal("CPF de login não normalizado")
	}
}

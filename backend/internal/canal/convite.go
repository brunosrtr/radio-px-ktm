package canal

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// alfabetoConvite não tem caracteres fáceis de confundir (0/O, 1/I/L).
const alfabetoConvite = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// TamanhoConvite é o tamanho do código, sem o hífen.
const TamanhoConvite = 8

// GerarCodigoConvite sorteia um código de convite com crypto/rand.
func GerarCodigoConvite() (string, error) {
	var b strings.Builder
	limite := big.NewInt(int64(len(alfabetoConvite)))
	for range TamanhoConvite {
		n, err := rand.Int(rand.Reader, limite)
		if err != nil {
			return "", err
		}
		b.WriteByte(alfabetoConvite[n.Int64()])
	}
	return b.String(), nil
}

// NormalizarCodigoConvite aceita o que a pessoa digita ("k7qm-2xpa",
// "K7QM 2XPA", o link radiopx://app/convite/K7QM2XPA) e devolve o código
// guardado no banco, ou "" se não for um código válido.
func NormalizarCodigoConvite(entrada string) string {
	entrada = strings.TrimSpace(entrada)
	if i := strings.LastIndex(entrada, "/"); i >= 0 {
		entrada = entrada[i+1:]
	}
	var b strings.Builder
	for _, r := range strings.ToUpper(entrada) {
		switch {
		case r == '-' || r == ' ':
			continue
		case strings.ContainsRune(alfabetoConvite, r):
			b.WriteRune(r)
		default:
			return ""
		}
	}
	if b.Len() != TamanhoConvite {
		return ""
	}
	return b.String()
}

// FormatarCodigoConvite mostra o código como XXXX-XXXX.
func FormatarCodigoConvite(codigo string) string {
	if len(codigo) != TamanhoConvite {
		return codigo
	}
	return codigo[:4] + "-" + codigo[4:]
}

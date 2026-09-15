// Package auth implementa hash de senha e emissão/validação de JWT — a
// autenticação é stateless, sem tabela de sessões (ver research.md §5).
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrTokenInvalido é retornado quando o JWT está ausente, expirado ou
// malformado — corresponde ao código de erro "token_invalido" do contrato
// REST/WebSocket.
var ErrTokenInvalido = errors.New("auth: token inválido")

// Papel identifica o tipo de usuário autenticado.
type Papel string

const (
	PapelAdmin     Papel = "admin"
	PapelMotorista Papel = "motorista"
	expiracaoToken       = 24 * time.Hour
)

// Claims são os dados carregados pelo JWT emitido no login (RF21, RNF11,
// RNF12): usuário, empresa e papel, suficientes para autorizar sem round-trip
// ao banco.
type Claims struct {
	UsuarioID string `json:"usuario_id"`
	EmpresaID string `json:"empresa_id"`
	Papel     Papel  `json:"papel"`
	jwt.RegisteredClaims
}

// HashSenha calcula o hash bcrypt de uma senha em texto plano.
func HashSenha(senha string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: erro ao gerar hash de senha: %w", err)
	}
	return string(hash), nil
}

// VerificarSenha compara uma senha em texto plano com um hash bcrypt
// previamente armazenado.
func VerificarSenha(hash, senha string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil
}

// GerarToken emite um JWT assinado contendo usuário, empresa e papel.
func GerarToken(segredo, usuarioID, empresaID string, papel Papel) (string, error) {
	agora := time.Now()
	claims := Claims{
		UsuarioID: usuarioID,
		EmpresaID: empresaID,
		Papel:     papel,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(agora),
			ExpiresAt: jwt.NewNumericDate(agora.Add(expiracaoToken)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	assinado, err := token.SignedString([]byte(segredo))
	if err != nil {
		return "", fmt.Errorf("auth: erro ao assinar token: %w", err)
	}
	return assinado, nil
}

// ValidarToken decodifica e valida um JWT, retornando seus claims.
func ValidarToken(segredo, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: método de assinatura inesperado: %v", t.Header["alg"])
		}
		return []byte(segredo), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrTokenInvalido
	}
	return claims, nil
}

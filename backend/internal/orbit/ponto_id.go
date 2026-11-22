package orbit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"
	// Garante o fuso America/Sao_Paulo mesmo em imagens sem tzdata.
	_ "time/tzdata"
)

// FusoContagem define o "dia" do ponto_id rotativo e da contribuição diária.
var FusoContagem = func() *time.Location {
	fuso, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*60*60)
	}
	return fuso
}()

// Dia devolve a data (no fuso de contagem) do instante informado.
func Dia(instante time.Time) string {
	return instante.In(FusoContagem).Format("2006-01-02")
}

// PontoID devolve o identificador opaco do ponto de transmissão do usuário
// no dia do instante informado: HMAC-SHA256(segredo, usuario_id ‖ dia),
// truncado em 16 hexadecimais. Muda todo dia e não pode ser revertido para
// o usuário sem o segredo (Princípio VII).
func PontoID(segredo, usuarioID string, instante time.Time) string {
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write([]byte(usuarioID))
	mac.Write([]byte{0})
	mac.Write([]byte(Dia(instante)))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

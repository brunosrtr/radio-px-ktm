// Package notificacao abstrai o envio de notificações push (FR-017), para
// que quem aciona uma notificação (ex.: remoção por geocerca) não dependa de
// um provedor concreto.
package notificacao

import (
	"context"
	"log"
)

// Notificador envia uma notificação push para um dispositivo identificado
// pelo seu push_token.
type Notificador interface {
	Enviar(ctx context.Context, pushToken, titulo, mensagem string) error
}

// NotificadorLog é a implementação padrão da v1: registra a notificação no
// log do servidor em vez de integrar um provedor real (FCM/APNs), que exige
// credenciais e um projeto externos fora do escopo desta entrega acadêmica —
// a interface Notificador existe justamente para permitir essa troca depois
// sem mexer em quem a usa.
type NotificadorLog struct{}

// Enviar implementa Notificador.
func (NotificadorLog) Enviar(_ context.Context, pushToken, titulo, mensagem string) error {
	log.Printf("notificacao: [push simulado] token=%s titulo=%q mensagem=%q", pushToken, titulo, mensagem)
	return nil
}

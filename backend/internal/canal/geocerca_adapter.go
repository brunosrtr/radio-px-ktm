package canal

import (
	"context"
	"fmt"
	"log"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/notificacao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// AdaptadorGeocerca implementa geofence.Removedor sobre o Gerenciador e o
// Repositorio de canais — existe para que geofence.Verificador possa agir
// sem o pacote geofence importar canal (canal já importa geofence para o
// cálculo de distância; a dependência não pode ir nos dois sentidos).
type AdaptadorGeocerca struct {
	gerenciador *Gerenciador
	repositorio *Repositorio
	usuarios    *usuario.Repositorio
	notificador notificacao.Notificador
}

// NovoAdaptadorGeocerca cria o adaptador sobre o gerenciador, o repositório
// de canais, o repositório de usuários (para os push tokens) e o
// notificador informados.
func NovoAdaptadorGeocerca(gerenciador *Gerenciador, repositorio *Repositorio, usuarios *usuario.Repositorio, notificador notificacao.Notificador) *AdaptadorGeocerca {
	return &AdaptadorGeocerca{
		gerenciador: gerenciador,
		repositorio: repositorio,
		usuarios:    usuarios,
		notificador: notificador,
	}
}

// CanaisAtivosDoUsuario implementa geofence.Removedor.
func (a *AdaptadorGeocerca) CanaisAtivosDoUsuario(usuarioID string) []string {
	return a.gerenciador.CanaisDoUsuario(usuarioID)
}

// ObterCanal implementa geofence.Removedor.
func (a *AdaptadorGeocerca) ObterCanal(ctx context.Context, canalID string) (*geofence.CanalComGeocerca, error) {
	c, err := a.repositorio.ObterCanal(ctx, canalID)
	if err != nil {
		return nil, err
	}
	if !c.GeocercaAtiva {
		return &geofence.CanalComGeocerca{GeocercaAtiva: false}, nil
	}
	return &geofence.CanalComGeocerca{
		GeocercaAtiva:   true,
		CentroLatitude:  *c.CentroLatitude,
		CentroLongitude: *c.CentroLongitude,
		RaioMetros:      *c.RaioMetros,
	}, nil
}

// Remover implementa geofence.Removedor: tira o usuário do hub, notifica-o
// com removido_canal na conexão ativa (se houver), grava o motivo em
// participacao_canal e envia um push (FR-017) — útil sobretudo quando o app
// está em segundo plano e a conexão WebSocket pode já estar suspensa.
func (a *AdaptadorGeocerca) Remover(canalID, usuarioID, motivo string) {
	ctx := context.Background()

	nomeCanal := canalID
	if c, err := a.repositorio.ObterCanal(ctx, canalID); err == nil {
		nomeCanal = c.Nome
	}

	if hub, ok := a.gerenciador.Obter(canalID); ok {
		hub.Remover(usuarioID, Evento{Tipo: "removido_canal", Dados: map[string]any{
			"canal_id": canalID,
			"motivo":   motivo,
		}})
	}

	if err := a.repositorio.RegistrarSaida(ctx, canalID, usuarioID, motivo); err != nil {
		log.Printf("canal: erro ao registrar saída forçada: %v", err)
	}

	a.notificarRemocao(ctx, usuarioID, nomeCanal, motivo)
}

func (a *AdaptadorGeocerca) notificarRemocao(ctx context.Context, usuarioID, nomeCanal, motivo string) {
	if motivo != "geocerca" {
		return
	}

	tokens, err := a.usuarios.PushTokens(ctx, usuarioID)
	if err != nil {
		log.Printf("canal: erro ao buscar push tokens para aviso de remoção: %v", err)
		return
	}

	mensagem := fmt.Sprintf("Você saiu da área do canal %q e foi desconectado.", nomeCanal)
	for _, token := range tokens {
		if err := a.notificador.Enviar(ctx, token, "Removido do canal", mensagem); err != nil {
			log.Printf("canal: erro ao enviar notificação push: %v", err)
		}
	}
}

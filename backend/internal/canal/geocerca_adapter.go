package canal

import (
	"context"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
)

// AdaptadorGeocerca implementa geofence.Removedor sobre o Gerenciador e o
// Repositorio de canais — existe para que geofence.Verificador possa agir
// sem o pacote geofence importar canal (canal já importa geofence para o
// cálculo de distância; a dependência não pode ir nos dois sentidos).
type AdaptadorGeocerca struct {
	gerenciador *Gerenciador
	repositorio *Repositorio
}

// NovoAdaptadorGeocerca cria o adaptador sobre o gerenciador e o repositório
// informados.
func NovoAdaptadorGeocerca(gerenciador *Gerenciador, repositorio *Repositorio) *AdaptadorGeocerca {
	return &AdaptadorGeocerca{gerenciador: gerenciador, repositorio: repositorio}
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
// com removido_canal e grava o motivo em participacao_canal.
func (a *AdaptadorGeocerca) Remover(canalID, usuarioID, motivo string) {
	if hub, ok := a.gerenciador.Obter(canalID); ok {
		hub.Remover(usuarioID, Evento{Tipo: "removido_canal", Dados: map[string]any{
			"canal_id": canalID,
			"motivo":   motivo,
		}})
	}
	_ = a.repositorio.RegistrarSaida(context.Background(), canalID, usuarioID, motivo)
}

package geofence

import "context"

// CanalComGeocerca é o subconjunto de dados de um canal necessário para
// decidir se um motorista deve continuar nele.
type CanalComGeocerca struct {
	GeocercaAtiva   bool
	CentroLatitude  float64
	CentroLongitude float64
	RaioMetros      int
}

// Removedor é implementado pela camada de canais (internal/canal) — o
// pacote geofence não importa canal para evitar um ciclo de importação
// (canal já importa geofence para o cálculo de distância).
type Removedor interface {
	// CanaisAtivosDoUsuario lista os canais em que o usuário está
	// atualmente conectado no hub em memória.
	CanaisAtivosDoUsuario(usuarioID string) []string
	// ObterCanal busca os dados de geocerca de um canal.
	ObterCanal(ctx context.Context, canalID string) (*CanalComGeocerca, error)
	// Remover tira o usuário do canal, notifica-o e grava o motivo da saída.
	Remover(canalID, usuarioID, motivo string)
}

// Verificador remove um motorista dos canais com geocerca ativa nos quais
// está conectado quando sua posição sai do raio configurado (FR-016/FR-017).
// É acionado a cada atualização de posição recebida, não por um polling
// separado (research.md §6).
type Verificador struct {
	removedor Removedor
}

// NovoVerificador cria o Verificador sobre o Removedor informado.
func NovoVerificador(removedor Removedor) *Verificador {
	return &Verificador{removedor: removedor}
}

// VerificarPosicao reavalia todos os canais com geocerca ativa em que o
// usuário está conectado e o remove dos que ficaram fora do raio.
func (v *Verificador) VerificarPosicao(ctx context.Context, usuarioID string, lat, lon float64) {
	for _, canalID := range v.removedor.CanaisAtivosDoUsuario(usuarioID) {
		c, err := v.removedor.ObterCanal(ctx, canalID)
		if err != nil || !c.GeocercaAtiva {
			continue
		}
		if DentroDoRaio(lat, lon, c.CentroLatitude, c.CentroLongitude, c.RaioMetros) {
			continue
		}
		v.removedor.Remover(canalID, usuarioID, "geocerca")
	}
}

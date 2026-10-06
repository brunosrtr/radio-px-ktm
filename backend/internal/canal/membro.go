package canal

import "sync"

// eventosBufferizados é o tamanho do buffer de eventos por membro: grande o
// suficiente para uma transmissão inteira (chunks Opus a ~24kbps por até
// 90s) sem bloquear a goroutine consumidora do hub caso o consumidor da
// conexão WebSocket fique momentaneamente atrás.
const eventosBufferizados = 256

// Membro representa um usuário conectado a um canal, do ponto de vista do
// hub. O handler WebSocket lê Eventos e escreve cada um na conexão.
type Membro struct {
	UsuarioID   string
	Nome        string
	Eventos     chan Evento
	Falha       chan struct{}
	falhaUmaVez sync.Once

	mu         sync.Mutex
	silenciado bool
}

func novoMembro(usuarioID, nome string) *Membro {
	return &Membro{
		UsuarioID: usuarioID,
		Nome:      nome,
		Eventos:   make(chan Evento, eventosBufferizados),
	}
}

// Silenciado indica se o membro optou por não ouvir o canal (FR-19), sem sair
// dele.
func (m *Membro) Silenciado() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.silenciado
}

// DefinirSilenciado atualiza a preferência de silenciamento do membro.
func (m *Membro) DefinirSilenciado(silenciado bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.silenciado = silenciado
}

// enviar entrega um evento ao membro sem bloquear o hub: um ouvinte lento
// perde eventos antigos em vez de travar a distribuição para os demais —
// coerente com a natureza efêmera do áudio (Princípio I).
func (m *Membro) enviar(e Evento) {
	select {
	case m.Eventos <- e:
	default:
		if m.Falha != nil {
			m.falhaUmaVez.Do(func() { close(m.Falha) })
		}
	}
}

package ws

import "github.com/coder/websocket"

// opcoesAccept libera o handshake WebSocket para origens em localhost com
// qualquer porta — necessário porque o navegador envia o header `Origin` da
// porta do Flutter Web (ex.: :9100) ao conectar no backend (:8080), e o
// coder/websocket recusa esse handshake cross-origin por padrão. Clientes
// nativos (app Android/iOS) não enviam Origin, então não são afetados por
// esta checagem de qualquer forma.
func opcoesAccept() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*"},
	}
}

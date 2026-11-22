package transporte

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Limitador conta tentativas por chave (IP) numa janela deslizante, em
// memória — suficiente para a instância única do backend (Princípio VI).
// Protege as rotas públicas de cadastro e recuperação de senha contra abuso.
type Limitador struct {
	mu         sync.Mutex
	maximo     int
	janela     time.Duration
	agora      func() time.Time
	tentativas map[string][]time.Time
}

// NovoLimitador permite até maximo tentativas por chave dentro de janela.
func NovoLimitador(maximo int, janela time.Duration) *Limitador {
	return &Limitador{maximo: maximo, janela: janela, agora: time.Now, tentativas: map[string][]time.Time{}}
}

// ComRelogio troca a fonte de tempo (testes).
func (l *Limitador) ComRelogio(agora func() time.Time) *Limitador {
	l.agora = agora
	return l
}

// Permitir registra uma tentativa da chave e informa se ela cabe na janela.
// Tentativas recusadas não contam, para que a chave volte a ser aceita assim
// que as antigas saírem da janela.
func (l *Limitador) Permitir(chave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	agora := l.agora()
	limite := agora.Add(-l.janela)
	recentes := l.tentativas[chave][:0]
	for _, t := range l.tentativas[chave] {
		if t.After(limite) {
			recentes = append(recentes, t)
		}
	}
	if len(recentes) >= l.maximo {
		l.tentativas[chave] = recentes
		return false
	}
	l.tentativas[chave] = append(recentes, agora)

	// Limpeza oportunista para o mapa não crescer sem limite.
	if len(l.tentativas) > 10_000 {
		for k, ts := range l.tentativas {
			if len(ts) == 0 || !ts[len(ts)-1].After(limite) {
				delete(l.tentativas, k)
			}
		}
	}
	return true
}

// Middleware recusa com 429 quando o IP da requisição passou do limite.
func (l *Limitador) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Permitir(ipDaRequisicao(r)) {
			responderErro(w, http.StatusTooManyRequests, "muitas_tentativas", "muitas tentativas; aguarde alguns minutos e tente de novo")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ipDaRequisicao(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

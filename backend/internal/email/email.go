// Package email envia os e-mails transacionais do backend — hoje, apenas o
// código de recuperação de senha (FR-025 da 003-rede-orbit).
package email

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
)

// Enviador entrega uma mensagem de texto simples a um destinatário.
type Enviador interface {
	Enviar(ctx context.Context, para, assunto, corpo string) error
}

// NovoEnviador devolve o Enviador SMTP configurado, ou nil quando não há
// SMTP — quem usa trata nil como "recuperação de senha indisponível".
func NovoEnviador(cfg config.Config) Enviador {
	if !cfg.SMTPConfigurado() {
		return nil
	}
	return &EnviadorSMTP{cfg: cfg.SMTP}
}

// EnviadorSMTP envia por net/smtp. Autentica só quando há usuário
// configurado (o Mailpit local não exige).
type EnviadorSMTP struct {
	cfg config.SMTP
}

// Enviar implementa Enviador.
func (e *EnviadorSMTP) Enviar(_ context.Context, para, assunto, corpo string) error {
	if strings.ContainsAny(para, "\r\n") || strings.ContainsAny(assunto, "\r\n") {
		return fmt.Errorf("email: cabeçalho inválido")
	}
	endereco := net.JoinHostPort(e.cfg.Host, e.cfg.Porta)
	var autenticacao smtp.Auth
	if e.cfg.Usuario != "" {
		autenticacao = smtp.PlainAuth("", e.cfg.Usuario, e.cfg.Senha, e.cfg.Host)
	}
	mensagem := strings.Join([]string{
		"From: Rádio PX KTM <" + e.cfg.Remetente + ">",
		"To: " + para,
		"Subject: " + assunto,
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		corpo,
	}, "\r\n")
	if err := smtp.SendMail(endereco, autenticacao, e.cfg.Remetente, []string{para}, []byte(mensagem)); err != nil {
		return fmt.Errorf("email: erro ao enviar para o servidor SMTP: %w", err)
	}
	return nil
}

// Mensagem é um e-mail guardado pelo EnviadorMemoria.
type Mensagem struct {
	Para, Assunto, Corpo string
}

// EnviadorMemoria guarda as mensagens em memória, para testes.
type EnviadorMemoria struct {
	mu        sync.Mutex
	Mensagens []Mensagem
}

// Enviar implementa Enviador.
func (e *EnviadorMemoria) Enviar(_ context.Context, para, assunto, corpo string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Mensagens = append(e.Mensagens, Mensagem{Para: para, Assunto: assunto, Corpo: corpo})
	return nil
}

// Ultima devolve a última mensagem enviada para o destinatário, se houver.
func (e *EnviadorMemoria) Ultima(para string) (Mensagem, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := len(e.Mensagens) - 1; i >= 0; i-- {
		if strings.EqualFold(e.Mensagens[i].Para, para) {
			return e.Mensagens[i], true
		}
	}
	return Mensagem{}, false
}

// Total devolve quantas mensagens foram enviadas.
func (e *EnviadorMemoria) Total() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.Mensagens)
}

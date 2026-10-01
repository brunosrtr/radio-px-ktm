# Estabilização do PTT

Base: main `365a6e6`. Data: 2026-09-29.

## Escopo

Correções do ciclo de transmissão conforme FR-003/004/005/007/008, mantendo
Go + WebSocket, Flutter, fila FIFO em memória e nenhuma persistência de voz.
A alteração local pré-existente em `backend/painel/embed.go` foi preservada.

## Comportamento implementado

- Soltar antes do slot cancela a captura sem abrir o microfone.
- Soltar durante a abertura assíncrona agenda a parada após a abertura;
  os chunks tardios não são enviados.
- Corte local pelo limite anunciado e corte independente no servidor.
- Uma captura ativa por remetente; autorização de cancelamento vinculada
  à transmissão ativa da conexão.
- Cancelamento/saída/remoção descarta a captura incompleta, libera sua vaga
  e acorda o consumidor; fechar o hub também descarta seus buffers.
- Atualização de participantes/fila por `estado_canal`.
- Queda detectada ou remoção para a captura, bloqueia o PTT e informa o usuário.

## Validação

Os quatro testes de regressão iniciais falharam antes da correção e passaram
após ela, incluindo execução com `-race`. A suíte completa do backend
passou com PostgreSQL 16 real e `go test -race -p 1 ./... -count=1`, após
a inclusão do teste de liberação imediata de slot cancelado e a ampliação
do teste WebSocket. `git diff --check` não apontou erros de whitespace.

Em banco vazio, a suíte anterior tem uma corrida entre migrações executadas
por pacotes distintos. Para reproduzir os testes sem essa concorrência de
preparação, usar banco de teste dedicado e execução sequencial por pacote:

```sh
cd backend
# DATABASE_URL deve apontar exclusivamente ao banco de teste.
go test -race -p 1 -v ./... -count=1
```

A sincronização das migrações concorrentes fica registrada como débito
anterior a esta etapa. Os testes não devem ser considerados integração
validada quando o helper os pula por indisponibilidade do PostgreSQL.

No app:

```sh
cd app
flutter test
flutter analyze
```

## Aceite manual pendente

Em dois aparelhos, entrar no mesmo canal e verificar: fala completa uma
única vez, soltura antes da autorização, fim aos 90s, liberação da fila,
saída durante uma captura e retorno manual após perda de conexão.
Inspecionar também a ausência de arquivos de voz no dispositivo e servidor.

## Limites desta etapa

- Não altera fila de mensagens para fila de vez. A diferença entre 10 itens
  totais (contrato/código) e 10 aguardando (FR-006) segue pendente.
- Não comprova codec, consumo/latência ou segundo plano em aparelhos reais.
- Reconexão automática, configuração nativa, descarte por idade de áudio em
  redes lentas e entrega confiável de controles sob saturação ficam para as
  etapas seguintes. Nenhum áudio é reenviado após reconexão por esta mudança.

## Verificação para commit — 2026-10-01

A implementação existente foi revisada e o comando
`go test -race -p 1 -v ./... -count=1` terminou sem falhas:
13 testes passaram e 9 testes de integração foram pulados por falha de
autenticação no PostgreSQL local. Esta execução não revalida a integração
com banco descrita no registro anterior. `git diff --check` passou.
O SDK Flutter não foi encontrado no ambiente; T080 e T081 continuam pendentes.
O comentário avulso em `backend/painel/embed.go` ficou fora deste commit.

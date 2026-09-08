# Fase 0 — Pesquisa e Decisões Técnicas

Feature: Rádio PX Digital v1 (`001-radio-px-digital`)

O briefing original da KTM já resolve a maior parte das decisões técnicas
(stack, bibliotecas sugeridas, layout de pastas, protocolo de API/WebSocket,
DDL). Esta fase não parte de incógnitas abertas na especificação — a spec não
tem nenhum `[NEEDS CLARIFICATION]` pendente — mas consolida a *justificativa*
de cada escolha técnica herdada do briefing e resolve o único ponto
deliberadamente deixado em aberto pelo documento: a biblioteca de
armazenamento local da fila offline de posições no app (`drift` **ou** `hive`).

## 1. Fila offline de posições no app: Hive vs. Drift

- **Decision**: Hive.
- **Rationale**: A fila offline de posições (RF31) é uma estrutura simples —
  uma lista de pontos (`latitude`, `longitude`, `precisao_metros`,
  `velocidade_kmh`, `capturado_em`) que só precisa suportar inserir, listar em
  lote e remover por confirmação do servidor. Não há relações entre entidades
  nem consultas complexas no dispositivo. Hive é um armazenamento chave-valor
  leve, sem dependência de SQLite nativo, com custo de inicialização e
  manutenção de schema bem menor que o de um banco relacional embutido. Drift
  (camada reativa sobre SQLite com geração de código e migrações) resolveria
  o mesmo problema, mas com uma complexidade desproporcional ao caso de uso —
  contraria o Princípio VI da constituição (simplicidade adequada ao escopo).
- **Alternatives considered**: Drift (SQLite reativo) — rejeitado por
  overhead de schema/migração para uma fila simples; `shared_preferences` —
  rejeitado por não ser adequado a uma lista que cresce e é podada
  continuamente (não é um armazenamento de chave única); arquivo JSON manual
  — rejeitado por exigir reimplementar controle de concorrência de leitura/
  escrita que o Hive já resolve.

## 2. Hub de canais: goroutine + channel bufferizado por canal

- **Decision**: Um hub em memória no processo Go, com uma goroutine e um
  `chan *Transmissao` (capacidade 10) por canal ativo, exatamente como descrito
  na seção 7 do briefing.
- **Rationale**: A fila do RF05/RF06 (até 10 mensagens, ordem FIFO) mapeia
  diretamente para um channel bufferizado da linguagem — não é necessário
  nenhum componente de fila externo (Redis, RabbitMQ) para o escopo da v1. Uma
  goroutine por canal ativo consome poucos recursos e simplifica o controle de
  concorrência (cada canal só é manipulado pela sua própria goroutine
  consumidora), evitando locks amplos.
- **Alternatives considered**: Fila em Redis (Redis List/Streams) — rejeitada
  na v1 por exigir uma peça de infraestrutura adicional sem necessidade real
  (uma única instância de backend é suficiente para o escopo acadêmico,
  Princípio VI); a interface do hub é isolada (`internal/canal`) justamente
  para permitir essa evolução futura sem reescrever os handlers, caso o
  produto cresça além de uma instância.

## 3. Validação de duração da transmissão no servidor, não só no app

- **Decision**: O backend corta a transmissão ao atingir 90 segundos,
  independentemente do que o app envie.
- **Rationale**: RF04/RNF04 exigem o limite como regra de negócio, não como
  UX. Confiar apenas no cliente permitiria a um app modificado ou com bug
  ultrapassar o limite. O corte no servidor é barato de implementar (basta
  medir o tempo decorrido desde o primeiro chunk da transmissão ativa da
  conexão) e é a única forma de garantir a regra de forma auditável para a
  banca.
- **Alternatives considered**: Confiar apenas na validação client-side —
  rejeitada por não ser uma garantia real do requisito.

## 4. Descarte de áudio: buffer em memória, nunca em disco

- **Decision**: Os chunks de uma transmissão ficam em `[][]byte` no processo
  Go enquanto ela está na fila ou em reprodução; o slice é liberado (`nil`)
  assim que a distribuição aos ouvintes termina.
- **Rationale**: É a única forma de atender RF09/RNF05/RNF06 "por construção,
  não por política", como o próprio briefing exige (seção 6) e como o
  Princípio I da constituição reforça. Qualquer desenho que grave o áudio em
  disco (mesmo temporariamente, mesmo criptografado) para depois apagar seria
  uma violação do requisito, não um detalhe de implementação.
- **Alternatives considered**: Gravar em disco/S3 temporário e apagar após
  reprodução — descartado explicitamente pelo requisito e pela constituição.

## 5. Autenticação: JWT stateless com claims de empresa e papel

- **Decision**: `golang-jwt`, token carregando `usuario_id`, `empresa_id`,
  `papel` (RF21, RNF11, RNF12), validado em middleware antes de qualquer rota
  protegida e antes do upgrade da conexão WebSocket.
- **Rationale**: Evita round-trip ao banco a cada request só para saber a
  empresa e o papel do usuário — a checagem de autorização por empresa
  (RNF12) e por papel (admin vs. motorista) pode ser feita a partir do próprio
  token. Contas são fornecidas pela empresa (sem autocadastro), então não há
  necessidade de fluxos de registro público, refresh token complexo ou OAuth
  de terceiros.
- **Alternatives considered**: Sessões com estado no servidor (cookie +
  tabela de sessões) — rejeitada por adicionar uma tabela e uma camada de
  invalidação sem benefício claro para o escopo da v1.

## 6. Cálculo de geocerca: distância Haversine sobre coordenadas planas

- **Decision**: Verificação de raio usando a fórmula de Haversine entre a
  posição atual do motorista e o centro do canal (`centro_latitude`,
  `centro_longitude`, `raio_metros`), reavaliada a cada atualização de posição
  recebida (não em um polling separado).
- **Rationale**: Simples, sem dependência externa (nenhuma biblioteca de SIG é
  necessária para um raio circular), suficiente para a precisão exigida em
  RF16-RF18 e barato de testar unitariamente (é uma função pura
  `distancia(lat1, lon1, lat2, lon2) float64`), o que atende diretamente o
  Princípio IV (teste obrigatório do cálculo de geocerca).
- **Alternatives considered**: PostGIS (`ST_DWithin`) — poderia simplificar
  consultas geoespaciais mais ricas no futuro, mas adicionaria uma extensão
  de banco não estritamente necessária para um raio circular simples na v1;
  fica registrado como possível evolução, não como requisito atual.

## 7. Compressão de áudio e envio incremental

- **Decision**: Codec Opus, no máximo 24 kbps (RNF14), com envio de chunks
  conforme são gerados pela gravação (não ao final), como já definido na
  seção 10 do briefing.
- **Rationale**: Reduz o consumo de dados em rodovia e a latência percebida
  (RNF07) — o primeiro ouvinte pode começar a receber áudio antes da gravação
  terminar. `flutter_sound` suporta gravação/streaming em Opus nativamente.
- **Alternatives considered**: Gravar em arquivo local e enviar ao final —
  rejeitado por aumentar a latência e por criar um arquivo de áudio em disco,
  o que o Princípio I proíbe mesmo que temporário.

## 8. Ingestão de posições em lote com atualização condicional

- **Decision**: `POST /posicoes` insere todos os pontos do lote em uma única
  operação (`INSERT ... SELECT unnest(...)` ou `COPY`/multi-row insert via
  `pgx`) e atualiza `posicao_atual` apenas se `capturado_em` do lote for mais
  recente que o valor já gravado (`ON CONFLICT ... WHERE EXCLUDED.capturado_em
  > posicao_atual.capturado_em`).
- **Rationale**: Atende RNF15 (lote, não ponto a ponto) e evita que um lote
  atrasado (chegando depois de outro mais recente, comum em trechos sem
  sinal) sobrescreva incorretamente a posição mais atual no mapa da empresa.
  É exatamente o comportamento que RF30/RF29 exigem e que precisa de teste
  automatizado dedicado (Princípio IV).
- **Alternatives considered**: Atualizar `posicao_atual` sempre com o último
  ponto do lote recebido, sem comparação de timestamp — rejeitado porque
  lotes atrasados poderiam regredir a posição exibida no mapa.

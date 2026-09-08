# Rádio PX Digital — Constituição

## Princípios Centrais

### I. Áudio nunca é persistido
Nenhuma mensagem de voz pode ser gravada em disco, banco de dados, log ou
cache, em nenhuma camada do sistema (app, backend ou infraestrutura). A fila
de reprodução e os buffers de áudio existem apenas em memória no processo e
são descartados permanentemente após a reprodução. Se uma solução técnica
parecer exigir persistência de áudio, a implementação para e o problema é
sinalizado como violação de requisito — não é um detalhe a ser contornado.

### II. Uma etapa por vez, com aceite validado
O trabalho é conduzido em etapas sequenciais (ver plano de implementação).
Cada etapa só é considerada concluída quando o projeto compila e o critério
de aceite definido para ela é demonstrado. A etapa seguinte não começa antes
da confirmação explícita do usuário sobre a etapa concluída.

### III. Nada de requisito inventado
Requisitos ambíguos ou não cobertos pela especificação não são resolvidos por
suposição. Toda ambiguidade é registrada explicitamente (seção de
pendências/clarifications) e esclarecida com o usuário antes da
implementação correspondente.

### IV. Teste obrigatório nas regras de negócio críticas
As seguintes regras exigem testes automatizados, pois são o núcleo técnico
defendido no trabalho acadêmico: comportamento da fila de reprodução (FIFO,
limite de 10 mensagens, recusa por fila cheia), limite de participantes por
canal, cálculo e verificação de geocerca, e ingestão em lote de posições
(incluindo atualização condicional de `posicao_atual`).

### V. Dados explícitos, sem ORM
Todo acesso ao PostgreSQL é feito com SQL escrito à mão (via `pgx`), sem uso
de ORM, para que o modelo de dados permaneça explícito e defensável. Migrações
são versionadas e idempotentes.

### VI. Simplicidade adequada ao escopo acadêmico
O hub de canais é local ao processo (uma única instância de backend na v1).
Complexidade adicional (ex.: Redis Pub/Sub para múltiplas instâncias,
particionamento de tabelas) só é introduzida se estritamente necessária e após
as etapas obrigatórias estarem concluídas; a interface do hub deve permitir
essa evolução sem reescrever os handlers.

## Padrões de Idioma e Nomenclatura

Identificadores de domínio (tabelas, campos, tipos, funções relacionadas a
regras de negócio), comentários e documentação de especificação são escritos
em português, usando os termos já fixados neste documento (motorista, canal,
empresa, geocerca, transmissão, fila, participação), para manter coerência com
a defesa do TCC. Nomes de convenções técnicas genéricas da linguagem/framework
(ex.: `main.go`, `Handler`, `Repository`) seguem o idioma usual dessas
convenções.

## Segurança e Configuração

Segredos (credenciais de banco, chave JWT, etc.) só existem em variáveis de
ambiente; nenhum segredo é commitado. Um `.env.example` é mantido versionado e
atualizado a cada nova variável introduzida. Toda rota, exceto login, exige
autenticação via JWT (`usuario_id`, `empresa_id`, `papel`); o acesso a canais é
sempre restrito por autorização de empresa e, quando aplicável, por geocerca.

## Governança

Esta constituição prevalece sobre preferências de implementação ad-hoc.
Alterações a estes princípios exigem justificativa registrada e, quando
afetarem trabalho já em andamento, um plano de migração. Toda revisão de plano
ou de código deve verificar aderência a estes princípios, especialmente aos
Princípios I (áudio nunca persiste) e IV (testes das regras críticas).

**Versão**: 1.0.0 | **Ratificada em**: 2026-09-08 | **Última alteração**: 2026-09-08

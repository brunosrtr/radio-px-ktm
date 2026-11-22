<!--
Sync Impact Report
- Versão: 1.0.0 → 2.0.0 (MAJOR: o modelo de acesso deixa de ser centrado em
  empresa — redefinição da seção Segurança e Configuração)
- Princípios modificados:
  - IV. Teste obrigatório nas regras de negócio críticas (ampliado com as
    regras de leitura Orbit e de acesso a canais da comunidade)
  - VI. Simplicidade adequada ao escopo acadêmico (ampliado com o ponto de
    integração KTM substituível)
- Princípios adicionados:
  - VII. Privacidade em mão dupla na rede Orbit
- Seções alteradas: Padrões de Idioma e Nomenclatura (novos termos fixados),
  Segurança e Configuração (contas abertas, empresa opcional)
- Seções removidas: nenhuma
- Templates: plan-template.md, spec-template.md, tasks-template.md não citam
  princípios específicos — ✅ sem alteração necessária
- Documentos de execução: CLAUDE.md ✅ atualizado
- TODOs: nenhum (empresa opcional confirmada em 2026-11-21 —
  specs/003-rede-orbit/spec.md FR-033)
-->
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
canal, cálculo e verificação de geocerca, ingestão em lote de posições
(incluindo atualização condicional de `posicao_atual`), deduplicação de
leituras Orbit por intervalo mínimo, ingestão em lote de leituras Orbit
(incluindo atualização condicional da última posição do Orbit) e controle de
acesso a canais da comunidade (visibilidade pública/privada e convite).

### V. Dados explícitos, sem ORM
Todo acesso ao PostgreSQL é feito com SQL escrito à mão (via `pgx`), sem uso
de ORM, para que o modelo de dados permaneça explícito e defensável. Migrações
são versionadas e idempotentes.

### VI. Simplicidade adequada ao escopo acadêmico
O hub de canais é local ao processo (uma única instância de backend na v1).
Complexidade adicional (ex.: Redis Pub/Sub para múltiplas instâncias,
particionamento de tabelas) só é introduzida se estritamente necessária e após
as etapas obrigatórias estarem concluídas; a interface do hub deve permitir
essa evolução sem reescrever os handlers. Pelo mesmo motivo, o encaminhamento
de leituras para a KTM e a definição do sinal do Orbit (identificação e
formato) ficam atrás de uma interface única e substituível: a versão de
demonstração (Orbit virtual e integração falsa) deve poder ser trocada pela
real sem alterar o restante do sistema.

### VII. Privacidade em mão dupla na rede Orbit
Quem atua como ponto de transmissão nunca vê dados dos Orbits ou caminhões que
seu celular leu; e a KTM, no painel, nunca vê dados pessoais (nome, apelido,
e-mail, telefone) de quem fez a leitura. Atuar como ponto de transmissão exige
consentimento explícito, separado do aceite dos termos de uso, e pode ser
desligado a qualquer momento sem perder acesso à rádio.

## Padrões de Idioma e Nomenclatura

Identificadores de domínio (tabelas, campos, tipos, funções relacionadas a
regras de negócio), comentários e documentação de especificação são escritos
em português, usando os termos já fixados neste documento (usuário, motorista, canal,
convite, empresa, geocerca, transmissão, fila, participação, Orbit, Orbit
virtual, leitura Orbit, ponto de transmissão, ponto fixo de leitura), para
manter coerência com a defesa do TCC. "Transmissão" designa sempre a
transmissão de voz; o celular que retransmite leituras Orbit é um "ponto de
transmissão" e o que ele envia é uma "leitura". Nomes de convenções técnicas genéricas da linguagem/framework
(ex.: `main.go`, `Handler`, `Repository`) seguem o idioma usual dessas
convenções.

## Segurança e Configuração

Segredos (credenciais de banco, chave JWT, etc.) só existem em variáveis de
ambiente; nenhum segredo é commitado. Um `.env.example` é mantido versionado e
atualizado a cada nova variável introduzida. Toda rota, exceto cadastro, login e
recuperação de senha, exige autenticação via JWT (`usuario_id`, `papel` e,
quando o usuário tiver vínculo, `empresa_id`). Qualquer pessoa pode criar
conta; a empresa é um vínculo opcional. O acesso a canais é restrito pela
visibilidade do canal (público, ou privado mediante convite) e, quando
aplicável, por geocerca. O painel Rede Orbit e o modo Orbit virtual são
restritos aos papéis de administração da KTM e de demonstração.

## Governança

Esta constituição prevalece sobre preferências de implementação ad-hoc.
Alterações a estes princípios exigem justificativa registrada e, quando
afetarem trabalho já em andamento, um plano de migração. Toda revisão de plano
ou de código deve verificar aderência a estes princípios, especialmente aos
Princípios I (áudio nunca persiste) e IV (testes das regras críticas).

**Versão**: 2.0.0 | **Ratificada em**: 2026-09-08 | **Última alteração**: 2026-11-21

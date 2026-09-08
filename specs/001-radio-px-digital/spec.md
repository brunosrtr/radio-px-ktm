# Feature Specification: Rádio PX Digital — Canais de Voz, Localização e Painel da Empresa (v1)

**Feature Branch**: `001-radio-px-digital`
**Created**: 2026-09-08
**Status**: Draft
**Input**: User description: "Aplicativo de rádio amador digital para a KTM Indústria Mecatrônica: comunicação por voz push-to-talk entre motoristas organizados em canais (sem texto, sem replay de áudio), rastreamento contínuo de localização em segundo plano com fila offline, e painel web da empresa para acompanhar posição atual e trajeto dos motoristas. V1 não integra hardware Orbit — localização vem do GPS do celular, comunicação trafega por internet móvel."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comunicação por voz em canais (Priority: P1)

Um motorista autenticado entra em um canal e conversa por voz com os demais motoristas conectados a ele, no estilo rádio PX: fala quando há espaço na fila, ouve as mensagens na ordem em que chegaram, e nunca tem como reouvir uma mensagem já reproduzida.

**Why this priority**: É a proposta central do produto — sem isso não existe "rádio digital". As demais capacidades (gestão de canais, geocerca, localização, painel) existem para sustentar e restringir esse loop de comunicação, não para substituí-lo.

**Independent Test**: Com dois motoristas de teste já inseridos no mesmo canal, um deles grava e envia uma mensagem de voz; o outro a recebe e ouve do início ao fim, na ordem correta, e a mensagem deixa de existir em qualquer lugar do sistema assim que termina de tocar — sem que nenhuma tela de gestão de canais precise existir ainda.

**Acceptance Scenarios**:

1. **Given** dois motoristas conectados ao mesmo canal, **When** o motorista A transmite uma mensagem de voz, **Then** o motorista B ouve a mensagem do início ao fim, na ordem em que foi enviada.
2. **Given** um canal com uma mensagem em reprodução, **When** um segundo motorista tenta transmitir, **Then** a nova mensagem entra na fila e só é reproduzida após a mensagem atual terminar.
3. **Given** uma fila com 10 mensagens aguardando, **When** um motorista tenta transmitir uma 11ª mensagem, **Then** o sistema informa que a fila está cheia e impede o início de uma nova gravação.
4. **Given** um motorista transmitindo, **When** a transmissão atinge 90 segundos, **Then** ela é encerrada automaticamente.
5. **Given** uma mensagem que acabou de ser reproduzida, **When** qualquer motorista tenta acessá-la novamente, **Then** não há nenhum meio de reouvi-la — a mensagem não existe mais em lugar nenhum do sistema.
6. **Given** um motorista ouvindo uma transmissão, **When** a reprodução começa, **Then** o nome do motorista remetente é exibido.

---

### User Story 2 - Gestão de canais e controle de acesso pela empresa (Priority: P1)

Um administrador da empresa cria e gerencia os canais de comunicação de seus motoristas: define nome, descrição, limite de participantes, e decide se o canal também aceita motoristas de empresas parceiras.

**Why this priority**: Sem canais geridos pela empresa, a comunicação por voz (User Story 1) não tem contexto organizacional nem controle de acesso — é o que torna o produto utilizável por uma frota real, com múltiplas empresas envolvidas.

**Independent Test**: Um administrador cria um canal e define seu limite de participantes; motoristas da própria empresa passam a ver o canal na lista, enquanto motoristas de uma empresa não autorizada não o veem — sem depender de a comunicação por voz ou a geolocalização estarem funcionando.

**Acceptance Scenarios**:

1. **Given** um administrador autenticado, **When** ele cria um canal com nome, descrição e limite de participantes, **Then** o canal passa a existir e fica disponível aos motoristas da própria empresa.
2. **Given** um canal privado da empresa A, **When** um motorista da empresa B consulta sua lista de canais, **Then** esse canal não aparece.
3. **Given** um canal marcado como compartilhado e liberado para a empresa B, **When** um motorista da empresa B consulta sua lista de canais, **Then** o canal aparece e ele pode entrar.
4. **Given** um canal com limite de 10 participantes já ocupado, **When** um 11º motorista tenta entrar, **Then** a entrada é recusada com aviso do motivo.
5. **Given** um motorista dentro de um canal, **When** ele opta por silenciar o canal, **Then** ele deixa de ouvir novas transmissões desse canal mas continua participando dele (não é removido).
6. **Given** um motorista dentro de um canal, **When** ele opta por sair manualmente, **Then** ele deixa de fazer parte do canal e o registro de saída indica o motivo "manual".
7. **Given** um motorista participando de um canal, **When** ele seleciona outro canal disponível, **Then** ele passa a ouvir e falar no novo canal.

---

### User Story 3 - Restrição de canais por geolocalização (geocerca) (Priority: P2)

O acesso a determinados canais é restrito à área geográfica onde o serviço faz sentido (por exemplo, um pátio ou uma região de operação). Um motorista só entra em um canal com geocerca se estiver dentro do raio configurado, e é removido automaticamente — com aviso — se sair dessa área.

**Why this priority**: Refina o controle de acesso da User Story 2 para o cenário real da KTM (frotas em rodovia, canais por região). Não bloqueia a comunicação básica, mas é obrigatório para o uso pretendido do produto.

**Independent Test**: Com um canal configurado com geocerca de um raio determinado, um motorista simulado dentro do raio consegue entrar no canal; ao simular deslocamento para fora do raio, ele é removido automaticamente do canal e recebe um aviso, sem que nenhuma outra funcionalidade precise estar presente.

**Acceptance Scenarios**:

1. **Given** um canal com geocerca ativa e um motorista dentro do raio configurado, **When** ele tenta entrar no canal, **Then** a entrada é aceita.
2. **Given** um canal com geocerca ativa e um motorista fora do raio configurado, **When** ele tenta entrar no canal, **Then** a entrada é recusada informando o motivo.
3. **Given** um motorista participando de um canal com geocerca, **When** sua localização sai do raio configurado, **Then** ele é removido automaticamente do canal, recebe um aviso do motivo, e o registro de participação indica o motivo "geocerca".

---

### User Story 4 - Rastreamento contínuo de localização (Priority: P2)

O aplicativo do motorista registra periodicamente sua posição, mesmo em segundo plano e em trechos sem sinal, guardando os pontos localmente até conseguir sincronizá-los com o servidor em lote.

**Why this priority**: É a segunda capacidade central do produto (ao lado da comunicação por voz) e é pré-requisito de dados para o painel da empresa (User Story 5) e para a geocerca (User Story 3), mas pode ser construída e validada de forma independente, antes de existir qualquer visualização.

**Independent Test**: Com o aplicativo em modo avião por alguns minutos e a conexão restaurada em seguida, todos os pontos de localização capturados durante o período offline aparecem registrados no servidor, com o horário real de captura preservado (não o horário de chegada).

**Acceptance Scenarios**:

1. **Given** o motorista logado com o app em segundo plano, **When** o tempo passa, **Then** novas posições continuam sendo capturadas periodicamente.
2. **Given** o dispositivo sem conexão com a internet, **When** o app captura novas posições, **Then** elas são armazenadas localmente sem serem perdidas.
3. **Given** posições acumuladas localmente durante um período offline, **When** a conexão é restabelecida, **Then** todas elas são enviadas ao servidor em um envio em lote (não uma por uma) e removidas da fila local apenas após confirmação.
4. **Given** o motorista parado no mesmo local por um longo período, **When** o sistema avalia se deve capturar uma nova posição, **Then** ele evita registrar pontos redundantes no mesmo local.

---

### User Story 5 - Painel da empresa com mapa e trajeto (Priority: P3)

Um administrador acessa um painel web onde vê, num mapa, a posição atual de todos os seus motoristas, e pode consultar o trajeto percorrido por um motorista específico em um período de tempo.

**Why this priority**: É o valor entregue à empresa a partir dos dados coletados na User Story 4; depende dela para ter dados reais, por isso vem depois, mas fecha o ciclo de valor do produto para quem contrata o serviço.

**Independent Test**: Com posições de teste já registradas para dois motoristas de empresas diferentes, um administrador da empresa A abre o painel e vê no mapa apenas os motoristas da empresa A; ao consultar o trajeto de um motorista num intervalo de datas, vê o caminho percorrido nesse intervalo.

**Acceptance Scenarios**:

1. **Given** um administrador autenticado, **When** ele abre o painel, **Then** vê no mapa a posição mais recente de cada motorista ativo da sua própria empresa, e não de outras empresas.
2. **Given** um motorista com histórico de posições, **When** o administrador consulta o trajeto desse motorista em um intervalo de datas, **Then** o painel exibe o caminho percorrido nesse intervalo.
3. **Given** um motorista sem nenhuma posição registrada no período consultado, **When** o administrador consulta o trajeto, **Then** o painel informa que não há dados para o período, sem erro.

---

### Edge Cases

- O que acontece quando dois motoristas tentam entrar no mesmo canal exatamente no instante em que o limite de participantes está prestes a ser atingido? Apenas um deve conseguir entrar; o outro recebe recusa.
- O que acontece quando a fila de reprodução de um canal atinge o limite exatamente enquanto uma mensagem termina de tocar? O espaço liberado deve ficar disponível de forma consistente, sem que o motorista perceba comportamento incorreto.
- O que acontece se o motorista perder conexão no meio de uma transmissão de voz? A transmissão incompleta é descartada — nunca fica "perdida" armazenada em algum lugar.
- O que acontece se um canal ficar sem nenhum motorista conectado? O canal deixa de consumir recursos de comunicação em tempo real, mas continua existindo para a empresa gerenciar.
- O que acontece se a empresa revogar o acesso de uma empresa parceira a um canal compartilhado enquanto há motoristas dessa parceira conectados? Ver Assunções.
- O que acontece se o aplicativo tentar transmitir voz ou enviar posições sem uma sessão autenticada válida? A ação é recusada e o motorista é levado de volta à tela de login.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O sistema DEVE permitir que motoristas autenticados conectados ao mesmo canal se comuniquem por voz.
- **FR-002**: O sistema DEVE continuar recebendo e reproduzindo mensagens de voz mesmo com o aplicativo do motorista em segundo plano.
- **FR-003**: O sistema DEVE permitir que um motorista envie uma mensagem de voz sempre que houver espaço na fila de reprodução do canal.
- **FR-004**: O sistema DEVE limitar cada mensagem de voz a no máximo 90 segundos, interrompendo a transmissão automaticamente ao atingir esse limite.
- **FR-005**: O sistema DEVE manter uma fila de espera quando já houver uma mensagem em reprodução no canal, reproduzindo as mensagens estritamente na ordem de chegada.
- **FR-006**: O sistema DEVE limitar a fila de cada canal a no máximo 10 mensagens simultâneas aguardando reprodução.
- **FR-007**: O sistema DEVE informar o motorista quando a fila do canal estiver cheia e impedir o início de uma nova gravação nesse momento.
- **FR-008**: O sistema DEVE excluir permanentemente cada mensagem de voz imediatamente após sua reprodução, sem deixar cópia em nenhum armazenamento persistente (disco, banco de dados, log ou cache). Esta é uma restrição inegociável do produto, não um detalhe de implementação.
- **FR-009**: O sistema DEVE permitir que cada empresa crie, edite e gerencie seus próprios canais de comunicação.
- **FR-010**: O sistema DEVE permitir que a empresa determine quais canais ficam disponíveis para seus motoristas.
- **FR-011**: O sistema DEVE permitir que o motorista alterne entre os canais disponibilizados para ele.
- **FR-012**: O sistema DEVE permitir que a empresa determine, por canal, se motoristas de outras empresas podem participar dele.
- **FR-013**: O sistema DEVE permitir que motoristas de empresas diferentes se comuniquem entre si nos canais marcados como compartilhados.
- **FR-014**: O sistema DEVE usar a localização atual do motorista para determinar quais canais com restrição geográfica estão disponíveis para ele.
- **FR-015**: O sistema DEVE liberar o acesso a canais com geocerca apenas quando o motorista estiver dentro do raio configurado para aquele canal.
- **FR-016**: O sistema DEVE remover automaticamente um motorista de um canal com geocerca assim que ele sair da área permitida.
- **FR-017**: O sistema DEVE informar o motorista quando ele for removido de um canal por geocerca.
- **FR-018**: O sistema DEVE permitir que o motorista saia manualmente de um canal a qualquer momento.
- **FR-019**: O sistema DEVE permitir que o motorista silencie um canal sem precisar sair dele.
- **FR-020**: O sistema DEVE exigir login com credenciais fornecidas pela empresa para acessar qualquer funcionalidade do aplicativo, e DEVE impedir qualquer acesso a canais por usuários não autenticados.
- **FR-021**: O sistema DEVE identificar, pelo nome, o motorista responsável por cada transmissão de voz, para os demais participantes do canal.
- **FR-022**: O sistema DEVE permitir que a empresa configure o limite de participantes de um canal escolhendo entre 5, 10, 15 ou 20.
- **FR-023**: O sistema DEVE impedir a entrada de um motorista em um canal que já atingiu seu limite de participantes.
- **FR-024**: O sistema DEVE registrar periodicamente a posição do motorista, inclusive com o aplicativo em segundo plano.
- **FR-025**: O sistema DEVE armazenar o histórico de posições de cada motorista por pelo menos 90 dias.
- **FR-026**: O sistema DEVE permitir que a empresa visualize, num mapa, a posição atual de cada um de seus motoristas.
- **FR-027**: O sistema DEVE permitir que a empresa consulte o trajeto percorrido por um motorista específico dentro de um período de datas.
- **FR-028**: O sistema DEVE armazenar as posições localmente no dispositivo quando não houver conexão, sincronizando-as com o servidor assim que a conexão for restabelecida.
- **FR-029**: O sistema DEVE restringir cada motorista aos canais autorizados pela sua própria empresa, incluindo os canais compartilhados liberados para ela.
- **FR-030**: O sistema DEVE suportar pelo menos 20 motoristas conectados simultaneamente em um mesmo canal.
- **FR-031**: O sistema DEVE priorizar a menor latência possível entre o momento em que um motorista fala e o momento em que os demais participantes ouvem a mensagem.
- **FR-032**: O sistema DEVE comprimir o áudio de transmissão para um consumo de dados compatível com conexão móvel em rodovia, a no máximo 24 kbps.
- **FR-033**: O sistema DEVE enviar as posições coletadas em lotes, nunca em uma requisição por ponto de localização.
- **FR-034**: O sistema DEVE estar disponível como aplicativo para Android e iOS.

*Fora do escopo desta especificação (v1)*: comunicação por voz via hardware Orbit, localização proveniente do hardware Orbit e mensagens de texto entre motoristas. Essas capacidades pertencem a uma v2 e dependem de definições técnicas que a KTM ainda não forneceu.

### Key Entities *(include if feature involves data)*

- **Empresa**: organização cliente do serviço; dona de motoristas, administradores e canais; pode liberar seus canais para outras empresas parceiras.
- **Usuário**: pessoa autenticada no sistema, com papel de motorista ou administrador, sempre vinculada a uma empresa.
- **Canal**: espaço de comunicação por voz, pertencente a uma empresa, com nome, descrição, limite de participantes, tipo de acesso (privado ou compartilhado) e, opcionalmente, uma geocerca (centro e raio).
- **Liberação de canal para empresa parceira**: relação que autoriza motoristas de uma empresa diferente da dona a acessar um canal compartilhado.
- **Participação em canal**: registro de um motorista ativo (ou que já esteve) em um canal, incluindo quando entrou, quando saiu e o motivo da saída (manual, geocerca, desconexão ou encerramento do canal).
- **Preferência de canal**: indica se um motorista silenciou um canal específico.
- **Posição**: ponto de localização de um motorista em um instante específico, incluindo o horário real de captura (que pode ser bem anterior ao horário de recebimento pelo servidor, em trechos sem sinal).
- **Posição atual**: última posição conhecida de cada motorista, usada para alimentar o mapa do painel da empresa.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Em condições normais de rede móvel, o tempo entre um motorista terminar de falar e os demais participantes do canal começarem a ouvir a mensagem é inferior a 2 segundos.
- **SC-002**: O sistema suporta pelo menos 20 motoristas conversando no mesmo canal simultaneamente, sem degradação perceptível de qualidade de áudio ou aumento de atraso.
- **SC-003**: Em qualquer inspeção do sistema (banco de dados, arquivos, logs), nenhuma mensagem de voz já reproduzida pode ser encontrada ou reproduzida novamente — 0% de mensagens recuperáveis após o primeiro playback.
- **SC-004**: 100% das posições capturadas durante um período sem conexão de até várias horas chegam ao servidor após a reconexão, preservando o horário real de captura.
- **SC-005**: Um administrador consegue ver a posição atual de todos os seus motoristas no painel com no máximo alguns minutos de atraso em relação à última posição real capturada.
- **SC-006**: Um motorista que sai da área permitida de um canal com geocerca é removido do canal e avisado em poucos segundos após a mudança de localização ser detectada pelo aplicativo.
- **SC-007**: Um motorista consegue abrir o aplicativo, escolher um canal autorizado e começar a falar em menos de 10 segundos, sem passar por telas de configuração adicionais.

## Assumptions

- A empresa fornece as contas (login) de motoristas e administradores; não há autocadastro pelos próprios motoristas na v1.
- A frequência de captura de localização considerada é de aproximadamente 200 metros ou 30 segundos de deslocamento, o que ocorrer primeiro, quando o motorista está em movimento; parado, novos pontos não são gerados. Essa frequência e a política de retenção (assumida em 90 dias) ainda dependem de confirmação final com a KTM (ver Pendências).
- O acompanhamento pela empresa é feito por um painel web; a v1 assume que a empresa não precisa de um aplicativo administrativo nativo separado. Essa decisão também depende de confirmação final com a KTM (ver Pendências).
- Revogar o acesso de uma empresa parceira a um canal compartilhado não desconecta imediatamente motoristas que já estejam participando de uma conversa em andamento; apenas impede novas entradas a partir daquele momento.
- A comunicação com o hardware Orbit está fora do escopo desta especificação e será tratada em uma v2, após a KTM definir banda e protocolo do hardware.
- Toda comunicação entre motoristas é por voz; mensagens de texto não fazem parte do produto.
- Não existe histórico nem possibilidade de reprodução repetida de mensagens de voz — isso é uma característica definida do produto, não uma limitação a ser resolvida.

## Pendências para validar com a KTM

Estas questões não bloqueiam a implementação da v1 (já existe um valor padrão assumido para cada uma, listado acima), mas devem ser confirmadas com a empresa antes da entrega final:

1. A fila de 10 mensagens de até 90 segundos permite um atraso de até 15 minutos entre uma mensagem ser falada e ser ouvida, o que pode contrariar a proposta de comunicação em tempo real do rádio PX. Vale propor à KTM reduzir o limite de gravação (por exemplo, para 30 segundos) ou descartar mensagens que envelheçam demais na fila.
2. Confirmar a frequência de coleta de localização esperada pela empresa e a política de retenção do histórico de posições.
3. Confirmar se a empresa realmente pretende usar um painel web ou prefere que o acompanhamento fique dentro de um aplicativo administrativo nativo.
4. Para a v2: levantar junto à KTM a banda disponível no link do hardware Orbit, sua capacidade de transportar áudio, e o protocolo que o aplicativo usará para se comunicar com o hardware.

# Feature Specification: Rede Orbit — Rede Aberta de Caminhoneiros e Retransmissão Colaborativa de Sinais Orbit

**Feature Branch**: `003-rede-orbit`
**Created**: 2026-11-21
**Status**: Draft
**Input**: User description: "A KTM tem um hardware chamado Orbit, instalado nos caminhões, que envia a localização para a KTM quando passa perto de pontos fixos de leitura espalhados pelas rodovias (os pontos 'bipam' quando o hardware se aproxima). O app deve transformar todo celular conectado em um ponto de transmissão móvel que faz esse 'bip' e envia as informações do caminhão para a KTM. Para isso o app deve ser uma rede social aberta para caminhoneiros: qualquer pessoa baixa, cria conta e cria canais — quanto mais gente usando, melhor o funcionamento do hardware. Como ainda não temos o hardware, deve ser possível demonstrar o bip usando outros celulares no lugar do Orbit, para apresentar à KTM. O app deve ter UX/UI melhor e mais intuitiva, adequada ao público geral."

## Contexto e relação com a especificação 001

A especificação [001-radio-px-digital](../001-radio-px-digital/spec.md) descreve um produto **centrado na empresa**: a empresa cria as contas dos motoristas, cria e libera canais e acompanha seus motoristas no painel. Esta especificação muda o propósito do produto:

- O **objetivo de negócio** passa a ser ampliar a rede de leitura do hardware Orbit. Cada celular com o aplicativo funciona como um **ponto de transmissão** móvel, complementando os pontos fixos de leitura que a KTM já possui.
- A **rádio PX** (canais de voz) continua existindo e é o motivo pelo qual as pessoas instalam e mantêm o aplicativo aberto. A retransmissão Orbit acontece em segundo plano, enquanto a pessoa usa o aplicativo.
- O aplicativo passa a ser **aberto ao público**: qualquer pessoa cria a própria conta e pode criar canais.

Situação dos requisitos da 001 nesta especificação:

| Situação | Requisitos da 001 |
|---|---|
| **Mantidos sem alteração** | FR-001 a FR-008 (voz, fila, áudio nunca persistido), FR-011, FR-014 a FR-019 (geocerca, sair, silenciar), FR-021, FR-023, FR-030 a FR-034 |
| **Substituídos por esta especificação** | FR-009, FR-010, FR-012, FR-013, FR-020, FR-022, FR-029 (contas e canais deixam de depender de empresa) |
| **Mantidos apenas para usuários com empresa** | FR-024 a FR-028 (rastreamento do trajeto do próprio usuário) e FR-026/FR-027 (painel da empresa) — ver FR-033 e FR-034 desta especificação |

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Celular como ponto de transmissão do Orbit (Priority: P1)

Uma pessoa com o aplicativo instalado e aberto (em primeiro ou segundo plano) passa perto de um caminhão equipado com Orbit. O celular dela detecta o sinal do Orbit, faz o "bip" (aviso sonoro e vibração curtos, que a pessoa pode desligar) e envia para a KTM a identificação do Orbit detectado junto com a localização do próprio celular e o horário da leitura. A pessoa não precisa fazer nada para isso acontecer.

**Why this priority**: É a razão de existir do produto: aumentar os pontos de leitura do Orbit. Sem isso, o aplicativo é apenas uma rádio.

**Independent Test**: Com um Orbit (real ou virtual, ver US2) ligado e um celular com o aplicativo a poucos metros, verificar que a leitura chega ao servidor com a identificação correta do Orbit, a localização do celular e o horário da captura, sem nenhuma ação do usuário.

**Acceptance Scenarios**:

1. **Given** um usuário autenticado que autorizou atuar como ponto de transmissão, **When** um Orbit entra no alcance do celular, **Then** o celular registra uma leitura (identificação do Orbit, localização do celular, horário, intensidade do sinal) e a envia ao servidor.
2. **Given** um Orbit que continua no alcance por vários minutos (ex.: caminhões lado a lado no posto), **When** o celular continua detectando o mesmo Orbit, **Then** o celular não envia uma leitura a cada detecção, mas no máximo uma leitura por Orbit a cada intervalo mínimo definido.
3. **Given** um celular sem conexão com a internet, **When** ele detecta um Orbit, **Then** a leitura é guardada no dispositivo e enviada quando a conexão voltar, preservando o horário real da detecção.
4. **Given** o aplicativo em segundo plano (tela apagada ou outro aplicativo aberto), **When** um Orbit entra no alcance, **Then** a leitura acontece mesmo assim, dentro das limitações do sistema operacional informadas ao usuário.
5. **Given** um usuário que desativou a função de ponto de transmissão, **When** um Orbit entra no alcance, **Then** nenhuma leitura é feita.

---

### User Story 2 - Orbit virtual para demonstração (Priority: P1)

Para apresentar a ideia à KTM sem ter o hardware, um celular com o aplicativo pode ser colocado em **modo Orbit virtual**: ele passa a emitir um sinal equivalente ao de um Orbit, com uma identificação de Orbit escolhida (ex.: `ORBIT-0042`) e, opcionalmente, uma placa de caminhão fictícia. Os demais celulares com o aplicativo tratam esse sinal exatamente como tratariam um Orbit real.

**Why this priority**: Sem o hardware, é a única forma de demonstrar o funcionamento à KTM, que é o objetivo imediato do projeto.

**Independent Test**: Com dois celulares — um em modo Orbit virtual e outro como ponto de transmissão comum — aproximar os aparelhos e verificar que o segundo faz o bip e a leitura aparece no painel.

**Acceptance Scenarios**:

1. **Given** um usuário com acesso ao modo de demonstração, **When** ele ativa o modo Orbit virtual e informa uma identificação de Orbit, **Then** o celular passa a emitir o sinal de Orbit e a tela mostra claramente que o modo está ativo.
2. **Given** um celular em modo Orbit virtual, **When** outro celular com o aplicativo se aproxima, **Then** o outro celular faz o bip e registra a leitura como se fosse um Orbit real.
3. **Given** um celular em modo Orbit virtual, **When** o usuário desativa o modo ou fecha o aplicativo, **Then** o celular para de emitir o sinal.
4. **Given** um usuário comum (sem acesso ao modo de demonstração), **When** ele navega pelo aplicativo, **Then** o modo Orbit virtual não aparece para ele.

---

### User Story 3 - Painel "Rede Orbit" ao vivo (Priority: P1)

A KTM acompanha, num painel web com mapa, as leituras de Orbit chegando em tempo real: onde cada Orbit foi lido pela última vez, por qual ponto de transmissão, e quantos pontos de transmissão estão ativos naquele momento. O painel também mostra uma comparação entre a cobertura dos pontos fixos de leitura (simulados na demonstração) e a cobertura somando a rede de celulares.

**Why this priority**: É a tela que será mostrada na apresentação à KTM e que comprova o valor da ideia.

**Independent Test**: Gerar leituras com um Orbit virtual e verificar que elas aparecem no mapa em poucos segundos, com o contador de pontos ativos e a última posição de cada Orbit atualizados.

**Acceptance Scenarios**:

1. **Given** o painel aberto, **When** um ponto de transmissão envia uma leitura, **Then** o mapa mostra a leitura em poucos segundos, com um destaque visual do "bip", e atualiza a última posição conhecida daquele Orbit.
2. **Given** o painel aberto, **When** um administrador seleciona um Orbit, **Then** o painel mostra o histórico de leituras daquele Orbit num período (trajeto aproximado) e quantos pontos de transmissão diferentes o leram.
3. **Given** pontos fixos de leitura cadastrados, **When** o administrador ativa a comparação de cobertura, **Then** o mapa diferencia as leituras feitas por pontos fixos das leituras feitas por celulares.
4. **Given** um usuário que não é administrador da KTM, **When** ele tenta acessar o painel, **Then** o acesso é negado.
5. **Given** o painel, **When** ele exibe informações sobre pontos de transmissão, **Then** não identifica pessoalmente os usuários que fizeram as leituras (nome, telefone ou e-mail).

---

### User Story 4 - Cadastro aberto e primeiros passos (Priority: P1)

Qualquer pessoa baixa o aplicativo e cria a própria conta em menos de dois minutos, sem precisar de convite de empresa. Na primeira abertura, o aplicativo explica em poucas telas o que é a rádio, como o celular ajuda a rede Orbit e por que cada permissão (microfone, localização, Bluetooth) é necessária, antes de pedi-la.

**Why this priority**: Sem cadastro aberto não há rede; e um primeiro contato confuso faz a pessoa desistir antes de virar um ponto de transmissão.

**Independent Test**: Instalar o aplicativo num celular sem conta, criar uma conta, passar pela apresentação inicial e chegar à lista de canais com as permissões concedidas.

**Acceptance Scenarios**:

1. **Given** uma pessoa sem conta, **When** ela informa nome, apelido de rádio, e-mail e senha e aceita os termos de uso, **Then** a conta é criada e ela entra no aplicativo.
2. **Given** um e-mail já cadastrado, **When** alguém tenta criar uma conta com ele, **Then** o cadastro é recusado com uma mensagem clara e a opção de recuperar a senha.
3. **Given** a primeira abertura após o cadastro, **When** o aplicativo vai pedir uma permissão, **Then** ele primeiro mostra uma tela curta explicando para que ela serve.
4. **Given** um usuário que negou uma permissão, **When** ele usa o aplicativo, **Then** as funções que dependem dela ficam indicadas como indisponíveis, com um atalho para conceder a permissão, sem bloquear o resto do aplicativo.
5. **Given** um usuário que esqueceu a senha, **When** ele solicita a recuperação, **Then** ele consegue definir uma nova senha a partir do e-mail cadastrado.
6. **Given** um usuário com conta, **When** ele solicita a exclusão da conta, **Then** a conta e seus dados pessoais são removidos.

---

### User Story 5 - Canais criados pela comunidade (Priority: P2)

Qualquer usuário cria um canal de voz, escolhendo nome, descrição, limite de participantes e se o canal é **público** (aparece na busca para todos) ou **privado** (só entra quem tem o convite, por código ou QR Code). Opcionalmente, o canal pode ter uma geocerca, como na 001. Os usuários encontram canais buscando pelo nome ou vendo os canais públicos próximos à sua localização.

**Why this priority**: Faz o aplicativo funcionar como rede social e dá motivo para as pessoas voltarem, mas a demonstração à KTM pode ser feita com canais de exemplo.

**Independent Test**: Um usuário cria um canal privado, compartilha o convite, um segundo usuário entra pelo convite e os dois conversam por voz.

**Acceptance Scenarios**:

1. **Given** um usuário autenticado, **When** ele cria um canal público, **Then** o canal aparece na busca e na lista de canais próximos (se tiver geocerca) para os demais usuários.
2. **Given** um canal privado, **When** um usuário sem convite procura o canal, **Then** o canal não aparece para ele.
3. **Given** o convite de um canal privado (código ou QR Code), **When** um usuário usa o convite, **Then** ele passa a ter acesso ao canal.
4. **Given** o criador de um canal, **When** ele remove um participante, **Then** o participante sai do canal e não consegue voltar enquanto estiver removido.
5. **Given** o criador de um canal, **When** ele edita ou apaga o canal, **Then** as mudanças valem para todos os participantes.
6. **Given** um canal que atingiu o limite de participantes, **When** outro usuário tenta entrar, **Then** a entrada é recusada (regra da 001 mantida).

---

### User Story 6 - Uso simples e seguro enquanto dirige (Priority: P2)

O aplicativo é organizado em poucas áreas fáceis de encontrar (canais, explorar, minha contribuição e perfil). A tela do canal ativo é pensada para quem está ao volante: botão de falar grande, informações essenciais legíveis de relance e nenhuma ação importante escondida em menus.

**Why this priority**: O público agora é geral, não treinado por uma empresa. A usabilidade decide se a pessoa continua usando — e, portanto, se continua sendo um ponto de transmissão.

**Independent Test**: Pedir a uma pessoa que nunca usou o aplicativo para entrar num canal e falar; medir o tempo e os erros, sem ajuda.

**Acceptance Scenarios**:

1. **Given** um usuário no canal ativo, **When** ele quer falar, **Then** o botão de falar ocupa uma área grande da tela e funciona com um único toque contínuo.
2. **Given** qualquer tela principal, **When** o usuário quer ir para outra área do aplicativo, **Then** ele chega a ela com no máximo um toque.
3. **Given** o sistema do celular em modo escuro ou com fonte ampliada, **When** o aplicativo é aberto, **Then** ele respeita essas preferências sem quebrar o layout.

---

### User Story 7 - Minha contribuição (Priority: P3)

O usuário vê quanto está ajudando a rede: quantos Orbits o celular dele leu no dia, na semana e no total. Ele também vê se a função de ponto de transmissão está ativa e funcionando.

**Why this priority**: Incentiva a pessoa a manter o aplicativo ativo, mas não é necessário para a demonstração.

**Independent Test**: Após algumas leituras de um Orbit virtual, verificar que os contadores do usuário aumentam.

**Acceptance Scenarios**:

1. **Given** um usuário que fez leituras, **When** ele abre a área "Minha contribuição", **Then** vê o número de leituras do dia, da semana e o total.
2. **Given** um usuário com a função de ponto de transmissão desativada ou sem a permissão de Bluetooth, **When** ele abre essa área, **Then** vê claramente que não está contribuindo e como reativar.
3. **Given** a área "Minha contribuição", **When** ela mostra as leituras, **Then** não exibe a identificação dos Orbits nem dados dos caminhões lidos.

---

### Edge Cases

- **Vários celulares leem o mesmo Orbit ao mesmo tempo** (ex.: caminhões parados num posto): todas as leituras são aceitas; o painel mostra a mais recente como última posição do Orbit, sem duplicar o Orbit no mapa.
- **O celular fica horas sem internet e acumula muitas leituras**: as leituras são enviadas em lotes quando a conexão volta, com o horário real de cada detecção; leituras guardadas há mais tempo que o prazo máximo de retenção no dispositivo são descartadas.
- **O horário do celular está errado**: o servidor guarda também o horário em que recebeu a leitura, e recusa ou marca leituras com horário no futuro.
- **Uma pessoa mal-intencionada emite sinais falsos de Orbit**: na demonstração isso é aceito (é exatamente o que o Orbit virtual faz); para produção, a autenticidade do sinal depende do protocolo real do Orbit (ver Pendências).
- **O celular em modo Orbit virtual também é um ponto de transmissão**: ele não registra leituras do próprio sinal.
- **O celular está sem localização (GPS desligado ou sem sinal)**: a leitura é registrada sem coordenadas precisas apenas se houver uma última localização recente; caso contrário, é descartada, pois uma leitura sem posição não tem utilidade para a KTM.
- **O usuário revoga a permissão de Bluetooth com o aplicativo aberto**: a função de ponto de transmissão para e o aplicativo avisa, sem afetar a rádio.
- **Bateria fraca**: o aplicativo continua funcionando, mas pode reduzir a frequência de busca por Orbits para economizar bateria, informando o usuário.
- **Dois usuários tentam criar canais com o mesmo nome**: é permitido; os canais se diferenciam pelo criador e pelo identificador.
- **O criador apaga um canal com pessoas conversando**: os participantes são avisados e desconectados; nenhum áudio é preservado.

## Requirements *(mandatory)*

### Functional Requirements

**Ponto de transmissão (leitura de Orbit)**

- **FR-001**: O aplicativo DEVE detectar automaticamente sinais de Orbit (reais ou virtuais) que estejam no alcance do celular, sem ação do usuário, enquanto a função de ponto de transmissão estiver ativa.
- **FR-002**: Para cada detecção aceita, o aplicativo DEVE registrar uma leitura contendo: identificação do Orbit, localização do celular no momento da detecção, precisão da localização, horário real da detecção e intensidade do sinal.
- **FR-003**: O aplicativo DEVE limitar as leituras a no máximo uma por Orbit a cada intervalo mínimo (padrão assumido: 30 segundos) por celular, mesmo que o Orbit continue no alcance.
- **FR-004**: O aplicativo DEVE enviar as leituras ao servidor em lotes, nunca uma requisição por leitura (mesmo princípio do FR-033 da 001).
- **FR-005**: O aplicativo DEVE guardar as leituras no dispositivo quando não houver conexão e enviá-las quando a conexão voltar, preservando o horário real da detecção, por até 7 dias.
- **FR-006**: O aplicativo DEVE continuar detectando Orbits com o aplicativo em segundo plano, dentro das limitações de cada sistema operacional, e DEVE informar ao usuário essas limitações quando forem relevantes (ex.: menor frequência de leitura em segundo plano em alguns aparelhos).
- **FR-007**: O aplicativo DEVE emitir um "bip" (som curto e/ou vibração) a cada leitura enviada; o usuário DEVE poder desligar o som e a vibração do bip sem desligar a função de ponto de transmissão.
- **FR-008**: O usuário DEVE poder ativar e desativar a função de ponto de transmissão a qualquer momento.
- **FR-009**: O servidor DEVE armazenar cada leitura com o horário da detecção e o horário de recebimento, e DEVE manter a última posição conhecida de cada Orbit atualizada apenas quando a leitura recebida for mais recente que a última registrada.
- **FR-010**: O servidor DEVE encaminhar as leituras para a KTM por meio de um ponto de integração substituível. Na demonstração, esse encaminhamento DEVE alimentar o painel "Rede Orbit"; quando a KTM fornecer sua própria forma de integração, ela DEVE poder ser adotada sem mudanças no aplicativo.
- **FR-011**: O aplicativo NÃO DEVE mostrar ao usuário comum a identificação dos Orbits lidos nem qualquer dado dos caminhões lidos.

**Orbit virtual (demonstração)**

- **FR-012**: O aplicativo DEVE oferecer um modo Orbit virtual, em que o celular emite um sinal equivalente ao do Orbit com uma identificação de Orbit configurável.
- **FR-013**: O modo Orbit virtual DEVE estar disponível apenas para usuários com permissão de demonstração (administradores da KTM ou contas de demonstração).
- **FR-014**: Enquanto o modo Orbit virtual estiver ativo, o aplicativo DEVE exibir uma indicação permanente e clara disso.
- **FR-015**: Um celular em modo Orbit virtual NÃO DEVE registrar leituras do próprio sinal.
- **FR-016**: A identificação do Orbit e o formato do sinal DEVEM ficar isolados numa definição única, para que possam ser trocados pelos do Orbit real quando a KTM fornecê-los, sem alterar o restante do sistema.

**Painel Rede Orbit**

- **FR-017**: O painel DEVE mostrar num mapa, em tempo real, as leituras recebidas e a última posição conhecida de cada Orbit.
- **FR-018**: O painel DEVE mostrar quantos pontos de transmissão estiveram ativos recentemente (padrão assumido: últimos 15 minutos).
- **FR-019**: O painel DEVE permitir consultar o histórico de leituras de um Orbit num período de datas.
- **FR-020**: O painel DEVE permitir cadastrar pontos fixos de leitura (localização e raio aproximado) e diferenciar visualmente as leituras de pontos fixos das leituras de celulares.
- **FR-021**: O painel DEVE ser acessível apenas a administradores da KTM.
- **FR-022**: O painel NÃO DEVE exibir dados pessoais dos usuários que fizeram as leituras.

**Contas**

- **FR-023**: Qualquer pessoa DEVE poder criar uma conta pelo próprio aplicativo informando nome, apelido de rádio, e-mail e senha, e aceitando os termos de uso e a política de privacidade. Substitui o FR-020 da 001.
- **FR-024**: O sistema DEVE impedir duas contas com o mesmo e-mail.
- **FR-025**: O usuário DEVE poder recuperar o acesso à conta pelo e-mail cadastrado.
- **FR-026**: O usuário DEVE poder excluir a própria conta e seus dados pessoais.
- **FR-027**: O sistema DEVE identificar o autor de cada transmissão de voz pelo apelido de rádio (adapta o FR-021 da 001).
- **FR-028**: O consentimento para atuar como ponto de transmissão DEVE ser pedido explicitamente, separado do aceite dos termos de uso, e o usuário DEVE poder usar a rádio mesmo sem consentir.

**Canais da comunidade**

- **FR-029**: Qualquer usuário autenticado DEVE poder criar canais, escolhendo nome, descrição, limite de participantes (5, 10, 15 ou 20), visibilidade (público ou privado) e, opcionalmente, geocerca. Substitui os FR-009, FR-010, FR-012, FR-013, FR-022 e FR-029 da 001.
- **FR-030**: Canais públicos DEVEM aparecer na busca por nome e, quando tiverem geocerca, na lista de canais próximos à localização do usuário. Canais privados DEVEM ser acessíveis apenas por convite (código ou QR Code).
- **FR-031**: O criador de um canal DEVE poder editar e apagar o canal, gerar um novo convite (invalidando o anterior) e remover participantes.
- **FR-032**: Usuários DEVEM poder marcar canais como favoritos para acessá-los rapidamente.

**Empresas e rastreamento do próprio usuário**

*Decidido com o usuário em 2026-11-21 (opções recomendadas).*

- **FR-033**: O vínculo com empresa DEVE ser opcional. Um usuário PODE estar vinculado a uma transportadora, cujo administrador continua cadastrando motoristas e veículos e acompanhando seus motoristas no painel (como na 001); usuários sem vínculo usam todas as funções do aplicativo normalmente.
- **FR-034**: O sistema DEVE registrar o trajeto (histórico de posições, FR-024 a FR-028 da 001) apenas de usuários vinculados a uma empresa. Para usuários sem vínculo, a localização DEVE ser usada somente na leitura de Orbit, na busca de canais próximos e na geocerca, sem guardar histórico de posições do usuário.

**Experiência de uso**

- **FR-035**: Na primeira abertura, o aplicativo DEVE apresentar uma introdução curta (no máximo 3 telas, que podem ser puladas) explicando a rádio e o ponto de transmissão.
- **FR-036**: Antes de pedir cada permissão do sistema (microfone, localização, Bluetooth, notificações), o aplicativo DEVE explicar em linguagem simples para que ela serve.
- **FR-037**: O aplicativo DEVE funcionar parcialmente sem permissões concedidas, indicando o que está indisponível e oferecendo um atalho para conceder a permissão.
- **FR-038**: O aplicativo DEVE organizar suas funções em no máximo 4 áreas principais, acessíveis com um toque de qualquer tela principal.
- **FR-039**: O aplicativo DEVE respeitar o modo escuro e o tamanho de fonte definidos no sistema do celular.
- **FR-040**: O aplicativo DEVE mostrar ao usuário a contagem das suas próprias leituras (dia, semana e total).

**Mantidos da 001**: voz em canais com fila FIFO de até 10 mensagens de até 90 segundos, áudio nunca persistido, recepção em segundo plano, geocerca com remoção automática, silenciar e sair de canal, limite de participantes, áudio comprimido e disponibilidade em Android e iOS (FR-001 a FR-008, FR-011, FR-014 a FR-019, FR-023, FR-030 a FR-034 da 001).

### Key Entities *(include if feature involves data)*

- **Usuário**: pessoa com conta própria, com nome, apelido de rádio, e-mail, papel (usuário comum, administrador da KTM ou demonstração), consentimento de ponto de transmissão e, opcionalmente, vínculo com uma empresa.
- **Empresa**: transportadora opcional à qual um usuário pode estar vinculado; só usuários vinculados têm trajeto registrado (ver FR-033 e FR-034).
- **Canal**: espaço de voz criado por um usuário, com nome, descrição, limite de participantes, visibilidade (público ou privado), convite (no caso de privado) e geocerca opcional.
- **Convite de canal**: código que dá acesso a um canal privado; pode ser substituído pelo criador.
- **Participação, preferência e favorito de canal**: entrada e saída de usuários nos canais, silenciamento e favoritos.
- **Orbit**: hardware identificado por um código único; na demonstração, pode ser virtual (emitido por um celular). Guarda a última posição conhecida.
- **Leitura Orbit**: detecção de um Orbit por um ponto de transmissão, com identificação do Orbit, localização e precisão do ponto de transmissão, horário da detecção, horário de recebimento, intensidade do sinal e origem (celular ou ponto fixo).
- **Ponto de transmissão**: um celular com o aplicativo e a função ativa; o painel conta os pontos ativos sem identificar a pessoa.
- **Ponto fixo de leitura**: ponto de leitura existente da KTM (simulado na demonstração), com localização e raio aproximado.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Na demonstração, com um celular em modo Orbit virtual a até 10 metros de um ponto de transmissão com o aplicativo aberto, a leitura aparece no painel em até 5 segundos após a aproximação, em pelo menos 9 de cada 10 tentativas.
- **SC-002**: Com o aplicativo em segundo plano em Android, um Orbit que permanece no alcance por pelo menos 1 minuto é lido pelo menos uma vez em 95% dos casos.
- **SC-003**: 100% das leituras feitas sem conexão chegam ao servidor após a reconexão (dentro do prazo de 7 dias), com o horário real da detecção.
- **SC-004**: Uma pessoa que nunca usou o aplicativo cria a conta e chega à lista de canais em menos de 2 minutos.
- **SC-005**: Pelo menos 8 de cada 10 pessoas que nunca usaram o aplicativo conseguem entrar num canal e falar pela primeira vez sem ajuda, em menos de 1 minuto após o cadastro.
- **SC-006**: Um usuário cria um canal privado e um segundo usuário entra nele pelo convite em menos de 1 minuto.
- **SC-007**: A função de ponto de transmissão aumenta o consumo de bateria do aplicativo em no máximo 5 pontos percentuais por hora em relação ao aplicativo com a função desligada.
- **SC-008**: Em nenhuma tela do aplicativo ou do painel um usuário comum consegue ver dados de caminhões lidos, nem a KTM consegue ver dados pessoais de quem fez a leitura.
- **SC-009**: Nenhuma mensagem de voz já reproduzida pode ser encontrada ou reproduzida novamente (SC-003 da 001 mantido).

## Assumptions

- **O Orbit se comunica por Bluetooth de baixo consumo**, emitindo um sinal que celulares próximos conseguem captar sem pareamento. Isso ainda não foi confirmado pela KTM (ver Pendências); o Orbit virtual segue essa suposição.
- **A KTM já tem um sistema próprio** que recebe localizações do Orbit pelos pontos fixos. Este projeto não substitui esse sistema; apenas entrega leituras adicionais a ele. Até a KTM fornecer a forma de integração, as leituras ficam no servidor deste projeto e no painel Rede Orbit.
- **O celular não lê dados internos do Orbit** (sensores, carga etc.), apenas sua identificação. A posição enviada é a do celular, o que é suficiente porque o Orbit está a poucos metros dele.
- **O modo Orbit virtual é usado em celular Android.** Em iPhone, a emissão de sinal com o aplicativo em segundo plano é limitada pelo sistema; na demonstração, o celular que faz o papel de Orbit deve ser Android ou manter o aplicativo aberto em primeiro plano.
- Valores padrão assumidos: intervalo mínimo de 30 segundos entre leituras do mesmo Orbit por celular; leituras guardadas no dispositivo por até 7 dias; pontos de transmissão "ativos" são os que enviaram sinal de vida nos últimos 15 minutos; histórico de leituras guardado no servidor por 90 dias (alinhado à 001).
- A autenticação continua com e-mail e senha; login social (Google/Apple) e verificação por telefone ficam fora desta versão.
- Os termos de uso e a política de privacidade serão textos simples, escritos para o contexto acadêmico e de demonstração; uma revisão jurídica fica fora do escopo.
- A moderação de conteúdo de voz não é possível por desenho (o áudio nunca é guardado); a moderação se limita a o criador remover participantes do canal.
- Mensagens de texto continuam fora do produto.

## Pendências para validar com a KTM

1. **Protocolo do Orbit**: confirmar se o Orbit usa Bluetooth de baixo consumo, qual o formato e a identificação do sinal emitido, a frequência de emissão e o alcance aproximado.
2. **Autenticidade do sinal**: como garantir que um sinal é de um Orbit real e não de um emissor falso (ex.: identificação criptografada ou rotativa). Sem isso, a rede pode receber leituras forjadas em produção.
3. **Integração com o sistema da KTM**: forma de entregar as leituras (API, formato, autenticação) e se a KTM quer receber todas as leituras ou apenas as que trazem informação nova.
4. **Pontos fixos reais**: se a KTM pode fornecer a localização dos pontos fixos atuais para a comparação de cobertura; caso contrário, a demonstração usa pontos simulados.
5. **Dados do caminhão**: se o painel deve associar cada Orbit a uma placa ou transportadora (dados que vêm do sistema da KTM).

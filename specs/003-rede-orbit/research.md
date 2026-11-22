# Pesquisa técnica — Rede Orbit (`003-rede-orbit`)

Cada seção registra **Decisão**, **Justificativa** e **Alternativas consideradas**.
Decisões herdadas da 001 (hub em memória, Opus, Hive, pgx, JWT stateless) não são
repetidas aqui, exceto quando esta feature as altera.

## 1. Formato do sinal do Orbit virtual

**Decisão**: o Orbit virtual anuncia por Bluetooth Low Energy (BLE) um pacote com:

- um **Service UUID de 128 bits fixo do projeto** (o "UUID Rede Orbit"), usado como filtro de busca;
- **Service Data** associado a esse UUID, com 8 bytes: versão do formato (1 byte), tipo (1 byte: `0x01` = Orbit virtual), identificação do Orbit (3 bytes, representada como `ORBIT-XXXXXX` em hexadecimal) e 3 bytes reservados para uma futura identificação rotativa ou assinada. Os bytes ficam na resposta de varredura; o pacote principal leva só o Service UUID, que é o que o filtro de busca precisa (detalhes em [contracts/sinal-orbit.md](./contracts/sinal-orbit.md)).

O formato fica num único arquivo de definição no app (`orbit_sinal.dart`), espelhado no contrato
[contracts/sinal-orbit.md](./contracts/sinal-orbit.md), para que possa ser trocado pelo do Orbit real (FR-016, Princípio VI).

**Justificativa**:
- Buscar com filtro de Service UUID é **obrigatório no iOS** para ler em segundo plano: o sistema só entrega anúncios em segundo plano para buscas filtradas por UUID e com o modo `bluetooth-central` declarado. Ele também continua a busca pelo app depois de encerrado, mas só para os UUIDs filtrados ([Apple Developer Forums](https://developer.apple.com/forums/thread/815286)).
- O Android permite anunciar Service Data. O iOS, como **anunciante**, só permite nome local e Service UUIDs, sem dados de fabricante nem Service Data. Por isso o Orbit virtual roda em Android, como já está assumido na spec. Como **leitor**, o iOS recebe o Service Data normalmente.
- Um UUID de 128 bits mais o Service Data não cabem juntos nos 31 bytes do anúncio legado. Por isso o Service Data vai na resposta de varredura, com um plano B de UUID de 16 bits registrado no contrato.

**Alternativas consideradas**:
- *iBeacon* (UUID + major/minor): o iOS não deixa um app comum **ler** iBeacons pelo CoreBluetooth, só pelo CoreLocation com regiões pré-registradas. Isso complica a leitura e prende o projeto a um formato da Apple.
- *Identificação do Orbit no nome local*: funciona em iOS e Android como anunciante, mas o nome é truncado, vai para a resposta de varredura (scan response) e não chega de forma confiável na busca em segundo plano.
- *Manufacturer Data* (dados de fabricante): exige um Company ID atribuído pelo Bluetooth SIG, que o projeto não tem.
- *Conexão GATT para ler o ID*: cada leitura teria um custo de conexão, com mais bateria e mais latência. É desnecessário para enviar só uma identificação.

## 2. Pacotes Flutter de BLE

**Decisão** (revista na Etapa 3, em 2026-11-21):
- **Leitura (central)**: `flutter_reactive_ble` (licença BSD-3, mantido pela Signify/Philips Hue). Ele filtra por Service UUID (`scanForDevices(withServices: ...)`), entrega `serviceData` e `rssi` em cada `DiscoveredDevice` e, no iOS, busca com `AllowDuplicates`, o que permite reler um Orbit que continua no alcance.
- **Anúncio (periférico, só no Orbit virtual)**: `flutter_ble_peripheral` 3.x (BSD-3). No Android, `AndroidAdvertiseSettings.advertiseResponseData` aceita `serviceDataUuid` + `serviceData`, o que permite pôr o Service Data na resposta de varredura, como pede o contrato. Tem verificação `isSupported` para aparelhos sem suporte a anúncio ([pub.dev](https://pub.dev/packages/flutter_ble_peripheral)).

**Por que não `flutter_blue_plus`**: a versão 2.x passou a usar a "FlutterBluePlus License". Ela exige licença comercial paga para uso por organizações com fins lucrativos e faz uma chamada de telemetria no build (nome do pacote, nome e versão do app). O TCC em si estaria isento como uso educacional, mas a adoção pela KTM não estaria, e a telemetria contraria a postura de privacidade do projeto.

**Alternativas consideradas**: `flutter_blue_plus` (licença acima); `ble_peripheral` / `ble_peripheral_plus` (foco em servidor GATT, mais do que precisamos).

## 3. Leitura em segundo plano

**Decisão**:
- **Android**: a busca roda dentro do *foreground service* que já existe (`background_service.dart`, `flutter_foreground_task`), com o tipo `connectedDevice` acrescentado ao tipo de serviço e as permissões `BLUETOOTH_SCAN` (com `neverForLocation` **desligado**, porque usamos a localização junto) e `BLUETOOTH_CONNECT`. Ciclos de busca de 10 s ligados e 20 s desligados em segundo plano (busca contínua em primeiro plano), no modo `lowPower` em segundo plano.
- **iOS**: acrescentar `bluetooth-central` em `UIBackgroundModes` e `NSBluetoothAlwaysUsageDescription`. Busca sempre filtrada pelo UUID Rede Orbit. A frequência menor em segundo plano é aceita e informada ao usuário (FR-006).

**Justificativa**: reaproveita o ciclo de vida de segundo plano já estabilizado na 001. O ciclo ligado/desligado é o que permite cumprir a meta de bateria (SC-007) sem perder um caminhão que fica parado por um minuto no alcance (SC-002).

**Alternativas consideradas**: `WorkManager`/tarefas periódicas, que no Android têm intervalo mínimo de 15 minutos e não servem para detectar uma passagem.

## 4. Deduplicação e envio de leituras

**Decisão**: o app tem um componente puro `DeduplicadorLeituras` (sem dependência de plataforma, com testes obrigatórios — Princípio IV) que aceita no máximo uma leitura por Orbit a cada 30 s (FR-003) e ignora o próprio Orbit virtual (FR-015). As leituras aceitas entram numa fila Hive separada (`fila_leituras_local`, mesmo padrão de `fila_posicoes_local.dart`), com validade de 7 dias (FR-005). Elas são enviadas em lotes de até 100 a cada 15 s ou quando a conexão volta (FR-004).

No servidor, `POST /leituras` grava o lote em uma única transação e atualiza `orbit.ultima_*` **só se** `detectado_em` for mais recente que a última registrada (FR-009). É o mesmo padrão de atualização condicional de `posicao_atual` e também tem teste obrigatório.

**Justificativa**: espelha um caminho já testado e defendido na 001 (ingestão de posições), o que reduz risco e mantém o discurso do TCC coerente.

## 5. Localização da leitura

**Decisão**: a leitura usa a última posição do `geolocator` com no máximo 60 s de idade. Sem uma posição recente, a leitura é descartada (edge case da spec). Para usuários sem empresa, o `LocalizacaoService` passa a operar no **modo leitura**: mantém a posição em memória para leituras, canais próximos e geocerca, mas **não** enfileira nem envia histórico de posições (FR-034). O histórico continua igual ao da 001 para quem tem empresa.

**Justificativa**: cumpre o FR-034 sem duplicar o serviço de localização. A geocerca já depende de `posicao_atual`, que vira "posição de sessão" para usuários sem empresa (ver data-model §2).

## 6. Integração com a KTM

**Decisão**: interface Go `orbit.Integracao` com um método `Encaminhar(ctx, []Leitura) error`, chamada de forma assíncrona depois da gravação do lote. Implementações:
- `IntegracaoPainel` (padrão): publica no `Broadcaster` da Rede Orbit, que alimenta o painel ao vivo;
- `IntegracaoHTTP` (esqueleto, desativado): envio para uma URL configurável (`KTM_INTEGRACAO_URL`), a ser preenchido quando a KTM fornecer a API.

Uma falha no encaminhamento nunca faz o lote falhar para o app: a leitura já está gravada e pode ser reenviada.

**Justificativa**: FR-010 e Princípio VI. O app nunca fala diretamente com a KTM, então trocar a integração não exige uma nova versão do app.

## 7. Contas abertas e recuperação de senha

**Decisão**:
- `usuario.empresa_id` passa a aceitar nulo. `login` deixa de ser o identificador do público e entra `email` (único, sem diferenciar maiúsculas de minúsculas). Contas antigas continuam entrando com `login`, e `POST /auth/login` aceita `login` **ou** `email` no mesmo campo `identificador`.
- Papéis: `usuario` (público), `motorista` e `admin` (de empresa, como hoje), `admin_ktm` (painel Rede Orbit e modo Orbit virtual) e `demo` (só o modo Orbit virtual).
- O JWT mantém as claims e `empresa_id` vira opcional (vazio quando não há vínculo).
- Recuperação de senha: código de 6 dígitos com validade de 15 minutos, guardado como hash em `recuperacao_senha` e enviado por e-mail via SMTP configurável (`SMTP_*`). Em desenvolvimento e na demo, o `docker compose` sobe o **Mailpit** (caixa de e-mail local com interface web). Sem SMTP configurado, o backend recusa a recuperação com `503`.
- Senha: mínimo de 8 caracteres e bcrypt, como hoje. Cadastro limitado a 5 tentativas por IP a cada 10 minutos (limitador em memória, coerente com a instância única do Princípio VI).

**Alternativas consideradas**: link mágico por e-mail (exige deep link nos dois sistemas, mais trabalho de configuração para pouco ganho); SMS (custo e provedor externo); login social (fora do escopo, ver spec).

## 8. Canais da comunidade

**Decisão**: `canal` ganha `criador_id`, `visibilidade` (`publico`/`privado`) e `codigo_convite` (8 caracteres sem ambiguidade, como `K7QM-2XPA`). `empresa_id` e `tipo_acesso` continuam para os canais de empresa existentes. O acesso para entrar no canal segue uma regra única e testada:

> pode entrar se (canal público) **ou** (é criador) **ou** (tem `acesso_canal` concedido por convite) **ou** (canal de empresa autorizado para a empresa do usuário — regra da 001), **e** não está banido, **e** está dentro da geocerca (se houver), **e** há vaga.

O QR Code codifica `radiopx://convite/<codigo>`. O app já tem `mobile_scanner` para ler, e gerar exige `qr_flutter`.

**Justificativa**: preserva os canais de empresa sem migração destrutiva e concentra a autorização numa função pura testável (Princípio IV ampliado).

## 9. Painel Rede Orbit

**Decisão**: nova página `backend/painel/rede.html` (Leaflet, como o painel atual), com WebSocket `/ws/rede` restrito a `admin_ktm`. Eventos: `leitura` (com pulso animado no mapa), `orbit_atualizado` e `pontos_ativos` (contagem a cada 30 s). Pontos fixos são cadastrados pelo próprio painel e mostrados como círculos de raio fixo. As leituras mostram `origem` (`celular`/`fixo`). Para gerar leituras de pontos fixos simuladas, o painel tem um botão "simular passagem por ponto fixo", que só aparece para `admin_ktm`.

**Privacidade (Princípio VII)**: o painel recebe só o `ponto_id`, um identificador opaco por dispositivo **renovado a cada 24 h**, e nunca `usuario_id`, nome ou e-mail. "Pontos ativos" é contado por `ponto_id` distintos que enviaram sinal de vida nos últimos 15 min.

## 10. Nova experiência (UX/UI)

**Decisão**:
- Navegação com `NavigationBar` (Material 3), 4 abas: **Canais · Explorar · Contribuição · Perfil** (FR-038). A tela do canal ativo é aberta por cima, em tela cheia.
- Rotas com `go_router`, que substitui as rotas nomeadas atuais e suporta deep link do convite (`radiopx://convite/...`).
- Apresentação inicial com 3 telas que podem ser puladas (FR-035), seguida de uma tela de permissões com explicação por item e botão "Permitir" individual (FR-036/037).
- Tela do canal ativo: botão de falar ocupando cerca de 40% da altura, nome do canal e de quem está falando em fonte grande, indicador de fila, e ações de sair e silenciar sempre visíveis (FR-019/US6).
- Tema: evolução do `theme.dart` atual (identidade KTM), com `ColorScheme` claro e escuro e tamanhos de texto que respeitam o tamanho de fonte do sistema (FR-039).
- Antes de codar, as telas principais são prototipadas e aprovadas pelo usuário (Princípio II).

**Alternativas consideradas**: manter a navegação atual por `Navigator.push` (não comporta deep link nem as 4 áreas de forma limpa); `auto_route` (gera código, mais complexo que o necessário).

## 11. Consentimento e LGPD

**Decisão**: tabela `consentimento` (tipo, versão do texto, aceito em, revogado em) para `termos` e `ponto_transmissao`. Os textos de termos e privacidade ficam versionados em `backend/painel/` e são servidos publicamente. Ao excluir a conta (FR-026), o sistema apaga os dados pessoais do usuário e as participações; as leituras feitas por ele são mantidas, porque já são anônimas por `ponto_id` rotativo.

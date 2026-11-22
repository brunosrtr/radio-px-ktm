# Checklist de privacidade: SC-008 e Princípio VII

**Feature**: [spec.md](../spec.md) · **Conferido em**: 2026-11-22 (Etapa 7) · **Base**: código da branch `003-rede-orbit`

**SC-008**: em nenhuma tela do aplicativo ou do painel um usuário comum consegue ver dados de caminhões lidos, nem a KTM consegue ver dados pessoais de quem fez a leitura.

Para cada tela, a coluna **Evidência** aponta o que garante o item: um teste automatizado ou a leitura do código.

## 1. O usuário comum não vê dados dos caminhões lidos

| Tela / rota | O que mostra sobre leituras | Resultado | Evidência |
|---|---|---|---|
| Apresentação (3 telas) | Explica a regra; nenhum dado | ✅ | `features/onboarding/boas_vindas_page.dart` |
| Permissões | Para que serve o Bluetooth; nenhum dado | ✅ | `features/onboarding/permissoes_page.dart` |
| Aba Canais, cartão do ponto de transmissão | Estado e **quantidade** de leituras da sessão | ✅ | `features/ponto/cartao_ponto_transmissao.dart` (usa só `leiturasSessao`, um número) |
| Aba Contribuição | Contadores de hoje, da semana e do total | ✅ | `GET /me/contribuicao` devolve só `hoje`, `semana`, `total`, `ponto_ativo` |
| Bip | Som e vibração, sem texto | ✅ | `orbit/bip.dart` |
| Notificação do foreground service | "Ponto de transmissão ativo" | ✅ | `services/background_service.dart` (`_texto`) |
| Resposta de `POST /leituras` | Só `aceitas` e `duplicadas` | ✅ | `TestContratoRedeOrbit` ("a resposta não deveria trazer dados do Orbit") |
| Perfil, canais, Explorar, convite | Nada da rede Orbit | ✅ | Leitura do código |
| Orbit virtual (só `admin_ktm` e `demo`) | Mostra o ID que **o próprio aparelho** anuncia | ✅ por desenho | Escondido de contas comuns: `PontoTransmissao.podeUsarOrbitVirtual` e teste "conta comum não vê o modo de demonstração" (`widget_test.dart`) |

## 2. A KTM não vê dados pessoais de quem leu

| Tela / rota | O que chega | Resultado | Evidência |
|---|---|---|---|
| Tabela `leitura_orbit` | `ponto_id` opaco, sem coluna nem chave estrangeira para `usuario` | ✅ | `TestLeituraOrbitNaoIdentificaUsuario` |
| `ponto_id` | HMAC(segredo, usuário ‖ dia): muda todo dia e não volta para o usuário sem o segredo | ✅ | `TestPontoIDRotativoEOpaco` |
| WebSocket `/ws/rede` (painel ao vivo) | Leitura com `ponto_id`, posição, RSSI e horário | ✅ | `TestContratoRedeOrbit`: os eventos não contêm id, nome, apelido nem e-mail |
| `GET /rede/orbits` e histórico | Orbits e leituras | ✅ | `TestContratoRedeOrbit` (asserção da Etapa 7 sobre as duas rotas) |
| Integração HTTP com a KTM (`KTM_INTEGRACAO_URL`) | O mesmo `LeituraRegistrada` do painel | ✅ | `orbit/integracao.go` serializa só `leituras` |
| Contadores de contribuição | Em `contribuicao_diaria`, que nenhuma rota do painel lê | ✅ | Leitura do código: só `GET /me/contribuicao` usa a tabela |
| Painel `rede.html` | Mostra "celular" + 6 primeiros caracteres do `ponto_id` | ✅ | `backend/painel/rede.html` |

## 3. Consentimento e controle (FR-008, FR-028)

| Item | Resultado | Evidência |
|---|---|---|
| Consentimento de ponto de transmissão separado dos termos | ✅ | `TestCriarUsuarioPublico` (só `termos` no cadastro); diálogo "Ser um ponto de transmissão" |
| Sem consentimento o servidor recusa leituras e sinal | ✅ | `TestContratoRedeOrbit` (`403 sem_consentimento`) |
| Desligar revoga no servidor e apaga a fila local sem enviar | ✅ | `PontoTransmissao.desativar`; teste "ao revogar, descartarPendentes apaga a fila e parar não envia" |
| Revogado em outro aparelho: o app apaga a fila e desliga | ✅ | Teste "sem consentimento no servidor, a fila é apagada sem enviar" |
| A rádio funciona sem o ponto de transmissão | ✅ | O consentimento só é exigido em `/leituras` e `/ponto/sinal` |

## 4. Guarda e exclusão

| Item | Resultado | Evidência |
|---|---|---|
| Leituras e histórico de posições apagados após 90 dias | ✅ | `TestLimparRespeitaOsPrazos`; `retencao.Rodar` a cada 24 h |
| Fila local de leituras: no máximo 7 dias | ✅ | `FilaLeiturasLocal.descartarAntigas` |
| Contas do público sem histórico de posições | ✅ | `TestPosicoesRecusaUsuarioSemEmpresa` (inclui `PUT /posicao-atual` sem histórico) |
| Excluir a conta apaga dados pessoais, posições e contribuição | ✅ | `TestExcluirConta` + `ExcluirConta` (`contribuicao_diaria`) |
| Áudio nunca persistido (SC-009) | ✅ | Hub da 001 sem mudança; Etapas 1–7 não criaram armazenamento de áudio |

## Riscos que ficam (registrados, não resolvidos nesta etapa)

1. **Rastro de um dia**: dentro do mesmo dia, as leituras de um celular compartilham o `ponto_id`, então a KTM vê o trajeto aproximado daquele aparelho naquele dia, sem saber de quem é. Isso é aceito por desenho: o painel precisa contar pontos distintos. Diminuir a janela (por exemplo, para 1 h) reduz o risco, mas atrapalha a contagem.
2. **Quem acessa o banco direto** poderia cruzar horários de `posicao_atual` (contas do público) com leituras. O painel e a API não permitem isso. A mitigação seria restringir o acesso ao banco em produção.
3. **Logs de acesso HTTP** (`chi` `middleware.Logger`) registram IP e caminho, incluindo o código de convite em `POST /convites/{código}`. Não registram corpo, então leituras e senhas ficam de fora. Em produção, vale reduzir ou rotacionar esses logs.
4. **Fila local** (Hive `fila_leituras`) guarda o ID do Orbit no aparelho até o envio. Não aparece em nenhuma tela; só seria visível com acesso ao armazenamento interno do celular (root).
5. **Painel da empresa (001)**: o administrador da transportadora vê o trajeto dos **próprios** motoristas. Isso é previsto no FR-034 e na política de privacidade; não envolve a rede Orbit.
6. **Termos versão `2026-11`**: o texto foi atualizado na Etapa 7 sem trocar a versão, porque o app ainda não foi publicado. Depois da publicação, qualquer mudança no texto exige nova versão e um novo aceite.

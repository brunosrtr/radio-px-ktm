# Roteiro da demonstração para a KTM — Rede Orbit

Duração: **cerca de 7 minutos** de demonstração mais perguntas. Objetivo: mostrar que **celulares comuns podem fazer o "bip" do Orbit** e ampliar a cobertura dos pontos fixos, sem hardware novo.

**Equipamento**

| Item | Papel | Observação |
|---|---|---|
| Notebook | Backend (Docker) e painel Rede Orbit no projetor | Na mesma rede Wi-Fi dos celulares |
| Celular A, **Android** | "Orbit virtual", o caminhão | Conta `demo@demo.radiopx.local` / `demo-orbit-2026` |
| Celular B (Android ou iPhone) | Caminhoneiro que acabou de instalar o app | Sem conta: a conta é criada na hora |
| Celular C (Android ou iPhone) | Segundo caminhoneiro | Já com conta criada na véspera |
| Painel | Tela da KTM | Conta `admin.ktm@demo.radiopx.local` / `demo-ktm-2026` |

Contas e senhas são valores de teste, só para o ambiente local da demonstração.

---

## 1. Na véspera

- [ ] `git pull` na branch `003-rede-orbit` e `docker compose up --build -d` (Postgres, backend e Mailpit)
- [ ] Descobrir as coordenadas do local da apresentação (no Google Maps, clique com o botão direito no lugar → copiar coordenadas) e semear em volta dele:
  ```bash
  docker compose exec backend /seed -demo -zerar -centro=-23.705,-46.895
  ```
  Os 4 pórticos simulados ficam numa "rodovia" que passa a 15 km e a 20 km do local, de propósito: **o local da apresentação fica entre dois pórticos, sem cobertura fixa**.
- [ ] Instalar o app nos 3 celulares (`cd app && flutter run --release -d <aparelho>`). O Android precisa do Android SDK no notebook.
- [ ] No celular C: criar conta, passar pela apresentação, permitir tudo e **ativar o ponto de transmissão**.
- [ ] No celular A: entrar com a conta `demo`, permitir Bluetooth e localização, abrir Perfil → **Modo Orbit virtual** e testar "Começar a anunciar"/"Parar".
- [ ] Ensaiar o roteiro inteiro **2 vezes seguidas sem falhas** (critério de aceite da Etapa 8). Entre os ensaios, rodar de novo o seed com `-zerar`.
- [ ] Gravar o **vídeo de plano B** (seção 5).
- [ ] Carregar os 3 celulares e o notebook.

## 2. No dia, 30 minutos antes

- [ ] Notebook e celulares no **mesmo Wi-Fi**; se o Wi-Fi do local bloquear a comunicação entre aparelhos, usar o roteador do celular do notebook.
- [ ] `docker compose up -d` e `docker compose exec backend /seed -demo -zerar -centro=<lat,lng do local>`
- [ ] Abrir `http://<ip-do-notebook>:8080/painel/rede.html` no projetor e entrar com a conta da KTM. Conferir "ao vivo" no canto.
- [ ] Celulares com **Bluetooth e localização ligados**, modo "não perturbe" desligado (para o bip tocar) e volume alto.
- [ ] Celular B **sem** o app aberto, com o app já instalado.
- [ ] Deixar um terminal pronto com o plano B ao vivo (seção 4).

## 3. Roteiro

| # | Tempo | O que fazer | O que falar | Se falhar |
|---|---|---|---|---|
| 1 | 0:00 | Painel no projetor. Clique no **Caminhão demo 1**: aparece visto pela última vez há 3 h num pórtico. Use **"Simular passagem por ponto fixo"** com o pórtico D e o caminhão 2. | "Hoje o Orbit só manda a posição quando passa num pórtico. Entre um pórtico e outro, a KTM fica sem saber onde o caminhão está." | Painel desconectado: recarregar a página; o token fica salvo. |
| 2 | 1:00 | Celular B: abrir o app, mostrar as **3 telas da apresentação** e **criar a conta** na hora. | "Qualquer caminhoneiro baixa e entra em menos de um minuto." | Sem internet no B: usar o celular C, que já tem conta. |
| 3 | 2:00 | Celular B: tela de permissões e o cartão âmbar **"Ajude a rede Orbit" → Ativar**. Ler em voz alta o diálogo de privacidade. | "O celular não vê nada do caminhão, e a KTM não sabe quem leu." | Recusou permissão sem querer: botão "Tentar de novo" no cartão. |
| 4 | 3:00 | Celular A: **Começar a anunciar** `ORBIT-0A1B2C`. Aproximar B de A (até 2 m). | "Este celular está fazendo o papel do Orbit instalado no caminhão." | Bluetooth falhou: **plano B ao vivo** (seção 4). |
| 5 | 3:30 | B **bipa e vibra**; o painel pulsa e o Caminhão demo 1 "pula" do pórtico para o local da apresentação. | "Esse bip é o caminhão sendo localizado entre dois pórticos, onde antes não havia cobertura." | Sem bip mas com a leitura no painel: o volume estava baixo; seguir. |
| 6 | 4:30 | Afastar B; aproximar **C** (com a tela **bloqueada**) de A. O contador **"pontos de transmissão ativos"** sobe para 2. | "Quanto mais gente com o app, mais pontos de leitura. E funciona com a tela apagada." | iPhone com a tela apagada demora: desbloquear e seguir ("no iPhone o sistema limita a frequência"). |
| 7 | 5:30 | Celular B em **modo avião**, aproximar de A, tirar do modo avião. A leitura chega com o horário real. | "Na estrada sem sinal, a leitura fica guardada e sai quando o sinal volta." | Passo opcional: pular se o tempo estiver curto. |
| 8 | 6:00 | B e C entram no canal **"Rádio KTM · Demo"** (aba Explorar → Todos os públicos) e conversam segurando o botão. | "É isso que faz o caminhoneiro manter o app aberto: o rádio PX dele, no celular." | Sem áudio: tocar na tela do canal para liberar o som (aviso no topo). |
| 9 | 6:40 | Aba **Contribuição** no B: contadores. No painel, clicar no caminhão e mostrar o histórico. | "Para o caminhoneiro, só números. Para a KTM, só o caminhão, nunca a pessoa." | — |

**Encerramento**: "Com o protocolo do Orbit real, trocamos um arquivo do app (`orbit_sinal.dart`) e a integração com o sistema da KTM (`orbit.Integracao`). O resto já está pronto." Ver as pendências na [spec.md](./spec.md#pendências-para-validar-com-a-ktm).

## 4. Plano B ao vivo (sem Bluetooth)

Se o Bluetooth falhar no palco, o simulador envia pela API as mesmas leituras que os celulares enviariam: o caminhão 2 "anda" pela rodovia simulada, lido por dois pontos de transmissão:

```bash
docker compose exec backend /simular -orbit ORBIT-0A1B2D -centro=<lat,lng do local> -passos 30 -intervalo 3s
```

O painel mostra os pulsos, o caminhão se movendo e o contador de pontos subindo. Fala: "aqui o Bluetooth do local não colaborou; estas são as mesmas leituras, enviadas pela API".

## 5. Vídeo de plano B

Gravar na véspera, com o roteiro dando certo:

1. Gravação de tela do notebook (macOS: `Cmd+Shift+5` → gravar tela inteira) mostrando o painel.
2. Outro celular filmando as mãos aproximando B de A, com o som do bip.
3. Juntar os dois lado a lado (iMovie ou CapCut) com até 2 minutos: passos 1, 4–6 e 9 do roteiro.
4. Deixar o vídeo **baixado** no notebook (sem depender da internet do local).

## 6. Medição de bateria (SC-007)

Meta: o ponto de transmissão aumenta o consumo do app em **no máximo 5 pontos percentuais por hora**.

1. Celular B carregado a 100%, brilho fixo em 50%, Wi-Fi ligado, sem outros apps abertos.
2. **Rodada 1 (sem ponto)**: app aberto na aba Canais com o ponto de transmissão **desligado**, tela apagada, por 60 min. Anotar a bateria no fim.
3. Recarregar a 100%. **Rodada 2 (com ponto)**: mesmo cenário com o ponto **ligado** e o celular A anunciando a até 5 m durante toda a hora.
4. Resultado = (consumo da rodada 2) − (consumo da rodada 1).

| Aparelho | Sistema | Rodada 1 (p.p./h) | Rodada 2 (p.p./h) | Diferença | ≤ 5? |
|---|---|---|---|---|---|
| | | | | | |

## 7. Teste com 3 pessoas que nunca usaram o app (SC-004/SC-005)

Para cada pessoa, com o app recém-instalado, sem ajuda e com cronômetro:

1. "Crie uma conta." → tempo até chegar à lista de canais (meta: **menos de 2 min**).
2. "Entre no canal 'Rádio KTM · Demo' e diga 'teste'." → tempo desde o fim do cadastro (meta: **menos de 1 min**; 8 de cada 10 pessoas sem ajuda).
3. Anotar onde a pessoa hesitou ou errou.

| Pessoa | Cadastro (min:s) | Primeira fala (min:s) | Precisou de ajuda? | Onde hesitou |
|---|---|---|---|---|
| 1 | | | | |
| 2 | | | | |
| 3 | | | | |

## 8. Depois da demonstração

- `docker compose exec backend /seed -demo -zerar` para limpar as leituras de teste.
- Levar para a reunião as pendências da spec: **protocolo real do Orbit, autenticidade do sinal, API de integração da KTM e localização dos pórticos reais**.

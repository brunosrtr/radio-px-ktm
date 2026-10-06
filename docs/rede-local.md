# Conexão local sem IP fixo

O computador executa a API, o banco e o painel. Os celulares descobrem o serviço
`_radiopx._tcp` por mDNS/DNS-SD. Não é necessário internet, DNS público ou
recompilar o aplicativo quando o IP muda.

## Iniciar no Linux com Docker

Use Docker Compose com suporte à tag `!reset` (2.24.4 ou posterior):

```bash
docker compose -f docker-compose.yml -f docker-compose.lan.yml up --build -d
```

Abra `http://localhost:<BACKEND_PORT>/painel/conectar.html` no computador
(`BACKEND_PORT` é o valor do seu `.env`). A página também pode ser aberta pelo
link **Conectar celulares** no painel, inclusive antes do login.

O backend usa a rede do host para o multicast alcançar o Wi-Fi. O PostgreSQL
continua em seu container e o backend o acessa pela porta publicada no host.
O comando explícito não carrega `docker-compose.override.yml`.
As credenciais interpoladas na URL precisam estar codificadas para URL se
contiverem caracteres reservados.

Para rodar o backend diretamente no computador, mantenha as variáveis normais
`DATABASE_URL`, `JWT_SECRET` e `PORT`, e acrescente `LOCAL_DISCOVERY=true`.
A descoberta é optativa; sem essa variável o backend mantém seu comportamento
anterior. Windows/macOS: prefira executar a API diretamente no host; o arquivo
Compose local foi preparado para Linux.

## Celulares

```bash
cd app
flutter pub get
flutter run
```

Não passe `API_BASE_URL` ou `WS_BASE_URL` para usar descoberta automática.
Conecte os aparelhos ao mesmo Wi-Fi e autorize rede local/câmera quando solicitado.
Na tela de login, o aplicativo procura o computador. Se encontrar apenas um,
seleciona-o; se encontrar vários, permite escolher. O identificador curto aparece
também na página de conexão do computador.

Se a descoberta não funcionar, toque em **Ler QR Code** e leia um código do painel.
O leitor e a geração do QR Code funcionam offline. O código contém somente o
endereço e a identidade da instalação, sem senha ou token. O login continua
obrigatório. O QR não atravessa bloqueios de comunicação da rede.

Para continuar usando USB ou endereço explícito, os `--dart-define` anteriores
seguem disponíveis e têm prioridade sobre a descoberta.

## Troca de rede e identidade

O aplicativo guarda a identidade do servidor e seu último endereço no Hive.
Antes de novas conexões HTTP/WebSocket, verifica o endereço e, se necessário,
procura novamente o mesmo computador. A verificação é compartilhada entre
requisições simultâneas e tem cache curto de 10 segundos. Não reenvia requisições
de escrita automaticamente após erro, evitando duplicar operações.

O backend reavalia suas interfaces a cada cinco segundos e renova o anúncio
quando os endereços mudam. O painel atualiza os códigos a cada dez segundos;
também há um botão de atualização. Se a página foi aberta pelo IP antigo,
reabra-a usando localhost no computador.

A identidade fica no volume `local_identity` no Docker ou no arquivo
`.radio-px-server-id` quando a API roda diretamente. Preserve esse arquivo/volume
entre reinícios. Apagá-lo cria outra identidade e exige selecionar/parear novamente.
O anúncio não autentica o servidor: este modo HTTP é para uma rede local confiável.

## Rede e limites

- Permita TCP na porta do backend e multicast UDP 5353 no firewall local.
- Wi-Fi de visitantes pode bloquear comunicação entre aparelhos.
- A implementação de descoberta anuncia IPv4; redes exclusivamente IPv6 não
  estão cobertas neste modo.
- A página de pareamento não usa CDN. O mapa do painel continua dependendo dos
  recursos externos já existentes; esta alteração não disponibiliza mapas offline.
- iOS precisa de validação em dispositivo físico, incluindo permissões e ATS.

## Verificação manual com dois celulares

1. Inicie o backend, abra a página de conexão e conecte dois celulares ao Wi-Fi.
2. Abra o app sem URLs de compilação e entre com contas diferentes.
3. Confirme voz entre os aparelhos no mesmo canal.
4. Mude computador e celulares para outra rede e confirme a redescoberta.
5. Bloqueie apenas multicast e teste o QR Code (TCP do backend deve continuar liberado).
6. Reinicie backend/app e confirme que a identidade foi preservada.
7. Com dois servidores anunciando, confirme a seleção e que o app não troca
   silenciosamente de instalação quando o computador salvo fica indisponível.

Durante uma queda de rede, o canal interrompe a fala e tenta reconectar a cada
cinco segundos. O heartbeat WebSocket dos celulares detecta conexões interrompidas.
A transmissão interrompida não é retomada: solte e pressione o botão novamente.

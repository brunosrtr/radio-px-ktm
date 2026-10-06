# Rádio PX — app Flutter

## Teste Android por USB

Com a depuração USB autorizada no celular e o backend em execução, rode a partir desta pasta:

```bash
flutter doctor
adb devices
flutter pub get
dart analyze lib test
flutter test
adb reverse tcp:8081 tcp:8081
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8081 --dart-define=WS_BASE_URL=ws://127.0.0.1:8081
```

A porta 8081 é a porta do backend neste ambiente. Ajuste os comandos se a porta publicada pelo Docker mudar. O encaminhamento `adb reverse` precisa ser refeito após reconectar o aparelho. Autorize microfone, localização e notificações quando o app solicitar.

## APK de teste

```bash
flutter build apk --debug --dart-define=API_BASE_URL=http://127.0.0.1:8081 --dart-define=WS_BASE_URL=ws://127.0.0.1:8081
adb install -r build/app/outputs/flutter-apk/app-debug.apk
```

Esse APK depende do computador e do encaminhamento USB para acessar o servidor. Para teste sem USB, configure URLs acessíveis pelo celular. O app permite HTTP para uso em rede local confiável; para distribuição fora dessa rede, use HTTPS/WSS e assinatura própria.

## Áudio em tempo real

O microfone e o player usam PCM16 mono a 16 kHz em memória. `opus_dart` e
`opus_flutter_new` convertem o áudio para pacotes Opus de 20 ms, com teto de
60 bytes (24 kbit/s), antes do WebSocket. Nenhum áudio é salvo em arquivo.
O backend repassa os pacotes sem alteração; os dois celulares devem estar
com a versão atualizada do app e contas diferentes no mesmo canal.

Ao testar sozinho, você não ouvirá sua própria transmissão: o servidor
não devolve áudio ao remetente. Segure o botão para falar e solte para parar.

Verificações: `dart analyze lib test integration_test` e `flutter test`. Os testes do codec em Linux
usam a biblioteca de sistema `libopus.so.0`.

## Teste de integração no Android

Com o backend de desenvolvimento, as contas de teste e um canal sem geocerca:

```bash
adb reverse tcp:8081 tcp:8081
flutter test integration_test/radio_audio_test.dart -d <id-do-celular> --dart-define=API_BASE_URL=http://127.0.0.1:8081 --dart-define=WS_BASE_URL=ws://127.0.0.1:8081
```

Autorize as permissões solicitadas na tela. O teste aciona o microfone em duas
falas curtas e toca um tom de retorno: valida captura, Opus, WebSocket, player
e liberação do botão entre falas. Usa somente memória, sem salvar áudio.
Depois, gere e instale o APK normal pelos comandos acima (o APK do teste
executa a automação ao abrir).

O Gradle mantém os intermediários nativos do Opus no cache Gradle, isolados
por projeto, pois `ndk-build` não aceita espaços no caminho de saída. Também
alinha o SDK de compilação desse plugin ao SDK 36 usado pelo app.

## Conexão automática pelo Wi-Fi

Execute o app sem `--dart-define` para procurar o computador automaticamente.
O botão **Ler QR Code** está disponível na tela de login. Consulte o
[guia de rede local](../docs/rede-local.md) para iniciar o backend e testar a troca de rede.

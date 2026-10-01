import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/services/ptt_controller.dart';

void main() {
  late ControladorPtt ptt;
  late StreamController<Uint8List> audio;
  late List<String> eventos;
  Completer<Stream<Uint8List>>? abertura;

  setUp(() {
    eventos = [];
    abertura = null;
    audio = StreamController<Uint8List>();
    ptt = ControladorPtt(
      solicitarSlot: () => eventos.add('solicitar'),
      finalizarTransmissao: (id) => eventos.add('finalizar:$id'),
      cancelarTransmissao: (id) => eventos.add('cancelar:$id'),
      iniciarGravacao: () async {
        eventos.add('abrir');
        return abertura == null ? audio.stream : await abertura!.future;
      },
      pararGravacao: () async => eventos.add('parar'),
      enviarAudio: (_) => eventos.add('audio'),
      aoMudar: () {},
      aoFalhar: (_) => eventos.add('erro'),
    );
  });

  tearDown(() async {
    await ptt.fechar();
    // close de stream sem ouvinte não deve bloquear a limpeza do teste.
    unawaited(audio.close());
  });

  test('soltar antes do slot cancela sem abrir o microfone', () async {
    ptt.pressionar();
    await ptt.soltar();
    ptt.pressionar(); // não cria outra solicitação enquanto aguarda resposta
    await ptt.conceder('a', const Duration(seconds: 90));
    expect(eventos, ['solicitar', 'cancelar:a']);
    expect(ptt.ocupado, isFalse);
  });

  test('soltar durante abertura para o microfone depois que ele abre', () async {
    abertura = Completer<Stream<Uint8List>>();
    ptt.pressionar();
    final inicio = ptt.conceder('a', const Duration(seconds: 90));
    final parada = ptt.soltar();
    abertura!.complete(audio.stream);
    await inicio;
    await parada;
    expect(eventos, ['solicitar', 'abrir', 'parar', 'finalizar:a']);
    expect(ptt.ocupado, isFalse);
  });

  test('corte do servidor para captura e ignora chunks posteriores', () async {
    ptt.pressionar();
    await ptt.conceder('a', const Duration(seconds: 90));
    audio.add(Uint8List.fromList([1]));
    await Future<void>.delayed(Duration.zero);
    await ptt.servidorEncerrou('a');
    audio.add(Uint8List.fromList([2]));
    await Future<void>.delayed(Duration.zero);
    expect(eventos.where((e) => e == 'audio'), hasLength(1));
    expect(eventos, contains('parar'));
    expect(eventos, isNot(contains('finalizar:a')));
    expect(ptt.ocupado, isFalse);
  });

  test('limite local encerra captura sem depender do servidor', () async {
    ptt.pressionar();
    await ptt.conceder('a', const Duration(milliseconds: 5));
    await Future<void>.delayed(const Duration(milliseconds: 30));
    expect(eventos, containsAllInOrder(['parar', 'finalizar:a']));
    expect(ptt.ocupado, isFalse);
  });

  test('desconexão durante abertura não deixa microfone ativo', () async {
    abertura = Completer<Stream<Uint8List>>();
    ptt.pressionar();
    final inicio = ptt.conceder('a', const Duration(seconds: 90));
    final parada = ptt.interromper();
    abertura!.complete(audio.stream);
    await inicio;
    await parada;
    expect(eventos, ['solicitar', 'abrir', 'parar']);
    expect(ptt.ocupado, isFalse);
  });

  test('slot negado permite nova tentativa', () {
    ptt.pressionar();
    ptt.negar();
    ptt.pressionar();
    expect(eventos, ['solicitar', 'solicitar']);
  });

  test('falha ao abrir o microfone cancela o slot e libera nova tentativa', () async {
    abertura = Completer<Stream<Uint8List>>();
    ptt.pressionar();
    final inicio = ptt.conceder('a', const Duration(seconds: 90));
    abertura!.completeError(StateError('permissão negada'));
    await inicio;
    await Future<void>.delayed(Duration.zero);
    expect(eventos, ['solicitar', 'abrir', 'erro', 'parar', 'cancelar:a']);
    expect(ptt.ocupado, isFalse);
  });

  test('encerramento de outra transmissão não interrompe a captura atual', () async {
    ptt.pressionar();
    await ptt.conceder('atual', const Duration(seconds: 90));
    await ptt.servidorEncerrou('anterior');
    expect(ptt.gravando, isTrue);
    expect(eventos, isNot(contains('parar')));
  });

}

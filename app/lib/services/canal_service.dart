import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../core/env.dart';
import '../core/token_storage.dart';

/// Um evento recebido do canal: de controle (JSON) ou um chunk de áudio
/// (frame binário) — nunca os dois ao mesmo tempo, espelhando `canal.Evento`
/// no backend.
class EventoCanal {
  EventoCanal.controle(this.tipo, this.dados) : audio = null;

  EventoCanal.audio(Uint8List bytes)
    : tipo = null,
      dados = null,
      audio = bytes;

  final String? tipo;
  final Map<String, dynamic>? dados;
  final Uint8List? audio;

  bool get isAudio => audio != null;
}

/// Conexão WebSocket com o canal ativo — eventos de controle e frames
/// binários de áudio na mesma conexão (contracts/websocket-protocol.md).
class CanalService {
  CanalService({TokenStorage? tokenStorage})
    : _tokenStorage = tokenStorage ?? TokenStorage.instance;

  final TokenStorage _tokenStorage;

  WebSocketChannel? _canal;
  StreamController<EventoCanal>? _eventos;

  Stream<EventoCanal> get eventos => _eventos!.stream;

  Future<void> conectar() async {
    final token = _tokenStorage.token;
    final uri = Uri.parse(
      '${Env.wsBaseUrl}/ws',
    ).replace(queryParameters: {'token': ?token});

    _eventos = StreamController<EventoCanal>.broadcast();
    _canal = WebSocketChannel.connect(uri);

    _canal!.stream.listen(
      _processarMensagemRecebida,
      onDone: () => _eventos?.close(),
      onError: (Object erro) => _eventos?.addError(erro),
    );

    // Sem isso, um `entrarCanal` chamado logo em seguida pode ser enviado
    // antes do handshake terminar — no navegador isso derruba a mensagem
    // silenciosamente (WebSocket ainda em CONNECTING), então o servidor
    // nunca registra a entrada e qualquer ação seguinte (ex.: solicitar
    // slot) volta com `erro: sem_permissao`.
    await _canal!.ready;
  }

  void _processarMensagemRecebida(dynamic mensagem) {
    if (mensagem is List<int>) {
      _eventos?.add(EventoCanal.audio(Uint8List.fromList(mensagem)));
      return;
    }

    final decodificado = jsonDecode(mensagem as String) as Map<String, dynamic>;
    _eventos?.add(
      EventoCanal.controle(
        decodificado['tipo'] as String,
        (decodificado['dados'] as Map<String, dynamic>?) ?? const {},
      ),
    );
  }

  void entrarCanal(String canalId) =>
      _enviarControle('entrar_canal', {'canal_id': canalId});

  void sairCanal(String canalId) =>
      _enviarControle('sair_canal', {'canal_id': canalId});

  void solicitarSlot(String canalId) =>
      _enviarControle('solicitar_slot', {'canal_id': canalId});

  void finalizarTransmissao(String transmissaoId) => _enviarControle(
    'finalizar_transmissao',
    {'transmissao_id': transmissaoId},
  );

  void ping() => _enviarControle('ping', const {});

  void enviarChunkAudio(Uint8List chunk) => _canal?.sink.add(chunk);

  void _enviarControle(String tipo, Map<String, dynamic> dados) {
    _canal?.sink.add(jsonEncode({'tipo': tipo, 'dados': dados}));
  }

  Future<void> desconectar() async {
    await _canal?.sink.close();
    await _eventos?.close();
    _canal = null;
    _eventos = null;
  }
}

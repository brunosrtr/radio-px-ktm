import 'dart:async';
import 'dart:typed_data';

enum EstadoPtt { ocioso, aguardandoSlot, iniciando, transmitindo, encerrando }

/// Coordena a intenção de falar com a autorização remota e o microfone.
/// Não armazena áudio: apenas encaminha chunks da captura ativa.
class ControladorPtt {
  ControladorPtt({
    required this.solicitarSlot,
    required this.finalizarTransmissao,
    required this.cancelarTransmissao,
    required this.iniciarGravacao,
    required this.pararGravacao,
    required this.enviarAudio,
    required this.aoMudar,
    required this.aoFalhar,
  });

  final void Function() solicitarSlot;
  final void Function(String) finalizarTransmissao;
  final void Function(String) cancelarTransmissao;
  final Future<Stream<Uint8List>> Function() iniciarGravacao;
  final Future<void> Function() pararGravacao;
  final void Function(Uint8List) enviarAudio;
  final void Function() aoMudar;
  final void Function(Object) aoFalhar;

  EstadoPtt _estado = EstadoPtt.ocioso;
  EstadoPtt get estado => _estado;
  bool get ocupado => _estado != EstadoPtt.ocioso;
  bool get gravando => _estado == EstadoPtt.transmitindo;
  bool get aguardando => _estado == EstadoPtt.aguardandoSlot ||
      _estado == EstadoPtt.iniciando || _estado == EstadoPtt.encerrando;

  bool _pressionado = false;
  bool _fechado = false;
  String? _transmissaoId;
  Timer? _limite;
  StreamSubscription<Uint8List>? _captura;
  Future<void>? _inicio;
  Future<void>? _parada;

  void _mudar(EstadoPtt estado) {
    _estado = estado;
    if (!_fechado) aoMudar();
  }

  void pressionar() {
    if (_fechado || ocupado) return;
    _pressionado = true;
    _mudar(EstadoPtt.aguardandoSlot);
    try {
      solicitarSlot();
    } catch (erro) {
      negar();
      aoFalhar(erro);
    }
  }

  Future<void> conceder(String id, Duration duracaoMaxima) {
    if (_fechado) return Future.value();
    if (_transmissaoId == id) return _inicio ?? Future.value();
    if (_estado != EstadoPtt.aguardandoSlot) {
      cancelarTransmissao(id);
      return Future.value();
    }
    if (!_pressionado || duracaoMaxima <= Duration.zero) {
      cancelarTransmissao(id);
      _pressionado = false;
      _mudar(EstadoPtt.ocioso);
      return Future.value();
    }
    _transmissaoId = id;
    _mudar(EstadoPtt.iniciando);
    _limite = Timer(duracaoMaxima, () => unawaited(soltar()));
    return _inicio = _iniciar(id);
  }

  Future<void> _iniciar(String id) async {
    try {
      final stream = await iniciarGravacao();
      _captura = stream.listen(
        (chunk) {
          if (!_fechado && _pressionado && _transmissaoId == id && gravando) {
            try {
              enviarAudio(chunk);
            } catch (erro) {
              aoFalhar(erro);
              unawaited(_encerrar(cancelar: true));
            }
          }
        },
        onError: (Object erro) {
          if (!_fechado) aoFalhar(erro);
          unawaited(_encerrar(cancelar: true));
        },
        onDone: () => unawaited(soltar()),
      );
      if (!_fechado && _pressionado && _transmissaoId == id) {
        _mudar(EstadoPtt.transmitindo);
      }
    } catch (erro) {
      if (!_fechado) aoFalhar(erro);
      unawaited(_encerrar(cancelar: true));
    }
  }

  Future<void> soltar() {
    _pressionado = false;
    // Se a autorização ainda não chegou, conservamos a solicitação pendente.
    // conceder irá cancelá-la sem abrir o microfone e sem solicitar outra.
    return _encerrar();
  }

  void negar() {
    if (_estado != EstadoPtt.aguardandoSlot) return;
    _pressionado = false;
    _mudar(EstadoPtt.ocioso);
  }

  Future<void> servidorEncerrou(String id) {
    if (_transmissaoId != id) return Future.value();
    return _encerrar(informarServidor: false);
  }

  /// Queda de conexão ou remoção: descarta a captura e a intenção de falar.
  Future<void> interromper() {
    _pressionado = false;
    if (_transmissaoId == null && _parada == null) {
      _mudar(EstadoPtt.ocioso);
      return Future.value();
    }
    return _encerrar(cancelar: true, informarServidor: false);
  }

  Future<void> _encerrar({bool cancelar = false, bool informarServidor = true}) {
    _pressionado = false;
    if (_parada != null) return _parada!;
    final id = _transmissaoId;
    if (id == null) return Future.value();
    _transmissaoId = null;
    _limite?.cancel();
    _mudar(EstadoPtt.encerrando);
    return _parada = _parar(id, cancelar, informarServidor)
        .whenComplete(() => _parada = null);
  }

  Future<void> _parar(String id, bool cancelar, bool informarServidor) async {
    // Uma permissão/inicialização nativa pode terminar depois de soltar o
    // botão. A parada deve acontecer depois dela, nunca em paralelo.
    await _inicio;
    try {
      await pararGravacao();
    } catch (erro) {
      cancelar = true;
      if (!_fechado) aoFalhar(erro);
    } finally {
      await _captura?.cancel();
      _captura = null;
      _inicio = null;
      if (informarServidor && !_fechado) {
        if (cancelar) {
          cancelarTransmissao(id);
        } else {
          finalizarTransmissao(id);
        }
      }
      _mudar(EstadoPtt.ocioso);
    }
  }

  Future<void> fechar() {
    _fechado = true;
    _limite?.cancel();
    return interromper();
  }
}

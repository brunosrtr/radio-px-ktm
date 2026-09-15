import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';

import '../../services/audio_service.dart';
import '../../services/background_service.dart';
import '../../services/canal_service.dart';
import '../../services/localizacao_service.dart';

/// Tela do canal ativo: botão push-to-talk, bloqueio quando a fila está
/// cheia e exibição do nome de quem está falando (FR-04 a FR-09, FR-21).
class CanalAtivoPage extends StatefulWidget {
  const CanalAtivoPage({
    super.key,
    required this.canalId,
    required this.nomeCanal,
  });

  final String canalId;
  final String nomeCanal;

  @override
  State<CanalAtivoPage> createState() => _CanalAtivoPageState();
}

class _CanalAtivoPageState extends State<CanalAtivoPage> {
  final _canalService = CanalService();
  final _audioService = AudioService();
  final _localizacaoService = LocalizacaoService();

  StreamSubscription<EventoCanal>? _assinaturaEventos;
  StreamSubscription<Uint8List>? _assinaturaGravacao;

  bool _filaCheia = false;
  bool _gravando = false;
  bool _aguardandoSlot = false;
  String? _transmissaoAtivaId;
  String? _remetenteFalando;
  String? _avisoRemocao;

  @override
  void initState() {
    super.initState();
    _iniciar();
  }

  Future<void> _iniciar() async {
    await _audioService.abrir();
    await _audioService.iniciarReproducao();

    await _canalService.conectar();
    _assinaturaEventos = _canalService.eventos.listen(_processarEvento);
    _canalService.entrarCanal(widget.canalId);

    await BackgroundService.iniciar();
    await _localizacaoService.iniciar();
  }

  void _processarEvento(EventoCanal evento) {
    if (evento.isAudio) {
      _audioService.reproduzirChunk(evento.audio!);
      return;
    }

    switch (evento.tipo) {
      case 'canal_entrado':
      case 'estado_canal':
        final tamanhoFila = evento.dados?['tamanho_fila'] as int?;
        if (tamanhoFila != null) {
          setState(() => _filaCheia = tamanhoFila >= 10);
        }
        break;

      case 'slot_concedido':
        final transmissaoId = evento.dados?['transmissao_id'] as String;
        setState(() {
          _transmissaoAtivaId = transmissaoId;
          _gravando = true;
          _aguardandoSlot = false;
        });
        unawaited(_iniciarStreamDeGravacao());
        break;

      case 'slot_negado':
        setState(() {
          _aguardandoSlot = false;
          _filaCheia = evento.dados?['motivo'] == 'fila_cheia';
        });
        break;

      case 'inicio_reproducao':
        setState(
          () => _remetenteFalando = evento.dados?['remetente_nome'] as String?,
        );
        break;

      case 'fim_reproducao':
        setState(() => _remetenteFalando = null);
        break;

      case 'removido_canal':
        setState(
          () => _avisoRemocao = _mensagemRemocao(evento.dados?['motivo'] as String?),
        );
        break;
    }
  }

  String _mensagemRemocao(String? motivo) {
    switch (motivo) {
      case 'geocerca':
        return 'Você foi removido do canal por sair da área definida.';
      case 'limite':
        return 'Você foi removido do canal por limite de participantes.';
      case 'canal_desativado':
        return 'Este canal foi desativado.';
      default:
        return 'Você foi removido do canal.';
    }
  }

  Future<void> _iniciarStreamDeGravacao() async {
    final stream = await _audioService.iniciarGravacao();
    _assinaturaGravacao = stream.listen(_canalService.enviarChunkAudio);
  }

  Future<void> _iniciarFala() async {
    if (_filaCheia || _gravando || _aguardandoSlot) return;
    setState(() => _aguardandoSlot = true);
    _canalService.solicitarSlot(widget.canalId);
  }

  Future<void> _pararFala() async {
    if (!_gravando) return;

    await _assinaturaGravacao?.cancel();
    await _audioService.pararGravacao();

    final transmissaoId = _transmissaoAtivaId;
    setState(() {
      _gravando = false;
      _transmissaoAtivaId = null;
    });

    if (transmissaoId != null) {
      _canalService.finalizarTransmissao(transmissaoId);
    }
  }

  @override
  void dispose() {
    _assinaturaEventos?.cancel();
    _assinaturaGravacao?.cancel();
    _canalService.sairCanal(widget.canalId);
    unawaited(_canalService.desconectar());
    unawaited(_audioService.fechar());
    unawaited(_localizacaoService.parar());
    unawaited(BackgroundService.parar());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final podeFalar = !_filaCheia && !_gravando && !_aguardandoSlot;

    return Scaffold(
      appBar: AppBar(title: Text(widget.nomeCanal)),
      body: Column(
        children: [
          if (_avisoRemocao != null)
            MaterialBanner(
              content: Text(_avisoRemocao!),
              actions: [
                TextButton(
                  onPressed: () => setState(() => _avisoRemocao = null),
                  child: const Text('OK'),
                ),
              ],
            ),
          Expanded(
            child: Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (_filaCheia)
                    const Text(
                      'Fila cheia — aguarde para falar',
                      style: TextStyle(color: Colors.red),
                    ),
                  if (_remetenteFalando != null)
                    Text('Falando agora: $_remetenteFalando'),
                ],
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(24),
            child: GestureDetector(
              onTapDown: (_) => _iniciarFala(),
              onTapUp: (_) => _pararFala(),
              onTapCancel: _pararFala,
              child: CircleAvatar(
                radius: 48,
                backgroundColor: !podeFalar && !_gravando
                    ? Colors.grey
                    : (_gravando ? Colors.red : Theme.of(context).colorScheme.primary),
                child: const Icon(Icons.mic, color: Colors.white, size: 40),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';

import '../../core/theme.dart';
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

  bool _conectado = false;
  bool _audioDestravado = false;
  bool _filaCheia = false;
  bool _gravando = false;
  bool _aguardandoSlot = false;
  int _tamanhoFila = 0;
  int _participantes = 0;
  String? _transmissaoAtivaId;
  String? _remetenteFalando;
  String? _avisoRemocao;

  Timer? _timeoutConexao;

  @override
  void initState() {
    super.initState();
    // Sem isso, uma falha silenciosa na conexão (que não gera nenhum evento
    // de erro do servidor — ex.: o handshake do WebSocket nem chega a abrir)
    // deixaria a tela girando no "Entrando no canal…" para sempre, sem
    // nenhum jeito de o usuário saber que algo deu errado.
    _timeoutConexao = Timer(const Duration(seconds: 10), () {
      if (!mounted || _conectado) return;
      setState(() {
        _avisoRemocao ??= 'Não foi possível conectar ao canal. Verifique sua conexão e tente novamente.';
      });
    });
    _iniciar();
  }

  Future<void> _iniciar() async {
    try {
      // Conectar e entrar no canal vem primeiro, e não pode esperar nada que
      // dependa do usuário responder um prompt do navegador (microfone,
      // localização) — senão a entrada no canal fica pendurada atrás de uma
      // permissão que ele pode demorar a conceder, enquanto a tela já mostra
      // tudo pronto para falar.
      await _canalService.conectar();
      _assinaturaEventos = _canalService.eventos.listen(_processarEvento);
      _canalService.entrarCanal(widget.canalId);
    } catch (_) {
      if (!mounted) return;
      setState(
        () => _avisoRemocao = 'Não foi possível conectar ao canal. Verifique sua conexão e tente novamente.',
      );
      return;
    }

    await _audioService.abrirParaOuvir();
    await _audioService.iniciarReproducao();

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
        _timeoutConexao?.cancel();
        final tamanhoFila = evento.dados?['tamanho_fila'] as int?;
        final participantes = evento.dados?['participantes'] as int?;
        setState(() {
          _conectado = true;
          if (tamanhoFila != null) {
            _tamanhoFila = tamanhoFila;
            _filaCheia = tamanhoFila >= 10;
          }
          if (participantes != null) _participantes = participantes;
        });
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

      // Qualquer erro do servidor (ex.: sem_permissao ao solicitar slot
      // antes de a entrada no canal ter sido confirmada) precisa liberar o
      // estado de espera — sem isso o botão fica preso em "aguardando"
      // indefinidamente, sem nenhum feedback pro usuário.
      case 'erro':
        setState(() {
          _aguardandoSlot = false;
          _avisoRemocao = evento.dados?['mensagem'] as String? ??
              'Não foi possível completar a ação.';
        });
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
    if (!_conectado || _filaCheia || _gravando || _aguardandoSlot) return;
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
    _timeoutConexao?.cancel();
    _assinaturaEventos?.cancel();
    _assinaturaGravacao?.cancel();
    _canalService.sairCanal(widget.canalId);
    unawaited(_canalService.desconectar());
    unawaited(_audioService.fechar());
    unawaited(_localizacaoService.parar());
    unawaited(BackgroundService.parar());
    super.dispose();
  }

  void _destravarAudioSeNecessario() {
    if (_audioDestravado) return;
    setState(() => _audioDestravado = true);
    unawaited(_audioService.destravarAudioNoToque());
  }

  String _iniciais(String nome) {
    final partes = nome.trim().split(RegExp(r'\s+'));
    final a = partes.isNotEmpty && partes[0].isNotEmpty ? partes[0][0] : '';
    final b = partes.length > 1 && partes[1].isNotEmpty ? partes[1][0] : '';
    return (a + b).toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    if (!_conectado) {
      return Scaffold(
        appBar: AppBar(title: Text(widget.nomeCanal)),
        body: Center(
          child: _avisoRemocao != null
              ? Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.error_outline, color: Colors.redAccent, size: 40),
                      const SizedBox(height: 12),
                      Text(_avisoRemocao!, textAlign: TextAlign.center),
                      const SizedBox(height: 20),
                      FilledButton(
                        onPressed: () => Navigator.of(context).pop(),
                        child: const Text('Voltar'),
                      ),
                    ],
                  ),
                )
              : const Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(),
                    SizedBox(height: 16),
                    Text('Entrando no canal…'),
                  ],
                ),
        ),
      );
    }

    final podeFalar = !_filaCheia && !_gravando && !_aguardandoSlot;
    final falando = _remetenteFalando != null;

    return Scaffold(
      appBar: AppBar(
        title: Text(widget.nomeCanal),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(20),
          child: Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              '$_participantes participante(s) · fila $_tamanhoFila/10',
              style: const TextStyle(color: Colors.white70, fontSize: 12),
            ),
          ),
        ),
      ),
      body: Listener(
        behavior: HitTestBehavior.translucent,
        onPointerDown: (_) => _destravarAudioSeNecessario(),
        child: Column(
        children: [
          if (!_audioDestravado)
            MaterialBanner(
              content: const Text('🔊 Toque em qualquer lugar da tela para ativar o áudio'),
              actions: [
                TextButton(
                  onPressed: _destravarAudioSeNecessario,
                  child: const Text('Ativar'),
                ),
              ],
            ),
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
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 250),
                child: falando
                    ? _bolhaFalando(_remetenteFalando!)
                    : _filaCheia
                    ? const _AvisoCentral(
                        icone: Icons.hourglass_top,
                        texto: 'Fila cheia — aguarde para falar',
                        cor: Colors.orange,
                      )
                    : const _AvisoCentral(
                        icone: Icons.radio,
                        texto: 'Canal em silêncio',
                        cor: Colors.white38,
                      ),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.only(bottom: 40, top: 8),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                _botaoFalar(podeFalar),
                const SizedBox(height: 12),
                Text(
                  _gravando
                      ? 'Transmitindo…'
                      : _aguardandoSlot
                      ? 'Aguardando vez na fila…'
                      : podeFalar
                      ? 'Segure para falar'
                      : 'Aguarde',
                  style: TextStyle(
                    color: Theme.of(
                      context,
                    ).textTheme.bodySmall?.color?.withValues(alpha: 0.8),
                    fontSize: 13,
                  ),
                ),
              ],
            ),
          ),
        ],
        ),
      ),
    );
  }

  Widget _bolhaFalando(String nome) {
    return Column(
      key: const ValueKey('falando'),
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 16),
          decoration: BoxDecoration(
            color: AppTheme.azulSecundario.withValues(alpha: 0.15),
            borderRadius: BorderRadius.circular(20),
            border: Border.all(color: AppTheme.azulSecundario, width: 1.4),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              CircleAvatar(
                radius: 20,
                backgroundColor: AppTheme.corDoAvatar(nome),
                child: Text(
                  _iniciais(nome),
                  style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    nome,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const Text('falando agora', style: TextStyle(fontSize: 12)),
                ],
              ),
              const SizedBox(width: 10),
              const Icon(Icons.volume_up, color: AppTheme.azulSecundario),
            ],
          ),
        ),
      ],
    );
  }

  Widget _botaoFalar(bool podeFalar) {
    final cor = !podeFalar && !_gravando
        ? Colors.grey
        : (_gravando ? Colors.redAccent : AppTheme.azulSecundario);

    return GestureDetector(
      onTapDown: (_) => _iniciarFala(),
      onTapUp: (_) => _pararFala(),
      onTapCancel: _pararFala,
      child: TweenAnimationBuilder<double>(
        tween: Tween(begin: 1, end: _gravando ? 1.18 : 1),
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOut,
        builder: (context, escala, filho) {
          return Stack(
            alignment: Alignment.center,
            children: [
              if (_gravando)
                Container(
                  width: 96 * escala,
                  height: 96 * escala,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: Colors.redAccent.withValues(alpha: 0.25),
                  ),
                ),
              filho!,
            ],
          );
        },
        child: CircleAvatar(
          radius: 48,
          backgroundColor: cor,
          child: _aguardandoSlot
              ? const SizedBox(
                  width: 28,
                  height: 28,
                  child: CircularProgressIndicator(
                    strokeWidth: 2.5,
                    color: Colors.white,
                  ),
                )
              : const Icon(Icons.mic, color: Colors.white, size: 40),
        ),
      ),
    );
  }
}

class _AvisoCentral extends StatelessWidget {
  const _AvisoCentral({
    required this.icone,
    required this.texto,
    required this.cor,
  });

  final IconData icone;
  final String texto;
  final Color cor;

  @override
  Widget build(BuildContext context) {
    return Column(
      key: ValueKey(texto),
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icone, size: 40, color: cor),
        const SizedBox(height: 10),
        Text(texto, style: TextStyle(color: cor)),
      ],
    );
  }
}

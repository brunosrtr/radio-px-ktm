import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show kIsWeb;

import '../../core/http_client.dart';
import '../../core/theme.dart';
import '../../services/audio_service.dart';
import '../../services/background_service.dart';
import '../../services/canal_service.dart';
import '../../services/localizacao_service.dart';
import '../../services/ptt_controller.dart';

/// Tela do canal ativo: botão push-to-talk, bloqueio quando a fila está
/// cheia e exibição do nome de quem está falando (FR-04 a FR-09, FR-21).
class CanalAtivoPage extends StatefulWidget {
  const CanalAtivoPage({
    super.key,
    required this.canalId,
    required this.nomeCanal,
    this.silenciadoInicial = false,
  });

  final String canalId;
  final String nomeCanal;

  /// Se a pessoa já tinha silenciado este canal (vem da lista).
  final bool silenciadoInicial;

  @override
  State<CanalAtivoPage> createState() => _CanalAtivoPageState();
}

class _CanalAtivoPageState extends State<CanalAtivoPage> {
  final _canalService = CanalService();
  final _audioService = AudioService();
  final _localizacaoService = LocalizacaoService();

  StreamSubscription<EventoCanal>? _assinaturaEventos;
  late final ControladorPtt _ptt;
  Future<void>? _inicializacao;
  bool _encerrando = false;
  bool _conexaoEncerrada = false;

  bool _conectado = false;
  bool _audioDestravado = !kIsWeb;
  bool _filaCheia = false;
  bool get _gravando => _ptt.gravando;
  bool get _aguardandoSlot => _ptt.aguardando;
  int _tamanhoFila = 0;
  int _participantes = 0;
  String? _remetenteFalando;
  String? _avisoRemocao;
  late bool _silenciado = widget.silenciadoInicial;
  final _api = ApiClient();

  Timer? _timeoutConexao;
  Timer? _reconexaoTimer;
  Future<void>? _reconectando;
  bool _servicosIniciados = false;
  bool _reconexaoPermitida = true;

  @override
  void initState() {
    super.initState();
    _audioService.aoFalharReproducao = _falharAudio;
    _ptt = ControladorPtt(
      solicitarSlot: () => _canalService.solicitarSlot(widget.canalId),
      finalizarTransmissao: _canalService.finalizarTransmissao,
      cancelarTransmissao: _canalService.cancelarTransmissao,
      iniciarGravacao: _audioService.iniciarGravacao,
      pararGravacao: _audioService.pararGravacao,
      enviarAudio: _canalService.enviarChunkAudio,
      aoMudar: () {
        if (mounted && !_encerrando) setState(() {});
      },
      aoFalhar: (erro) {
        if (mounted && !_encerrando) {
          setState(
            () => _avisoRemocao = erro is FalhaAudio ? erro.mensagem : 'Não foi possível transmitir. Solte o botão e tente novamente.',
          );
        }
      },
    );
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
    _inicializacao = _iniciar();
  }

  Future<void> _iniciar() async {
    try {
      // O player precisa estar pronto antes de receber o primeiro chunk.
      await _audioService.abrirParaOuvir();
      if (_encerrando) return;
      await _audioService.iniciarReproducao();
      if (_encerrando) return;
      await _canalService.conectar();
      if (_encerrando) return;
      _assinaturaEventos = _canalService.eventos.listen(
        _processarEvento,
        onDone: _perderConexao,
        onError: (Object _) => _perderConexao(),
      );
      _canalService.entrarCanal(widget.canalId);
      await BackgroundService.iniciar();
      if (_encerrando) return;
      await _iniciarHistoricoDePosicoes();
      _servicosIniciados = true;
    } on FalhaAudio catch (erro) {
      _falharAudio(erro);
    } catch (_) {
      _perderConexao();
    }
  }

  void _falharAudio(Object erro) {
    _reconexaoPermitida = false;
    _reconexaoTimer?.cancel();
    _perderConexao();
    if (!mounted || _encerrando) return;
    setState(
      () => _avisoRemocao = erro is FalhaAudio
          ? erro.mensagem
          : 'Não foi possível iniciar o áudio do canal.',
    );
  }

  void _perderConexao() {
    if (!mounted || _encerrando || _conexaoEncerrada) return;
    _conexaoEncerrada = true;
    _timeoutConexao?.cancel();
    unawaited(_ptt.interromper());
    unawaited(_audioService.pararReproducao());
    setState(() {
      _conectado = false;
      _remetenteFalando = null;
      _avisoRemocao =
          'Conexão perdida. Procurando o computador para reconectar…';
    });
    _agendarReconexao();
  }

  void _agendarReconexao() {
    if (_encerrando || !_reconexaoPermitida) return;
    _reconexaoTimer?.cancel();
    _reconexaoTimer = Timer(const Duration(seconds: 5), () {
      if (_encerrando || !_reconexaoPermitida) return;
      _reconectando = _reconectar();
    });
  }

  Future<void> _reconectar() async {
    try {
      await _inicializacao;
      if (_encerrando) return;
      await _ptt.interromper();
      await _assinaturaEventos?.cancel();
      await _canalService.desconectar();
      if (_encerrando) return;
      await _audioService.pararReproducao();
      await _audioService.iniciarReproducao();
      if (_encerrando) return;
      await _canalService.conectar();
      if (_encerrando) return;
      _conexaoEncerrada = false;
      _assinaturaEventos = _canalService.eventos.listen(
        _processarEvento,
        onDone: _perderConexao,
        onError: (Object _) => _perderConexao(),
      );
      _canalService.entrarCanal(widget.canalId);
      _timeoutConexao = Timer(const Duration(seconds: 10), () {
        if (!_conectado) _perderConexao();
      });
      if (!_servicosIniciados) {
        await BackgroundService.iniciar();
        if (_encerrando) return;
        await _iniciarHistoricoDePosicoes();
        _servicosIniciados = true;
      }
    } on FalhaAudio catch (erro) {
      _falharAudio(erro);
    } catch (_) {
      _conexaoEncerrada = true;
      _agendarReconexao();
    }
  }

  void _processarEvento(EventoCanal evento) {
    if (!mounted || _encerrando || _conexaoEncerrada) return;
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
          _avisoRemocao = null;
          _filaCheia = false;
          if (tamanhoFila != null) {
            _tamanhoFila = tamanhoFila;
            _filaCheia = tamanhoFila >= 10;
          }
          if (participantes != null) _participantes = participantes;
        });
        break;

      case 'slot_concedido':
        unawaited(
          _ptt.conceder(
            evento.dados!['transmissao_id'] as String,
            Duration(milliseconds: evento.dados!['duracao_maxima_ms'] as int),
          ),
        );
        break;

      case 'slot_negado':
        _ptt.negar();
        setState(() => _filaCheia = evento.dados?['motivo'] == 'fila_cheia');
        break;

      case 'transmissao_encerrada':
        unawaited(
          _ptt.servidorEncerrou(evento.dados!['transmissao_id'] as String),
        );
        break;

      case 'inicio_reproducao':
        _audioService.iniciarRecepcao();
        setState(
          () => _remetenteFalando = evento.dados?['remetente_nome'] as String?,
        );
        break;

      case 'fim_reproducao':
        _audioService.encerrarRecepcao();
        setState(() => _remetenteFalando = null);
        break;

      case 'removido_canal':
        _reconexaoPermitida = false;
        _reconexaoTimer?.cancel();
        _conexaoEncerrada = true;
        unawaited(_ptt.interromper());
        unawaited(_audioService.pararReproducao());
        _conectado = false;
        setState(
          () => _avisoRemocao = _mensagemRemocao(
            evento.dados?['motivo'] as String?,
          ),
        );
        break;

      // Qualquer erro do servidor (ex.: sem_permissao ao solicitar slot
      // antes de a entrada no canal ter sido confirmada) precisa liberar o
      // estado de espera — sem isso o botão fica preso em "aguardando"
      // indefinidamente, sem nenhum feedback pro usuário.
      case 'erro':
        if (!_conectado) {
          _reconexaoPermitida = false;
          _timeoutConexao?.cancel();
          _reconexaoTimer?.cancel();
        }
        _ptt.negar();
        setState(() {
          _avisoRemocao =
              evento.dados?['mensagem'] as String? ??
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

  void _iniciarFala() {
    if (!_conectado || _filaCheia || _encerrando) return;
    _ptt.pressionar();
  }

  Future<void> _pararFala() => _ptt.soltar();

  @override
  void dispose() {
    _encerrando = true;
    _reconexaoTimer?.cancel();
    _timeoutConexao?.cancel();
    unawaited(_ptt.fechar());
    unawaited(_encerrarRecursos());
    super.dispose();
  }

  Future<void> _encerrarRecursos() async {
    await _inicializacao;
    await _reconectando;
    await _ptt.fechar();
    await _assinaturaEventos?.cancel();
    _canalService.sairCanal(widget.canalId);
    await _canalService.desconectar();
    await _audioService.fechar();
    await _localizacaoService.parar();
    await BackgroundService.parar();
  }

  void _destravarAudioSeNecessario() {
    if (_audioDestravado) return;
    setState(() => _audioDestravado = true);
    unawaited(_audioService.destravarAudioNoToque().catchError(_falharAudio));
  }

  /// Silencia ou reativa o canal sem sair dele (FR-019); o servidor para de
  /// enviar áudio na hora.
  Future<void> _alternarSilencio() async {
    final novo = !_silenciado;
    setState(() => _silenciado = novo);
    try {
      await _api.dio.put('/canais/${widget.canalId}/preferencia', data: {'silenciado': novo});
    } catch (_) {
      if (!mounted) return;
      setState(() => _silenciado = !novo);
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Não foi possível mudar o som do canal.')));
    }
  }

  /// Contas com empresa enviam o trajeto; as do público só a posição atual,
  /// para a geocerca (FR-034 da 003-rede-orbit) — o LocalizacaoService
  /// decide pelo tipo de conta.
  Future<void> _iniciarHistoricoDePosicoes() => _localizacaoService.iniciar();

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
                      Icon(Icons.error_outline, color: Theme.of(context).colorScheme.error, size: 44),
                      const SizedBox(height: 12),
                      Text(_avisoRemocao!, textAlign: TextAlign.center, style: Theme.of(context).textTheme.titleMedium),
                      const SizedBox(height: 20),
                      FilledButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Voltar')),
                    ],
                  ),
                )
              : const Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [CircularProgressIndicator(), SizedBox(height: 16), Text('Entrando no canal…')],
                ),
        ),
      );
    }

    final tema = Theme.of(context);
    final podeFalar = _conectado && !_filaCheia && !_ptt.ocupado;

    return Scaffold(
      body: SafeArea(
        child: Listener(
          behavior: HitTestBehavior.translucent,
          onPointerDown: (_) => _destravarAudioSeNecessario(),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(8, 8, 16, 24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    IconButton(
                      tooltip: 'Voltar aos canais',
                      iconSize: 28,
                      onPressed: () => Navigator.of(context).pop(),
                      icon: const Icon(Icons.chevron_left),
                    ),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            widget.nomeCanal,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: tema.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700),
                          ),
                          Text(
                            '$_participantes no canal${_silenciado ? ' · silenciado' : ''}',
                            style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                if (!_audioDestravado)
                  Padding(
                    padding: const EdgeInsets.only(left: 8, top: 8),
                    child: MaterialBanner(
                      content: const Text('Toque em qualquer lugar da tela para ativar o áudio'),
                      actions: [TextButton(onPressed: _destravarAudioSeNecessario, child: const Text('Ativar'))],
                    ),
                  ),
                if (_avisoRemocao != null)
                  Padding(
                    padding: const EdgeInsets.only(left: 8, top: 8),
                    child: MaterialBanner(
                      content: Text(_avisoRemocao!),
                      actions: [TextButton(onPressed: () => setState(() => _avisoRemocao = null), child: const Text('OK'))],
                    ),
                  ),
                const SizedBox(height: 12),
                Padding(padding: const EdgeInsets.only(left: 8), child: _painelEstado(tema)),
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, limites) {
                      // Botão grande para quem dirige: ~40% da altura útil,
                      // limitado pela largura (US6).
                      final tamanho = math.max(120.0, math.min(300.0, math.min(limites.maxHeight * 0.78, limites.maxWidth * 0.78)));
                      return Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          _botaoFalar(podeFalar, tamanho),
                          const SizedBox(height: 12),
                          Text(
                            _gravando
                                ? 'Solte para enviar'
                                : _aguardandoSlot
                                ? 'Aguardando sua vez na fila…'
                                : podeFalar
                                ? 'Segure para falar, solte para enviar'
                                : _filaCheia
                                ? 'Fila cheia — aguarde para falar'
                                : 'Aguarde',
                            style: tema.textTheme.bodyLarge?.copyWith(color: tema.colorScheme.onSurfaceVariant),
                          ),
                        ],
                      );
                    },
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.only(left: 8),
                  child: Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          style: OutlinedButton.styleFrom(minimumSize: const Size.fromHeight(60)),
                          onPressed: _alternarSilencio,
                          icon: Icon(_silenciado ? Icons.volume_up : Icons.volume_off),
                          label: Text(_silenciado ? 'Ativar som' : 'Silenciar'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: OutlinedButton.icon(
                          style: OutlinedButton.styleFrom(
                            minimumSize: const Size.fromHeight(60),
                            foregroundColor: tema.colorScheme.error,
                            side: BorderSide(color: tema.colorScheme.error.withValues(alpha: 0.6)),
                          ),
                          onPressed: () => Navigator.of(context).pop(),
                          icon: const Icon(Icons.logout),
                          label: const Text('Sair'),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// Quem está falando (ou o estado do canal) em letra grande, mais a fila.
  Widget _painelEstado(ThemeData tema) {
    final Widget topo;
    if (_gravando) {
      topo = _linhaEstado(
        tema,
        cor: AppTheme.vermelhoFalando,
        icone: Icons.mic,
        rotulo: 'TRANSMITINDO',
        corRotulo: const Color(0xFFFF8A8A),
        titulo: 'Você',
      );
    } else if (_remetenteFalando != null) {
      final nome = _remetenteFalando!;
      topo = _linhaEstado(
        tema,
        cor: AppTheme.corDoAvatar(nome),
        texto: AppTheme.iniciais(nome),
        rotulo: 'FALANDO AGORA',
        corRotulo: AppTheme.verdeFalando,
        titulo: nome,
      );
    } else {
      topo = _linhaEstado(
        tema,
        cor: tema.colorScheme.outlineVariant,
        icone: Icons.radio,
        rotulo: 'CANAL EM SILÊNCIO',
        corRotulo: tema.colorScheme.onSurfaceVariant,
        titulo: 'Ninguém falando',
      );
    }
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AnimatedSwitcher(duration: const Duration(milliseconds: 200), child: topo),
            const SizedBox(height: 12),
            const Divider(),
            const SizedBox(height: 10),
            Text(
              _tamanhoFila == 0
                  ? 'Fila vazia'
                  : '$_tamanhoFila ${_tamanhoFila == 1 ? 'mensagem' : 'mensagens'} na fila (máx. 10)',
              style: tema.textTheme.bodyLarge,
            ),
          ],
        ),
      ),
    );
  }

  Widget _linhaEstado(
    ThemeData tema, {
    required Color cor,
    required String rotulo,
    required Color corRotulo,
    required String titulo,
    IconData? icone,
    String? texto,
  }) {
    return Row(
      key: ValueKey(rotulo + titulo),
      children: [
        CircleAvatar(
          radius: 28,
          backgroundColor: cor,
          child: icone != null
              ? Icon(icone, color: Colors.white)
              : Text(texto ?? '', style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 20)),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(rotulo, style: TextStyle(fontSize: 13, fontWeight: FontWeight.w700, letterSpacing: 0.8, color: corRotulo)),
              Text(
                titulo,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: tema.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _botaoFalar(bool podeFalar, double tamanho) {
    final cor = _gravando
        ? AppTheme.vermelhoFalando
        : podeFalar
        ? AppTheme.azulSecundario
        : Colors.grey.shade700;
    final anel = _gravando ? const Color(0xFFFF6B6B) : (podeFalar ? AppTheme.azulPrimario : Colors.grey.shade600);

    return Semantics(
      button: true,
      label: 'Segure para falar',
      child: GestureDetector(
        onTapDown: (_) => _iniciarFala(),
        onTapUp: (_) => _pararFala(),
        onTapCancel: _pararFala,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          width: tamanho,
          height: tamanho,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: cor,
            border: Border.all(color: anel, width: 8),
            boxShadow: _gravando
                ? [BoxShadow(color: const Color(0xFFFF6B6B).withValues(alpha: 0.35), blurRadius: 40, spreadRadius: 6)]
                : null,
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              if (_aguardandoSlot)
                const SizedBox(width: 40, height: 40, child: CircularProgressIndicator(strokeWidth: 3, color: Colors.white))
              else
                Icon(Icons.mic, color: Colors.white, size: tamanho * 0.25),
              const SizedBox(height: 8),
              Text(
                _gravando ? 'Falando…' : 'Falar',
                style: TextStyle(color: Colors.white, fontSize: tamanho * 0.085, fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

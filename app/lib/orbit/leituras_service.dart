import 'dart:async';

import 'package:flutter/foundation.dart';

import '../data/fila_leituras_local.dart';
import 'deduplicador_leituras.dart';
import 'leitor_orbit.dart';

/// Posição do celular no momento de uma leitura.
class PosicaoLeitura {
  const PosicaoLeitura({
    required this.latitude,
    required this.longitude,
    required this.capturadaEm,
    this.precisaoMetros,
  });

  final double latitude;
  final double longitude;
  final double? precisaoMetros;
  final DateTime capturadaEm;
}

/// Devolve a posição mais recente do celular, ou null se não houver.
typedef ObterPosicao = Future<PosicaoLeitura?> Function();

/// O servidor recusou o lote por estar fora do contrato (422). Essas
/// leituras nunca serão aceitas — ficar reenviando travaria a fila.
class LoteRecusado implements Exception {
  const LoteRecusado(this.motivo);
  final String motivo;
}

/// O servidor recusou porque o usuário não consente mais em ser ponto de
/// transmissão (403 sem_consentimento) — por exemplo, revogou em outro
/// aparelho. As leituras na fila não podem mais ser enviadas.
class SemConsentimento implements Exception {
  const SemConsentimento();
}

/// Envio das leituras ao backend (POST /leituras e /ponto/sinal).
abstract interface class EnviadorLeituras {
  Future<void> enviarLote(List<Map<String, dynamic>> leituras);
  Future<void> enviarSinal();
}

/// Transforma detecções em leituras e as entrega ao servidor (US1):
/// junta a posição do celular, deduplica, guarda na fila offline, avisa com
/// o bip e envia em lotes. Nunca envia uma requisição por leitura (FR-004).
class LeiturasService {
  LeiturasService({
    required FilaLeituras fila,
    required ObterPosicao obterPosicao,
    required EnviadorLeituras enviador,
    DeduplicadorLeituras? deduplicador,
    Future<void> Function()? aoRegistrar,
    DateTime Function()? relogio,
    this.aoPerderConsentimento,
  }) : _fila = fila, // ignore: prefer_initializing_formals
       _obterPosicao = obterPosicao, // ignore: prefer_initializing_formals
       _enviador = enviador, // ignore: prefer_initializing_formals
       deduplicador = deduplicador ?? DeduplicadorLeituras(),
       _aoRegistrar = aoRegistrar, // ignore: prefer_initializing_formals
       _relogio = relogio ?? DateTime.now;

  /// Idade máxima da posição usada numa leitura: sem posição recente, a
  /// leitura é descartada — ela não teria utilidade para a KTM.
  static const idadeMaximaPosicao = Duration(seconds: 60);
  static const tamanhoLote = 100;
  static const intervaloEnvio = Duration(seconds: 15);
  static const intervaloSinal = Duration(minutes: 5);

  final FilaLeituras _fila;
  final ObterPosicao _obterPosicao;
  final EnviadorLeituras _enviador;
  final DeduplicadorLeituras deduplicador;
  final Future<void> Function()? _aoRegistrar;
  final DateTime Function() _relogio;

  /// Chamado quando o servidor diz que não há mais consentimento.
  final void Function()? aoPerderConsentimento;

  /// Leituras registradas nesta sessão (mostradas ao usuário, sem detalhes
  /// do Orbit — FR-011).
  final leiturasSessao = ValueNotifier<int>(0);

  StreamSubscription<DeteccaoOrbit>? _assinatura;
  Timer? _timerEnvio;
  Timer? _timerSinal;
  bool _enviando = false;

  Future<void> iniciar(Stream<DeteccaoOrbit> deteccoes) async {
    await _fila.abrir();
    await _fila.descartarAntigas(_relogio());
    await _assinatura?.cancel();
    _assinatura = deteccoes.listen((d) => unawaited(processar(d)));
    _timerEnvio ??= Timer.periodic(intervaloEnvio, (_) => unawaited(enviarPendentes()));
    _timerSinal ??= Timer.periodic(intervaloSinal, (_) => unawaited(_sinal()));
    unawaited(_sinal());
    unawaited(enviarPendentes());
  }

  /// Registra uma detecção como leitura, se o deduplicador aceitar e houver
  /// posição recente. Devolve true quando a leitura entrou na fila.
  Future<bool> processar(DeteccaoOrbit deteccao) async {
    if (!deduplicador.aceitar(deteccao.orbitId, deteccao.instante)) return false;
    final posicao = await _obterPosicao();
    if (posicao == null || deteccao.instante.difference(posicao.capturadaEm).abs() > idadeMaximaPosicao) {
      return false;
    }
    await _fila.adicionar(LeituraLocal(
      orbitId: deteccao.orbitId,
      latitude: posicao.latitude,
      longitude: posicao.longitude,
      precisaoMetros: posicao.precisaoMetros,
      rssi: deteccao.rssi,
      detectadoEm: deteccao.instante,
    ));
    leiturasSessao.value++;
    await _aoRegistrar?.call();
    return true;
  }

  /// Envia a fila em lotes de até [tamanhoLote]. Só remove o que o servidor
  /// confirmou; sem conexão, tudo continua na fila para o próximo ciclo.
  Future<void> enviarPendentes() async {
    if (_enviando) return;
    _enviando = true;
    try {
      while (true) {
        final pendentes = _fila.listarPendentes(limite: tamanhoLote);
        if (pendentes.isEmpty) return;
        try {
          await _enviador.enviarLote([for (final l in pendentes.values) l.paraJson()]);
        } on LoteRecusado {
          // Descarta o lote inválido (ex.: relógio do celular muito errado)
          // para não bloquear as leituras seguintes.
        } on SemConsentimento {
          // Sem consentimento, nada do que está na fila pode sair do
          // aparelho (FR-028, Princípio VII).
          await descartarPendentes();
          aoPerderConsentimento?.call();
          return;
        }
        await _fila.removerConfirmados(pendentes.keys);
        if (pendentes.length < tamanhoLote) return;
      }
    } catch (_) {
      // Fica para o próximo ciclo (FR-005).
    } finally {
      _enviando = false;
    }
  }

  Future<void> _sinal() async {
    try {
      await _enviador.enviarSinal();
    } catch (_) {}
  }

  /// Apaga a fila local sem enviar — usado quando o usuário revoga o
  /// consentimento.
  Future<void> descartarPendentes() async {
    await _fila.abrir();
    await _fila.removerConfirmados(_fila.listarPendentes(limite: 1 << 30).keys);
  }

  Future<void> parar({bool enviarPendentes = true}) async {
    await _assinatura?.cancel();
    _assinatura = null;
    _timerEnvio?.cancel();
    _timerEnvio = null;
    _timerSinal?.cancel();
    _timerSinal = null;
    if (enviarPendentes) await this.enviarPendentes();
  }
}

import 'dart:typed_data';

/// Formato **provisório** do sinal do Orbit (contracts/sinal-orbit.md da
/// 003-rede-orbit). É o único lugar do app que conhece esse formato
/// (FR-016): quando a KTM fornecer o protocolo do Orbit real, só este
/// arquivo muda.
///
/// - Pacote de anúncio: lista completa de Service UUIDs de 128 bits com o
///   [uuidRedeOrbit] — é o que o filtro de busca (inclusive no iOS em
///   segundo plano) procura.
/// - Resposta de varredura: Service Data desse UUID com 8 bytes:
///   versão (1), tipo (1), identificação (3), reservado (3).
abstract final class OrbitSinal {
  /// Service UUID da Rede Orbit.
  static const uuidRedeOrbit = '6b7e0001-2c4f-4e8a-9d3b-0a5f1c7e9b21';

  static const versao = 0x01;
  static const tipoVirtual = 0x01;
  static const tipoReal = 0x02;
  static const tamanhoCarga = 8;

  static final _formatoId = RegExp(r'^ORBIT-[0-9A-F]{6}$');

  /// Informa se [orbitId] segue o formato `ORBIT-` + 6 hexadecimais.
  static bool idValido(String orbitId) => _formatoId.hasMatch(orbitId);

  /// Monta os 8 bytes de carga que o Orbit virtual anuncia.
  static Uint8List codificarCarga(String orbitId, {int tipo = tipoVirtual}) {
    if (!idValido(orbitId)) {
      throw ArgumentError.value(orbitId, 'orbitId', 'use ORBIT- seguido de 6 hexadecimais');
    }
    final hex = orbitId.substring(6);
    return Uint8List.fromList([
      versao,
      tipo,
      for (var i = 0; i < 6; i += 2) int.parse(hex.substring(i, i + 2), radix: 16),
      0,
      0,
      0,
    ]);
  }

  /// Lê a identificação do Orbit a partir do Service Data recebido, ou
  /// devolve null quando a carga não é de um Orbit conhecido (versão ou tipo
  /// desconhecidos, tamanho errado).
  static String? lerCarga(List<int>? carga) {
    if (carga == null || carga.length < tamanhoCarga) return null;
    if (carga[0] != versao) return null;
    if (carga[1] != tipoVirtual && carga[1] != tipoReal) return null;
    final id = carga
        .sublist(2, 5)
        .map((b) => b.toRadixString(16).padLeft(2, '0'))
        .join()
        .toUpperCase();
    return 'ORBIT-$id';
  }
}

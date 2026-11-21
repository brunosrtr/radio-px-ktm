# Contrato do sinal Orbit (BLE) — `003-rede-orbit`

Definição **provisória**, usada pelo Orbit virtual até a KTM fornecer o protocolo do Orbit real
(spec, Pendências 1 e 2). Ela vive num único lugar no app (`app/lib/orbit/orbit_sinal.dart`), e trocar
este contrato não pode exigir mudanças fora desse arquivo (FR-016).

## Anúncio

| Campo BLE | Valor |
|---|---|
| Tipo | anúncio legado, não conectável |
| Flags | `0x06` (LE General Discoverable, BR/EDR não suportado) |
| Lista completa de Service UUIDs de 128 bits | `UUID Rede Orbit` = `6b7e0001-2c4f-4e8a-9d3b-0a5f1c7e9b21` |
| Service Data (128 bits) | UUID acima + 8 bytes de carga |
| Intervalo | `advertiseMode = lowLatency` na demonstração (~100 ms) |
| Potência | `txPowerLevel = high` na demonstração |

O tamanho total é 3 (flags) + 18 (UUID) + 26 (Service Data) = 47 bytes, **o que ultrapassa os 31 bytes** do anúncio legado. Por isso:
- **pacote de anúncio**: flags + lista de Service UUIDs (21 bytes), que é o que o filtro de busca, inclusive no iOS em segundo plano, precisa encontrar;
- **resposta de varredura** (scan response): Service Data (26 bytes).

O leitor usa busca ativa (padrão do `flutter_blue_plus`) para receber a resposta de varredura.

> Se nos testes de campo algum aparelho não entregar a resposta de varredura em segundo plano, a
> alternativa registrada é usar um **UUID de 16 bits** provisório para o Service Data (4 + 8 = 12 bytes
> de carga), que cabe junto no pacote principal. A troca fica restrita ao `orbit_sinal.dart`.

## Carga (8 bytes, big-endian)

| Byte | Campo | Valores |
|---|---|---|
| 0 | versão | `0x01` |
| 1 | tipo | `0x01` = Orbit virtual; `0x02` reservado para Orbit real |
| 2–4 | identificação | 3 bytes → `ORBIT-` + 6 dígitos hexadecimais maiúsculos |
| 5–7 | reservado | zeros na v1 (futura identificação rotativa/assinada) |

Identificação da v1: `ORBIT-` + hex maiúsculo dos bytes 2–4 (ex.: `0A 1B 2C` → `ORBIT-0A1B2C`), coerente com `orbit.id varchar(12)`.
Os bytes 5–7 ficam reservados para uma futura identificação rotativa ou assinada (Pendência 2) e são ignorados na v1.

## Regras do leitor

1. Buscar **sempre** com filtro `withServices: [UUID Rede Orbit]`.
2. Descartar resultados sem Service Data, com versão ≠ `0x01` ou com tipo desconhecido.
3. Descartar o próprio `orbitId` quando o aparelho estiver em modo Orbit virtual.
4. Passar `(orbitId, rssi, agora)` ao `DeduplicadorLeituras` (janela de 30 s por Orbit).

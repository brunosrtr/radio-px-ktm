import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/core/rede_local.dart';

void main() {
  test('QR válido preserva identidade e porta do computador', () {
    final servidor = ServidorLocal.doQr(
      '{"service":"radio-px-ktm","version":1,"id":"computador-a","url":"http://192.168.1.3:8081/"}',
    );
    expect(servidor.id, 'computador-a');
    expect(servidor.url, 'http://192.168.1.3:8081');
  });

  test('rejeita QR de outro serviço, versão ou com credenciais', () {
    for (final qr in [
      'texto qualquer',
      '{"service":"outro","version":1,"id":"a","url":"http://192.168.1.3"}',
      '{"service":"radio-px-ktm","version":2,"id":"a","url":"http://192.168.1.3"}',
      '{"service":"radio-px-ktm","version":1,"id":"","url":"http://192.168.1.3"}',
      '{"service":"radio-px-ktm","version":1,"id":"a","url":"http://user:password@192.168.1.3"}',
      '{"service":"radio-px-ktm","version":1,"id":"a","url":"file:///etc/passwd"}',
      '{"service":"radio-px-ktm","version":1,"id":"a","url":"http://192.168.1.3/auth/login"}',
    ]) {
      expect(() => ServidorLocal.doQr(qr), throwsFormatException);
    }
  });
  test('troca de IP reencontra a mesma instalação', () async {
    var urlAtual = 'http://192.168.1.2:8081';
    var buscas = 0;
    final rede = RedeLocal.paraTeste(
      descobrir: () async {
        buscas++;
        return [urlAtual];
      },
      verificar: (url, id) async => url == urlAtual
          ? ServidorLocal(id: 'computador-a', nome: 'KTM', url: url)
          : null,
    );
    await rede.procurar();
    expect(rede.apiUrl, urlAtual);
    urlAtual = 'http://10.0.0.5:8081';
    await rede.assegurarConexao(forcar: true);
    expect(rede.apiUrl, urlAtual);
    expect(rede.servidor!.id, 'computador-a');
    expect(rede.wsUrl, 'ws://10.0.0.5:8081');
    expect(buscas, 2);
    rede.dispose();
  });

  test('não troca automaticamente para outro computador', () async {
    var idAtual = 'a';
    final rede = RedeLocal.paraTeste(
      descobrir: () async => ['http://192.168.1.2:8081'],
      verificar: (url, id) async => id != null && id != idAtual
          ? null
          : ServidorLocal(id: idAtual, nome: 'KTM', url: url),
    );
    await rede.procurar();
    idAtual = 'b';
    await expectLater(rede.assegurarConexao(forcar: true), throwsStateError);
    expect(rede.servidor!.id, 'a');
    rede.dispose();
  });

  test(
    'vários servidores exigem seleção e buscas simultâneas são compartilhadas',
    () async {
      var buscas = 0;
      final rede = RedeLocal.paraTeste(
        descobrir: () async {
          buscas++;
          return ['http://192.168.1.2', 'http://192.168.1.3'];
        },
        verificar: (url, id) async =>
            ServidorLocal(id: url, nome: 'KTM', url: url),
      );
      await Future.wait([rede.procurar(), rede.procurar()]);
      expect(buscas, 1);
      expect(rede.servidor, isNull);
      expect(rede.encontrados, hasLength(2));
      await rede.selecionar(rede.encontrados.last);
      expect(rede.apiUrl, 'http://192.168.1.3');
      rede.dispose();
    },
  );
}

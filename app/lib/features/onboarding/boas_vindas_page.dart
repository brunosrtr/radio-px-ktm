import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/sessao_app.dart';
import '../../core/theme.dart';

/// Apresentação inicial em 3 telas que podem ser puladas (FR-035): a rádio,
/// o ponto de transmissão e a privacidade em mão dupla.
class BoasVindasPage extends StatefulWidget {
  const BoasVindasPage({super.key});

  @override
  State<BoasVindasPage> createState() => _BoasVindasPageState();
}

class _BoasVindasPageState extends State<BoasVindasPage> {
  final _paginas = PageController();
  int _atual = 0;

  @override
  void dispose() {
    _paginas.dispose();
    super.dispose();
  }

  Future<void> _concluir() async {
    await SessaoApp.instance.marcarApresentacaoVista();
    if (mounted) context.go('/cadastro');
  }

  void _proxima() {
    if (_atual == 2) {
      _concluir();
    } else {
      _paginas.nextPage(duration: const Duration(milliseconds: 280), curve: Curves.easeOut);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cores = Theme.of(context).colorScheme;
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
          child: Column(
            children: [
              Row(
                children: [
                  if (_atual > 0)
                    IconButton(
                      tooltip: 'Voltar',
                      onPressed: () => _paginas.previousPage(duration: const Duration(milliseconds: 280), curve: Curves.easeOut),
                      icon: const Icon(Icons.chevron_left),
                    )
                  else
                    Text(
                      'KTM · RÁDIO PX',
                      style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, letterSpacing: 2, color: cores.primary),
                    ),
                  const Spacer(),
                  TextButton(onPressed: _concluir, child: const Text('Pular')),
                ],
              ),
              Expanded(
                child: PageView(
                  controller: _paginas,
                  onPageChanged: (i) => setState(() => _atual = i),
                  children: const [
                    _Pagina(
                      icone: Icons.mic,
                      cor: AppTheme.azulPrimario,
                      titulo: 'A estrada inteira\nno mesmo canal.',
                      texto: 'Converse por voz com quem está no mesmo trecho. Aperte, fale, solte — como no rádio PX, mas pela internet.',
                    ),
                    _Pagina(
                      rotulo: 'REDE ORBIT',
                      icone: Icons.sensors,
                      cor: AppTheme.ambar,
                      titulo: 'Seu celular ajuda\na achar caminhões.',
                      texto: 'Quando você passa perto de um caminhão com o equipamento Orbit, seu celular faz um bip e avisa a KTM onde ele foi visto. Sozinho, em segundo plano.',
                    ),
                    _PaginaPrivacidade(),
                  ],
                ),
              ),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  for (var i = 0; i < 3; i++)
                    AnimatedContainer(
                      duration: const Duration(milliseconds: 200),
                      margin: const EdgeInsets.symmetric(horizontal: 4),
                      width: i == _atual ? 24 : 8,
                      height: 8,
                      decoration: BoxDecoration(
                        color: i == _atual ? cores.primary : cores.outlineVariant,
                        borderRadius: BorderRadius.circular(4),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 20),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  key: const Key('boas-vindas-proximo'),
                  style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(56)),
                  onPressed: _proxima,
                  child: Text(_atual == 2 ? 'Criar minha conta' : 'Próximo'),
                ),
              ),
              TextButton(
                onPressed: () async {
                  await SessaoApp.instance.marcarApresentacaoVista();
                  if (context.mounted) context.go('/entrar');
                },
                child: const Text('Já tenho conta'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Pagina extends StatelessWidget {
  const _Pagina({required this.icone, required this.cor, required this.titulo, required this.texto, this.rotulo});

  final IconData icone;
  final Color cor;
  final String titulo;
  final String texto;
  final String? rotulo;

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 24),
        child: Column(
          children: [
            Container(
              width: 180,
              height: 180,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: cor.withValues(alpha: 0.12),
                border: Border.all(color: cor.withValues(alpha: 0.35), width: 2),
              ),
              child: Icon(icone, size: 84, color: cor),
            ),
            const SizedBox(height: 32),
            if (rotulo != null) ...[
              Text(rotulo!, style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, letterSpacing: 2, color: cor)),
              const SizedBox(height: 10),
            ],
            Text(
              titulo,
              textAlign: TextAlign.center,
              style: tema.textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w700, height: 1.15),
            ),
            const SizedBox(height: 14),
            Text(
              texto,
              textAlign: TextAlign.center,
              style: tema.textTheme.bodyLarge?.copyWith(height: 1.5, color: tema.colorScheme.onSurfaceVariant),
            ),
          ],
        ),
      ),
    );
  }
}

class _PaginaPrivacidade extends StatelessWidget {
  const _PaginaPrivacidade();

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'VOCÊ NO CONTROLE',
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, letterSpacing: 2, color: AppTheme.ambar),
            ),
            const SizedBox(height: 10),
            Text(
              'Ninguém vê o que\nnão precisa ver.',
              style: tema.textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w700, height: 1.15),
            ),
            const SizedBox(height: 24),
            const _ItemPrivacidade(
              icone: Icons.visibility_off_outlined,
              cor: AppTheme.ambar,
              titulo: 'Você não vê o caminhão',
              texto: 'Seu celular só repassa o sinal. Placa, carga e dono continuam com a KTM.',
            ),
            const _ItemPrivacidade(
              icone: Icons.person_off_outlined,
              cor: AppTheme.ambar,
              titulo: 'A KTM não vê você',
              texto: 'Cada leitura vai com um código anônimo que muda todo dia — nunca seu nome ou e-mail.',
            ),
            const _ItemPrivacidade(
              icone: Icons.mic_off_outlined,
              cor: AppTheme.azulPrimario,
              titulo: 'A voz não fica gravada',
              texto: 'Cada mensagem toca uma vez e some. Não existe histórico de áudio.',
            ),
          ],
        ),
      ),
    );
  }
}

class _ItemPrivacidade extends StatelessWidget {
  const _ItemPrivacidade({required this.icone, required this.cor, required this.titulo, required this.texto});

  final IconData icone;
  final Color cor;
  final String titulo;
  final String texto;

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(icone, color: cor, size: 28),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(titulo, style: tema.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
                    const SizedBox(height: 4),
                    Text(texto, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

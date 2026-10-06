# Central: canais, presença e fila de voz

Abra `http://localhost:8081/painel/` no computador e entre como administrador.
A seção **Canais e voz** fica na barra lateral do mapa. Selecione um canal
para ver os participantes realmente conectados, independentemente da data
da última posição GPS. A lista é atualizada a cada três segundos.

## Ouvir a fila

Clique em **Ouvir canal** antes das falas que deseja acompanhar. A central
recebe as próximas transmissões e reproduz uma mensagem por vez, na ordem
FIFO do canal, depois que cada fala termina. Uma fala já em andamento quando
a escuta começa não é entregue parcialmente. Mensagens antigas já descartadas
não são recuperadas.

**Pausar** interrompe a reprodução e conserva as mensagens recebidas em memória;
**Continuar** retoma a fila. **Parar**, trocar de página ou perder a conexão
encerra a escuta e descarta os áudios. Para trocar de canal, pare a escuta.
A fila local suporta dez mensagens incluindo a atual; se a central ficar
pausada além desse limite, a escuta é encerrada com aviso. Áudio nunca é salvo
em arquivo, banco de dados, localStorage ou cache.

Use Chrome/Edge com suporte a WebCodecs no computador em **localhost** ou
em HTTPS. O navegador precisa liberar áudio pelo clique em **Ouvir canal**.
O formato é Opus mono, pacotes de 20 ms a 16 kHz; a central decodifica com
WebCodecs e usa a taxa efetiva da saída para a reprodução. O cabeçalho Opus
segue a [especificação WebCodecs](https://www.w3.org/TR/webcodecs-opus-codec-registration/).
O painel recebe áudio; transmissão de voz pela central não está nesta etapa.

## Criar e editar canais

Clique em **Novo canal**, informe nome e limite de participantes (5, 10, 15
ou 20) e salve. Os canais criados pertencem à empresa do administrador.
**Editar canal** fica disponível nos canais pertencentes à sua empresa.

Ao ativar **Limitar acesso por distância**, informe o raio em quilômetros e
selecione o centro no mapa. A área aparece como um círculo antes de salvar.
Essa regra mede a distância de cada celular até o ponto escolhido; ela não
mede a distância entre dois celulares. A central administrativa pode ouvir
os canais autorizados da empresa sem fornecer GPS nem ocupar vaga de motorista.

Os celulares devem permitir localização, manter GPS ligado e alcançar o
backend. O app envia uma posição antes de consultar/entrar em canais com área
limitada. A entrada fora do raio é recusada, e novas posições recebidas podem
remover quem saiu da área. Reduzir o raio no painel também reavalia os
participantes já conectados pela última posição conhecida. A posição é GPS,
não a distância de alcance do Wi-Fi. Mudanças de movimento dependem do recebimento
das posições, normalmente enviadas em lotes de 90 segundos.

O modo local depende da rede entre celulares e computador. QR code e raio
geográfico não dão acesso ao backend quando essa comunicação está bloqueada.

## Contas de caminhoneiros e veículos

No painel, clique em **Contas e veículos**. O acesso é exclusivo de
administradores e os cadastros pertencem à empresa da conta autenticada.
Informe nome, sobrenome, CPF válido, senha (mínimo de oito caracteres),
confirmação da senha e veículo utilizado. A senha é armazenada somente como
hash bcrypt e sua confirmação não é gravada.

O CPF, com ou sem pontuação, será o login da nova conta no aplicativo. Os
logins existentes continuam funcionando. Nome e sobrenome compõem o nome
completo exibido nas falas e no mapa.

Escolha um veículo já cadastrado ou marque **Cadastrar um novo veículo nesta
conta** e informe nome/modelo e placa. O cadastro conjunto é transacional:
se a conta não puder ser criada, o novo veículo também não é cadastrado.
Também é possível cadastrar um veículo separadamente na mesma página.
A lista mostra os caminhoneiros cadastrados, CPF, login e veículo.

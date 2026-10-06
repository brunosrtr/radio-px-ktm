/* A fila de voz existe somente em memória. Nenhum áudio usa storage,
   arquivos, downloads, blobs persistidos ou caches. */
'use strict';
const CentralVoz = (() => {
  const el = id => document.getElementById(id);
  const configOpus = {codec:'opus', sampleRate:48000, numberOfChannels:1, description:new Uint8Array([79,112,117,115,72,101,97,100,1,1,0,0,128,62,0,0,0,0,0])};
  let token, empresa, mapa, atualizarMapa, editando = null, circuloArea, escolhendoCentro = false, canais = [], consulta, consultando = false, disponivel = false;
  let socket, contexto, fonte, fila = [], recebendo, tocando, pausado = false, versao = 0, decodificando = false;
  const texto = (id, valor) => { el(id).textContent = valor; };
  function itemLista(id, textos) {
    const itens = textos.map(valor => { const li = document.createElement('li'); li.textContent = valor; return li; });
    el(id).replaceChildren(...itens);
  }
  function canalSelecionado() { return canais.find(c => c.id === el('canal-central').value); }
  function canalDoUsuario(id) {
    if (!disponivel) return 'Presença no canal indisponível';
    const ativos = canais.filter(c => c.participantes.some(p => p.usuario_id === id));
    return ativos.length ? ativos.map(c => c.nome).join(', ') : 'Sem canal conectado';
  }
  function mostrarCanal() {
    const c = canalSelecionado();
    el('ouvir-canal').disabled = !c || !disponivel || !!socket;
    el('editar-canal').disabled = !c?.pode_editar || !disponivel;
    texto('status-canais', !disponivel ? 'Não foi possível atualizar os canais. Tentando novamente…' : c ? c.participantes.length + ' participante(s) conectado(s)' + (c.geocerca_ativa ? ' · raio de ' + (c.raio_metros / 1000) + ' km' : ' · sem limite de distância') : 'Nenhum canal disponível. Crie um canal para começar.');
    itemLista('participantes-central', c && disponivel && c.participantes.length ? c.participantes.map(p => p.nome) : ['Nenhum participante conectado.']);
    itemLista('fila-servidor', c && disponivel && c.fila.length ? c.fila.map(m => m.remetente_nome + ' · ' + m.estado) : ['Fila vazia.']);
  }
  async function consultar() {
    if (consultando || !token) return;
    consultando = true;
    try {
      const resposta = await fetch('/central/canais', {headers:{Authorization:'Bearer ' + token},cache:'no-store'});
      if (!resposta.ok) throw new Error();
      const dados = await resposta.json();
      const anterior = el('canal-central').value;
      canais = dados.canais;
      disponivel = true;
      const opcoes = canais.map(c => { const opcao = document.createElement('option'); opcao.value = c.id; opcao.textContent = c.nome; return opcao; });
      el('canal-central').replaceChildren(...opcoes);
      if (canais.some(c => c.id === anterior)) el('canal-central').value = anterior;
      else if (socket) parar('O canal deixou de estar disponível. A escuta foi encerrada.');
    } catch { disponivel = false; }
    finally { consultando = false; mostrarCanal(); atualizarMapa?.(); }
  }
  function mostrarFila() {
    const itens = [];
    if (tocando) itens.push(tocando.nome + (pausado ? ' · pausada' : ' · tocando'));
    itens.push(...fila.map(m => m.nome + (m.pronta ? ' · aguardando' : ' · recebendo')));
    itemLista('fila-central', itens.length ? itens : ['Nenhuma mensagem recebida.']);
    el('pausar-voz').disabled = !socket;
    el('parar-voz').disabled = !socket && !tocando && !fila.length;
    el('pausar-voz').textContent = pausado ? 'Continuar' : 'Pausar';
  }
  function parar(mensagem = 'Escuta encerrada. O áudio pendente foi descartado.') {
    versao++;
    if (socket) { const antigo = socket; socket = null; antigo.onclose = null; antigo.close(); }
    if (fonte) { fonte.onended = null; fonte.stop(); fonte.disconnect(); fonte.buffer = null; fonte = null; }
    if (contexto) { contexto.close().catch(() => {}); contexto = null; }
    fila = []; recebendo = null; tocando = null; pausado = false; decodificando = false;
    el('canal-central').disabled = false;
    texto('status-voz', mensagem); mostrarFila(); mostrarCanal();
  }
  async function ouvir() {
    const c = canalSelecionado();
    if (!c || socket) return;
    const tentativa = ++versao;
    el('ouvir-canal').disabled = true;
    try {
      if (!window.isSecureContext || !window.AudioDecoder) throw new Error('Para ouvir, abra a central em localhost no Chrome/Edge atualizado, ou use HTTPS.');
      const suporte = await AudioDecoder.isConfigSupported(configOpus);
      if (!suporte.supported) throw new Error('Este navegador não oferece reprodução Opus. Use Chrome ou Edge atualizado.');
      if (tentativa !== versao) return;
      contexto = new AudioContext();
      await contexto.resume();
      if (tentativa !== versao) return;
      el('canal-central').disabled = true;
      const conexao = socket = new WebSocket(location.origin.replace(/^http/,'ws') + '/ws/central?canal_id=' + encodeURIComponent(c.id) + '&token=' + encodeURIComponent(token));
      conexao.binaryType = 'arraybuffer';
      texto('status-voz','Conectando ao áudio…');
      conexao.onopen = () => { if (socket !== conexao) return; texto('status-voz','Ouvindo ' + c.nome + '. Aguardando a próxima mensagem…'); mostrarFila(); };
      conexao.onmessage = evento => {
        if (socket !== conexao) return;
        if (evento.data instanceof ArrayBuffer) {
          if (!recebendo) return;
          if (evento.data.byteLength > 60 || recebendo.pacotes.length >= 4500) { parar('Mensagem fora do formato esperado. Escuta encerrada.'); return; }
          recebendo.pacotes.push(new Uint8Array(evento.data));
          return;
        }
        let msg; try { msg = JSON.parse(evento.data); } catch { return; }
        if (msg.tipo === 'inicio_reproducao') {
          if (recebendo || fila.length + (tocando ? 1 : 0) >= 10) { parar('A fila da central atingiu o limite. O áudio pendente foi descartado; clique em Ouvir canal para retomar.'); return; }
          recebendo = {id:msg.dados.transmissao_id,nome:msg.dados.remetente_nome,pacotes:[],pronta:false};
          fila.push(recebendo); mostrarFila();
        } else if (msg.tipo === 'fim_reproducao' && recebendo?.id === msg.dados.transmissao_id) {
          recebendo.pronta = true; recebendo = null; mostrarFila(); reproduzir();
        }
      };
      conexao.onclose = () => { if (socket === conexao) parar('A conexão de voz caiu. O áudio pendente foi descartado. Clique em Ouvir canal para reconectar.'); };
      conexao.onerror = () => conexao.close();
      mostrarCanal(); mostrarFila();
    } catch (erro) { if (tentativa === versao) parar(erro.message || 'Não foi possível iniciar a escuta.'); }
  }
  async function reproduzir() {
    if (tocando || decodificando || pausado || !contexto || !fila[0]?.pronta) return;
    const atualVersao = versao, ctx = contexto, mensagem = fila[0];
    const partes = [];
    let taxa = 16000, falha, decoder;
    decodificando = true;
    try {
      decoder = new AudioDecoder({
        output: audio => {
          try {
            if (atualVersao !== versao) return;
            taxa = audio.sampleRate;
            const pcm = new Float32Array(audio.numberOfFrames);
            audio.copyTo(pcm,{planeIndex:0,format:'f32-planar'});
            partes.push(pcm);
          } finally { audio.close(); }
        },
        error: erro => { falha = erro; }
      });
      decoder.configure(configOpus);
      for (let i=0; i<mensagem.pacotes.length; i++) {
        decoder.decode(new EncodedAudioChunk({type:'key',timestamp:i*20000,duration:20000,data:mensagem.pacotes[i]}));
      }
      await decoder.flush();
      if (falha) throw falha;
      if (atualVersao !== versao) return;
      mensagem.pacotes = [];
      const tamanho = partes.reduce((total,p) => total+p.length,0);
      if (!tamanho) { fila.shift(); decodificando = false; mostrarFila(); reproduzir(); return; }
      const buffer = ctx.createBuffer(1,tamanho,taxa);
      let inicio = 0;
      for (const parte of partes) { buffer.copyToChannel(parte,0,inicio); inicio += parte.length; }
      fonte = ctx.createBufferSource(); fonte.buffer = buffer; fonte.connect(ctx.destination);
      fila.shift(); tocando = mensagem; decodificando = false;
      const tocador = fonte;
      tocador.onended = () => {
        if (atualVersao !== versao) return;
        tocador.disconnect(); tocador.buffer = null; fonte = null; tocando = null;
        texto('status-voz','Mensagem reproduzida. Aguardando a próxima…'); mostrarFila(); reproduzir();
      };
      texto('status-voz','Ouvindo mensagem de ' + mensagem.nome); mostrarFila();
      tocador.start();
    } catch { if (atualVersao === versao) parar('Não foi possível decodificar a voz. Confira se os celulares usam a versão atual do app.'); }
    finally { if (decoder && decoder.state !== 'closed') decoder.close(); partes.length = 0; }
  }
  async function pausar() {
    if (!contexto || !socket) return;
    try {
      pausado = !pausado;
      if (pausado) await contexto.suspend(); else { await contexto.resume(); reproduzir(); }
      texto('status-voz',pausado ? 'Reprodução pausada. As mensagens continuam chegando à fila.' : 'Reprodução retomada.'); mostrarFila();
    } catch { parar('Não foi possível controlar a reprodução. Inicie a escuta novamente.'); }
  }
  async function criarCanal(evento) {
    evento.preventDefault();
    el('salvar-canal').disabled = true;
    texto('status-form-canal','Salvando canal…');
    try {
      const dadosCanal = {nome:el('nome-canal').value.trim(),descricao:el('descricao-canal').value.trim(),tipo_acesso:editando?.tipo_acesso || 'privado',limite_participantes:Number(el('limite-canal').value),geocerca_ativa:el('limitar-area').checked};
      if (!dadosCanal.nome) throw new Error('Informe o nome do canal.');
      if (dadosCanal.geocerca_ativa) {
        if (!el('latitude-canal').value || !el('longitude-canal').value) throw new Error('Escolha o centro da área no mapa.');
        dadosCanal.centro_latitude = Number(el('latitude-canal').value);
        dadosCanal.centro_longitude = Number(el('longitude-canal').value);
        dadosCanal.raio_metros = Math.round(Number(el('raio-canal').value)*1000);
        if (dadosCanal.raio_metros < 1) throw new Error('Informe um raio maior que zero.');
      }
      const resposta = await fetch(editando ? '/canais/' + encodeURIComponent(editando.id) : '/canais',{method:editando ? 'PATCH' : 'POST',headers:{Authorization:'Bearer '+token,'Content-Type':'application/json'},body:JSON.stringify(dadosCanal)});
      const dados = await resposta.json();
      if (!resposta.ok) throw new Error(dados.mensagem || 'Não foi possível criar o canal.');
      fecharFormulario();
      await consultar();
      if (!socket) el('canal-central').value = dados.id;
      mostrarCanal(); texto('status-form-canal','');
    } catch (erro) { texto('status-form-canal',erro.message); }
    finally { el('salvar-canal').disabled = false; }
  }
  function mostrarArea() {
    el('campos-area').hidden = !el('limitar-area').checked;
    for (const id of ['raio-canal','latitude-canal','longitude-canal']) el(id).required = el('limitar-area').checked;
    if (circuloArea) { mapa.removeLayer(circuloArea); circuloArea = null; }
    if (!el('limitar-area').checked || !el('latitude-canal').value || !el('longitude-canal').value) return;
    const lat = Number(el('latitude-canal').value), lon = Number(el('longitude-canal').value), raio = Number(el('raio-canal').value)*1000;
    if (lat < -90 || lat > 90 || lon < -180 || lon > 180 || raio <= 0) return;
    circuloArea = L.circle([lat,lon],{radius:raio,color:'#229ed9',fillOpacity:0.12}).addTo(mapa);
  }
  function fecharFormulario() {
    editando = null; escolhendoCentro = false;
    el('form-canal').hidden = true; el('form-canal').reset();
    texto('aviso-centro','');
    if (circuloArea) { mapa.removeLayer(circuloArea); circuloArea = null; }
    mapa.getContainer().style.cursor = '';
  }
  function abrirFormulario(canal) {
    fecharFormulario(); editando = canal || null;
    el('form-canal').hidden = false;
    texto('titulo-form-canal',canal ? 'Editar canal' : 'Novo canal');
    texto('salvar-canal',canal ? 'Salvar alterações' : 'Criar canal');
    texto('status-form-canal','');
    el('nome-canal').value = canal?.nome || '';
    el('descricao-canal').value = canal?.descricao || '';
    el('limite-canal').value = canal?.limite_participantes || 10;
    el('limitar-area').checked = !!canal?.geocerca_ativa;
    const centro = mapa.getCenter();
    el('latitude-canal').value = canal?.centro_latitude ?? centro.lat.toFixed(6);
    el('longitude-canal').value = canal?.centro_longitude ?? centro.lng.toFixed(6);
    el('raio-canal').value = canal?.raio_metros ? canal.raio_metros/1000 : 5;
    mostrarArea(); el('nome-canal').focus();
  }
  function iniciar(jwt,empresaId,mapaDaCentral,aoAtualizar) {
    token = jwt; empresa = empresaId; mapa = mapaDaCentral; atualizarMapa = aoAtualizar;
    el('canal-central').onchange = mostrarCanal;
    el('ouvir-canal').onclick = ouvir; el('pausar-voz').onclick = pausar;
    el('parar-voz').onclick = () => parar();
    el('novo-canal').onclick = () => abrirFormulario();
    el('editar-canal').onclick = () => abrirFormulario(canalSelecionado());
    el('cancelar-canal').onclick = fecharFormulario;
    el('limitar-area').onchange = mostrarArea;
    for (const id of ['raio-canal','latitude-canal','longitude-canal']) el(id).oninput = mostrarArea;
    el('escolher-centro').onclick = () => {
      escolhendoCentro = true; mapa.getContainer().style.cursor = 'crosshair';
      texto('aviso-centro','Clique no mapa para definir o centro da área.');
    };
    mapa.on('click',evento => {
      if (!escolhendoCentro) return;
      escolhendoCentro = false; mapa.getContainer().style.cursor = '';
      el('latitude-canal').value = evento.latlng.lat.toFixed(6);
      el('longitude-canal').value = evento.latlng.lng.toFixed(6);
      texto('aviso-centro','Centro selecionado. Ajuste o raio e salve o canal.'); mostrarArea();
    });
    el('form-canal').onsubmit = criarCanal;
    consultar(); clearInterval(consulta); consulta = setInterval(consultar,3000);
  }
  function encerrar() { clearInterval(consulta); token = null; parar(); }
  window.addEventListener('pagehide',encerrar);
  return {iniciar,encerrar,canalDoUsuario};
})();

const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const codigo = fs.readFileSync(path.join(__dirname,'../painel/central.js'),'utf8');
async function preparar() {
  const elementos = new Map(), sockets = [], fontes = [], chamadas = [], contextos = [], circulos = [];
  function el(id) {
    if (!elementos.has(id)) elementos.set(id,{id,textContent:'',value:'',children:[],checked:false,hidden:true,disabled:false,
      focus(){},reset(){},replaceChildren(...itens){this.children=itens;if(id==='canal-central')this.value=itens[0]?.value || '';}});
    return elementos.get(id);
  }
  const canais = [{id:'canal-1',nome:'Operação',participantes:[{usuario_id:'motorista-1',nome:'Motorista'}],fila:[],pode_editar:true,limite_participantes:10,geocerca_ativa:false,tipo_acesso:'privado'}];
  class Socket { constructor(url){this.url=url;sockets.push(this);} close(){this.closed=true;this.onclose?.();} }
  class Audio {
    constructor(){this.state='running';contextos.push(this);}
    async resume(){this.state='running';} async suspend(){this.state='suspended';} async close(){this.state='closed';}
    createBuffer(canais,frames,taxa){const pcm=new Float32Array(frames);return {pcm,copyToChannel(parte,canal,inicio){pcm.set(parte,inicio);}};}
    createBufferSource(){const fonte={buffer:null,connect(){},disconnect(){},stop(){this.stopped=true;},start(){this.started=true;this.valor=this.buffer.pcm[0];}};fontes.push(fonte);return fonte;}
  }
  class Decoder {
    static async isConfigSupported(){return {supported:true};}
    constructor(callbacks){this.callbacks=callbacks;this.state='unconfigured';}
    configure(){this.state='configured';}
    decode(chunk){this.callbacks.output({sampleRate:16000,numberOfFrames:320,copyTo(pcm){pcm.fill(chunk.data[0]);},close(){}});}
    async flush(){} close(){this.state='closed';}
  }
  const mapa={getCenter:()=>({lat:-28.45,lng:-52.2}),getContainer:()=>({style:{}}),on(){},removeLayer(){}};
  const contexto=vm.createContext({document:{getElementById:el,createElement:()=>({textContent:'',value:''})},window:{isSecureContext:true,AudioDecoder:Decoder,addEventListener(){}},AudioDecoder:Decoder,EncodedAudioChunk:class{constructor(dados){Object.assign(this,dados);}},AudioContext:Audio,WebSocket:Socket,ArrayBuffer,Uint8Array,Float32Array,location:{origin:'http://localhost:8081'},L:{circle(centro,config){circulos.push({centro,config});return {addTo(){return this;}};}},setInterval:()=>1,clearInterval(){},fetch:async(url,opcoes)=>{chamadas.push({url,opcoes});return {ok:true,json:async()=>url==='/central/canais'?{canais}: {id:'canal-novo'}};}});
  vm.runInContext(codigo,contexto);
  const central=vm.runInContext('CentralVoz',contexto);
  central.iniciar('token','empresa',mapa,()=>{});
  const esperar=async()=>{for(let i=0;i<12;i++)await Promise.resolve();};
  await esperar();
  async function ouvir(){await el('ouvir-canal').onclick();sockets.at(-1).onopen();return sockets.at(-1);}
  function mensagem(socket,id,nome,valor){socket.onmessage({data:JSON.stringify({tipo:'inicio_reproducao',dados:{transmissao_id:id,remetente_nome:nome}})});socket.onmessage({data:new Uint8Array([valor]).buffer});socket.onmessage({data:JSON.stringify({tipo:'fim_reproducao',dados:{transmissao_id:id}})});}
  return {el,central,sockets,fontes,chamadas,contextos,circulos,esperar,ouvir,mensagem};
}
test('mostra presença real e toca duas mensagens em FIFO sem sobreposição',async()=>{
 const h=await preparar();assert.equal(h.central.canalDoUsuario('motorista-1'),'Operação');assert.equal(h.central.canalDoUsuario('ausente'),'Sem canal conectado');
 const socket=await h.ouvir();h.mensagem(socket,'a','A',1);h.mensagem(socket,'b','B',2);await h.esperar();
 assert.equal(h.fontes.length,1);assert.equal(h.fontes[0].valor,1);assert.match(h.el('fila-central').children[1].textContent,/B.*aguardando/);
 h.fontes[0].onended();await h.esperar();assert.equal(h.fontes.length,2);assert.equal(h.fontes[1].valor,2);assert.equal(h.fontes[0].buffer,null);
 h.fontes[1].onended();assert.equal(h.el('fila-central').children[0].textContent,'Nenhuma mensagem recebida.');h.central.encerrar();
});
test('pausa conserva a fila e parar libera os áudios e a conexão',async()=>{
 const h=await preparar(),socket=await h.ouvir();await h.el('pausar-voz').onclick();h.mensagem(socket,'a','A',1);await h.esperar();assert.equal(h.fontes.length,0);
 await h.el('pausar-voz').onclick();await h.esperar();assert.equal(h.fontes.length,1);h.el('parar-voz').onclick();assert.equal(h.fontes[0].buffer,null);assert.equal(socket.closed,true);assert.equal(h.contextos[0].state,'closed');assert.equal(h.el('canal-central').disabled,false);
});
test('fila da central tem limite e a queda interrompe a escuta',async()=>{
 const h=await preparar(),socket=await h.ouvir();await h.el('pausar-voz').onclick();for(let i=0;i<11;i++)h.mensagem(socket,String(i),'Motorista',i);assert.match(h.el('status-voz').textContent,/atingiu o limite/);assert.equal(socket.closed,true);
 const novo=await h.ouvir();novo.onclose();assert.match(h.el('status-voz').textContent,/conexão de voz caiu/);assert.equal(h.el('fila-central').children[0].textContent,'Nenhuma mensagem recebida.');
});
test('criação e edição enviam centro e raio em metros com autorização',async()=>{
 const h=await preparar();h.el('novo-canal').onclick();h.el('nome-canal').value='Novo';h.el('limite-canal').value='15';h.el('limitar-area').checked=true;h.el('raio-canal').value='3.5';h.el('latitude-canal').value='-28.45';h.el('longitude-canal').value='-52.2';h.el('limitar-area').onchange();assert.equal(h.circulos.at(-1).config.radius,3500);
 await h.el('form-canal').onsubmit({preventDefault(){}});const criada=h.chamadas.find(c=>c.opcoes?.method==='POST');const corpo=JSON.parse(criada.opcoes.body);assert.equal(corpo.raio_metros,3500);assert.equal(corpo.geocerca_ativa,true);assert.equal(corpo.limite_participantes,15);assert.equal(criada.opcoes.headers.Authorization,'Bearer token');
 h.el('canal-central').value='canal-1';h.el('editar-canal').onclick();h.el('nome-canal').value='Editado';await h.el('form-canal').onsubmit({preventDefault(){}});assert.equal(h.chamadas.find(c=>c.opcoes?.method==='PATCH').url,'/canais/canal-1');h.central.encerrar();
});

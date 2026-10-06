const test=require('node:test'),assert=require('node:assert/strict'),vm=require('node:vm'),fs=require('node:fs'),path=require('node:path');
const codigo=fs.readFileSync(path.join(__dirname,'../painel/contas.js'),'utf8');
async function preparar(papel='admin'){
 const elementos=new Map(),chamadas=[];let conflito=false;
 class Elemento{
  constructor(id){this.id=id;this.value='';this.textContent='';this.children=[];this.checked=false;this.hidden=true;this.disabled=false;this.dataset={};this.validade='';}
  append(...itens){this.children.push(...itens);}replaceChildren(...itens){this.children=itens;if(this.id==='veiculo')this.value=itens[0]?.value||'';}get firstChild(){return this.children[0];}
  setCustomValidity(texto){this.validade=texto;}
  reportValidity(){return !['cpf','senha','confirmacao-senha'].some(id=>el(id).validade);}
  reset(){if(this.id==='form-conta'){for(const id of ['nome','sobrenome','cpf','senha','confirmacao-senha','veiculo-descricao','veiculo-placa'])el(id).value='';el('novo-veiculo').checked=false;el('mostrar-senhas').checked=false;}}
 }
 function el(id){if(!elementos.has(id))elementos.set(id,new Elemento(id));return elementos.get(id);}
 const contexto=vm.createContext({document:{getElementById:el,createElement:()=>new Elemento('')},localStorage:{getItem:()=> 'token-teste'},TextEncoder,fetch:async(caminho,opcoes)=>{
  chamadas.push({caminho,opcoes});let dados;if(caminho==='/me')dados={papel};else if(opcoes.method==='POST'){
   if(conflito)return{ok:false,status:409,json:async()=>({mensagem:'CPF já cadastrado.'})};
   dados=caminho==='/admin/motoristas'?{nome:'Ana Da Estrada',login:'52998224725',id:'m1'}:{id:'v2',descricao:'Scania',placa:'DEF1234'};
  }else if(caminho==='/admin/veiculos')dados={veiculos:[{id:'v1',descricao:'Volvo',placa:'ABC1234'}]};else dados={motoristas:[]};
  return {ok:true,json:async()=>dados};
 }});
 vm.runInContext(codigo,contexto);async function esperar(){for(let i=0;i<20;i++)await Promise.resolve();}await esperar();
 function preencher(){el('nome').value='Ana';el('sobrenome').value='Da Estrada';el('cpf').value='529.982.247-25';el('senha').value=el('confirmacao-senha').value='senha-teste-123';el('veiculo').value='v1';}
 return{el,chamadas,preencher,esperar,conflito:()=>{conflito=true;}};
}
test('cadastro com veículo existente envia campos corretos e limpa as senhas',async()=>{
 const h=await preparar();h.preencher();await h.el('form-conta').onsubmit({preventDefault(){}});const req=h.chamadas.find(c=>c.opcoes.method==='POST');const dados=JSON.parse(req.opcoes.body);
 assert.equal(dados.veiculo_id,'v1');assert.equal(dados.nome,'Ana');assert.equal(dados.sobrenome,'Da Estrada');assert.equal(dados.confirmacao_senha,dados.senha);assert.equal(dados.novo_veiculo,undefined);assert.equal(h.el('senha').value,'');assert.equal(h.el('confirmacao-senha').value,'');assert.match(h.el('estado-conta').textContent,/Conta de Ana.*criada/);
});
test('novo veículo é enviado junto da conta e senhas diferentes são recusadas',async()=>{
 const h=await preparar();h.preencher();h.el('confirmacao-senha').value='diferente';await h.el('form-conta').onsubmit({preventDefault(){}});assert.equal(h.chamadas.filter(c=>c.opcoes.method==='POST').length,0);
 h.el('confirmacao-senha').value=h.el('senha').value;h.el('novo-veiculo').checked=true;h.el('novo-veiculo').onchange();assert.equal(h.el('veiculo').disabled,true);h.el('veiculo-descricao').value='Scania';h.el('veiculo-placa').value='DEF1234';await h.el('form-conta').onsubmit({preventDefault(){}});
 const dados=JSON.parse(h.chamadas.find(c=>c.opcoes.method==='POST').opcoes.body);assert.deepEqual(dados.novo_veiculo,{descricao:'Scania',placa:'DEF1234'});assert.equal(dados.veiculo_id,undefined);
});
test('CPF inválido impede envio e conflito não apresenta sucesso nem perde o formulário',async()=>{
 const h=await preparar();h.preencher();h.el('cpf').value='111.111.111-11';await h.el('form-conta').onsubmit({preventDefault(){}});assert.equal(h.chamadas.filter(c=>c.opcoes.method==='POST').length,0);
 h.el('cpf').value='529.982.247-25';h.conflito();await h.el('form-conta').onsubmit({preventDefault(){}});assert.equal(h.el('estado-conta').dataset.tipo,'erro');assert.equal(h.el('nome').value,'Ana');assert.equal(h.el('criar-conta').disabled,false);
});
test('conta de motorista não abre o formulário administrativo',async()=>{
 const h=await preparar('motorista');assert.equal(h.el('conteudo').hidden,true);assert.match(h.el('acesso').textContent,/exclusivo para administradores/);assert.equal(h.chamadas.length,1);
});

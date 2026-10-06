/* Dados da conta e senhas ficam somente no formulário e na requisição.
   O servidor armazena apenas o hash da senha; a confirmação não é persistida. */
'use strict';
const Cadastros = (() => {
 const el=id=>document.getElementById(id);
 let token,ocupado=false,criandoVeiculo=false,carregando=false;
 function estado(id,mensagem,tipo=''){el(id).textContent=mensagem;el(id).dataset.tipo=tipo;}
 function formatarCPF(valor){const n=valor.replace(/\D/g,'').slice(0,11);return n.length<=3?n:n.length<=6?n.slice(0,3)+'.'+n.slice(3):n.length<=9?n.slice(0,3)+'.'+n.slice(3,6)+'.'+n.slice(6):n.slice(0,3)+'.'+n.slice(3,6)+'.'+n.slice(6,9)+'-'+n.slice(9);}
 function cpfValido(valor){const n=valor.replace(/\D/g,'');if(n.length!==11 || /^(\d)\1{10}$/.test(n))return false;for(let tamanho=9;tamanho<=10;tamanho++){let soma=0;for(let i=0;i<tamanho;i++)soma+=Number(n[i])*(tamanho+1-i);let digito=soma*10%11;if(digito===10)digito=0;if(Number(n[tamanho])!==digito)return false;}return true;}
 async function api(caminho,dados){
  const resposta=await fetch(caminho,{method:dados?'POST':'GET',headers:{Authorization:'Bearer '+token,...(dados?{'Content-Type':'application/json'}:{})},cache:'no-store',...(dados?{body:JSON.stringify(dados)}:{})});
  let corpo;try{corpo=await resposta.json();}catch{throw new Error('Não foi possível ler a resposta da central.');}
  if(!resposta.ok){if(resposta.status===401)throw new Error('Sua sessão expirou. Volte à central e entre novamente.');if(resposta.status===403)throw new Error('Apenas administradores podem acessar este cadastro.');throw new Error(corpo.mensagem||'Não foi possível concluir a operação.');}
  return corpo;
 }
 function linha(textos){const tr=document.createElement('tr');for(const texto of textos){const td=document.createElement('td');td.textContent=texto;tr.append(td);}return tr;}
 async function atualizar(){
  if(carregando)return;carregando=true;el('atualizar-listas').disabled=true;estado('estado-listas','Atualizando cadastros…');
  try{
   const [v,m]=await Promise.all([api('/admin/veiculos'),api('/admin/motoristas')]);
   const anterior=el('veiculo').value;const vazio=document.createElement('option');vazio.value='';vazio.textContent=v.veiculos.length?'Selecione o veículo':'Cadastre um veículo abaixo';
   const opcoes=v.veiculos.map(veiculo=>{const o=document.createElement('option');o.value=veiculo.id;o.textContent=veiculo.descricao+' · '+veiculo.placa;return o;});el('veiculo').replaceChildren(vazio,...opcoes);if(v.veiculos.some(x=>x.id===anterior))el('veiculo').value=anterior;
   const itens=v.veiculos.map(veiculo=>{const li=document.createElement('li'),nome=document.createElement('strong'),placa=document.createElement('span');nome.textContent=veiculo.descricao;placa.textContent=veiculo.placa;li.append(nome,placa);return li;});
   if(!itens.length){const li=document.createElement('li');li.textContent='Nenhum veículo cadastrado. Cadastre o primeiro junto à conta ou pelo formulário acima.';itens.push(li);}el('lista-veiculos').replaceChildren(...itens);
   el('total-veiculos').textContent=v.veiculos.length+' veículo(s)';el('total-motoristas').textContent=m.motoristas.length+' conta(s)';
   const registros=m.motoristas.map(p=>linha([p.nome,p.cpf?formatarCPF(p.cpf):'Não informado',p.login,p.veiculo?p.veiculo.descricao+' · '+p.veiculo.placa:'Não informado']));
   if(!registros.length){const tr=linha(['Nenhum caminhoneiro cadastrado.']);tr.firstChild.colSpan=4;registros.push(tr);}el('lista-motoristas').replaceChildren(...registros);estado('estado-listas','');
  }catch(erro){estado('estado-listas',erro.message,'erro');}
  finally{carregando=false;el('atualizar-listas').disabled=false;}
 }
 function mostrarVeiculoNovo(){const novo=el('novo-veiculo').checked;el('novo-veiculo-campos').hidden=!novo;el('veiculo').disabled=novo;el('veiculo').required=!novo;el('veiculo-descricao').required=novo;el('veiculo-placa').required=novo;}
 function limpar(){el('form-conta').reset();el('senha').type=el('confirmacao-senha').type='password';for(const id of ['cpf','senha','confirmacao-senha'])el(id).setCustomValidity('');mostrarVeiculoNovo();}
 async function criarConta(evento){
  evento.preventDefault();if(ocupado)return;
  el('cpf').setCustomValidity(cpfValido(el('cpf').value)?'':'Informe um CPF válido.');el('confirmacao-senha').setCustomValidity(el('senha').value===el('confirmacao-senha').value?'':'As senhas não coincidem.');el('senha').setCustomValidity(new TextEncoder().encode(el('senha').value).length>72?'A senha deve ter no máximo 72 bytes.':'');
  if(!el('form-conta').reportValidity())return;
  const dados={nome:el('nome').value.trim(),sobrenome:el('sobrenome').value.trim(),cpf:el('cpf').value,senha:el('senha').value,confirmacao_senha:el('confirmacao-senha').value};
  if(el('novo-veiculo').checked)dados.novo_veiculo={descricao:el('veiculo-descricao').value.trim(),placa:el('veiculo-placa').value};else dados.veiculo_id=el('veiculo').value;
  ocupado=true;el('campos-conta').disabled=true;el('criar-conta').disabled=el('limpar-conta').disabled=true;estado('estado-conta','Criando conta…');
  try{const criada=await api('/admin/motoristas',dados);limpar();estado('estado-conta','Conta de '+criada.nome+' criada. Login no app: CPF '+formatarCPF(criada.login)+'.','sucesso');await atualizar();}
  catch(erro){estado('estado-conta',erro.message,'erro');}
  finally{ocupado=false;el('campos-conta').disabled=false;el('criar-conta').disabled=el('limpar-conta').disabled=false;mostrarVeiculoNovo();}
 }
 async function criarVeiculo(evento){
  evento.preventDefault();if(criandoVeiculo || !el('form-veiculo').reportValidity())return;
  const dados={descricao:el('cadastro-descricao').value.trim(),placa:el('cadastro-placa').value};criandoVeiculo=true;el('campos-veiculo').disabled=true;el('salvar-veiculo').disabled=true;estado('estado-veiculo','Cadastrando veículo…');
  try{const v=await api('/admin/veiculos',dados);el('form-veiculo').reset();estado('estado-veiculo','Veículo cadastrado: '+v.descricao+' · '+v.placa+'.','sucesso');await atualizar();if(!ocupado)el('veiculo').value=v.id;}
  catch(erro){estado('estado-veiculo',erro.message,'erro');}
  finally{criandoVeiculo=false;el('campos-veiculo').disabled=false;el('salvar-veiculo').disabled=false;}
 }
 async function iniciar(){
  token=localStorage.getItem('radiopx_painel_token');if(!token){estado('acesso','Entre como administrador no painel da central para criar contas.','erro');return;}
  try{const me=await api('/me');if(me.papel!=='admin')throw new Error('Este cadastro é exclusivo para administradores.');estado('acesso','');el('conteudo').hidden=false;await atualizar();}
  catch(erro){estado('acesso',erro.message,'erro');return;}
  el('form-conta').onsubmit=criarConta;el('form-veiculo').onsubmit=criarVeiculo;el('novo-veiculo').onchange=mostrarVeiculoNovo;
  el('mostrar-senhas').onchange=()=>{el('senha').type=el('confirmacao-senha').type=el('mostrar-senhas').checked?'text':'password';};
  el('cpf').oninput=()=>{el('cpf').value=formatarCPF(el('cpf').value);el('cpf').setCustomValidity('');};
  el('senha').oninput=()=>{el('senha').setCustomValidity('');el('confirmacao-senha').setCustomValidity('');};el('confirmacao-senha').oninput=()=>el('confirmacao-senha').setCustomValidity('');
  el('limpar-conta').onclick=()=>{limpar();estado('estado-conta','');};el('atualizar-listas').onclick=atualizar;mostrarVeiculoNovo();
 }
 return {iniciar};
})();
Cadastros.iniciar();

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import { randomUUID } from 'node:crypto';
import ts from 'typescript';
const source=readFileSync(new URL('../src/features/payments/pages/PaginaPagoInterbancario.tsx',import.meta.url),'utf8');
const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
const account={idCuenta:'account-1',idCliente:'client-1',numeroCuenta:'123456789',estado:'ACTIVA',saldoCentavos:20000};
const profile={clienteId:'client-1',estado:'ACTIVO',kycStatus:'VERIFIED'};
function setup({simulation=false,latest=profile,fail=false}={}){
 const values=[[account],profile,false,'',false,false,undefined,'','account-1','accept'];let index=0;
 const writes=new Map(),storage=new Map(),calls=[];
 const react={useState:initial=>{const i=index++;return [values[i]??initial,v=>writes.set(i,v)];},useCallback:f=>f,useEffect:()=>{},useRef:v=>({current:v})};
 const service={perfil:async()=>latest,enviar:async(id,payload)=>{calls.push({id,payload});if(fail)throw new Error('network');return{interbankTransferId:id,status:'PROCESSING'};}};
 const jsx=(type,props)=>({type,props});const exports={};
 runInNewContext(code,{exports,require:name=>{
  if(name==='react')return react;if(name==='react/jsx-runtime')return{jsx,jsxs:jsx};
  if(name==='react-router-dom')return{Link:'a'};
  if(name.includes('ContextoAutenticacion'))return{useAutenticacion:()=>({usuario:profile})};
  if(name.includes('servicioCuentas'))return{servicioCuentas:{buscar:async()=>account}};
  if(name.includes('servicioInterbancario'))return{servicioInterbancario:service};
  if(name.includes('clienteApi'))return{ErrorApi:class extends Error{}};
  if(name.endsWith('.css'))return{};throw new Error(name);
 },sessionStorage:{setItem:(k,v)=>storage.set(k,v),removeItem:k=>storage.delete(k)},crypto:{randomUUID},FormData:class{constructor(value){this.value=value;}get(k){return this.value[k];}},Intl});
 const tree=exports.PaginaPagoInterbancario({simulacion:simulation});
 function forms(node){if(!node||typeof node!=='object')return[];return[...(node.type==='form'?[node]:[]),...[node.props?.children].flat(Infinity).flatMap(forms)];}
 const send=forms(tree)[0].props.onSubmit;
 const event={preventDefault(){},currentTarget:{monto:'10.50',destino:'OTHER-123',descripcion:'Prueba'}};
 return{send:()=>send(event),calls,storage,writes};
}
test('Un KYC que dejó de estar verificado bloquea el POST aunque la sesión lo mostrara aprobado',async()=>{
 const s=setup({latest:{...profile,kycStatus:'REJECTED'}});await s.send();assert.equal(s.calls.length,0);assert.equal(s.storage.size,0);assert.match(s.writes.get(3),/KYC verificado/);
});
test('Escenarios no envía saldo real al endpoint del banco pareja',async()=>{
 const s=setup({simulation:true});await s.send();assert.equal(s.calls.length,0);
});
test('Doble envío concurrente produce un único depósito con la cuenta numérica del cliente',async()=>{
 const s=setup();await Promise.all([s.send(),s.send()]);assert.equal(s.calls.length,1);assert.equal(s.calls[0].payload.originAccount,'123456789');assert.equal(s.calls[0].payload.destinationAccount,'OTHER-123');assert.equal(s.calls[0].payload.amount,'10.50');
});
test('Un POST incierto conserva su identificador y no genera reintentos automáticos',async()=>{
 const s=setup({fail:true});await s.send();assert.equal(s.calls.length,1);assert.equal([...s.storage.values()][0],s.calls[0].id);assert.equal(s.writes.get(6).status,'RESPUESTA_INCIERTA');
});

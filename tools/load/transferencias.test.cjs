const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function harness(overrides = {}, env = {}) {
  const fixture = {
    purpose: 'load-test', ready: true, runId: 'carga-test', baseUrl: 'https://bank.example', token: 'private',
    pairs: [{source: '00000000-0000-4000-8000-000000000001', destination: '00000000-0000-4000-8000-000000000002'}],
    ...overrides,
  };
  const calls = [], metrics = {};
  class Metric {
    constructor(name) { this.name = name; metrics[name] = []; }
    add(value) { metrics[this.name].push(value); }
  }
  const response = (status, value) => ({status, json: () => value});
  const context = {
    __ENV: {FIXTURE_FILE: 'private.json', CONFIRM_LOAD: 'dedicated-test-accounts', PROFILE: 'smoke', ...env},
    __VU: 1, __ITER: 0, open: () => JSON.stringify(fixture),
    Counter: Metric, Rate: Metric, Trend: Metric, sleep: () => {}, console: {log: () => {}},
    exec: { test: { abort: reason => { throw new Error(reason); } } },
    http: {
      get: (url) => {
        calls.push(['GET', url]);
        if (url.endsWith('/api/clientes/perfil')) return response(200, {
          correo: 'carga-test@load.bank.invalid', estado: 'ACTIVO', kycStatus: 'VERIFIED', rol: 'CLIENTE', clienteId: 'customer',
        });
        if (url.includes('/api/cuentas?')) return response(200, fixture.pairs.flatMap(p => [p.source, p.destination]).map(id => ({
          idCuenta: id, idCliente: 'customer', estado: 'ACTIVA', tipoCuenta: 'CORRIENTE',
          comisionTransaccionCentavos: 0, saldoReservadoCentavos: 0, saldoDisponibleCentavos: 100000,
        })));
        return response(200, {estado: 'COMPLETADA'});
      },
      post: (url, body) => { calls.push(['POST', url, JSON.parse(body)]); return response(202, {operationId: 'operation'}); },
    },
  };
  const code = fs.readFileSync(__dirname + '/transferencias.js', 'utf8')
    .replace(/^import .*;\n/gm, '')
    .replace('export const options', 'const options')
    .replace('export default function ()', 'function iteration()')
    .replace(/export function /g, 'function ') + '\nthis.api = {options, setup, iteration, handleSummary};';
  vm.createContext(context);
  vm.runInContext(code, context);
  return {context, calls, metrics, fixture, api: context.api};
}

test('three-iteration smoke and durable completion (not just 202)', () => {
  const h = harness();
  h.api.setup(); h.api.iteration();
  assert.equal(h.api.options.scenarios.transfers.iterations, 3);
  assert.equal(h.metrics.transfers_completed.length, 1);
  assert.ok(h.calls.some(([method, url]) => method === 'GET' && url.endsWith('/api/transferencias/operation')));
  assert.ok(h.calls.every(([,url]) => !url.includes('/operaciones') && !url.includes('/interbank')));
});
test('uncertain POST is not retried and stops load', () => {
  const h = harness(); let attempts = 0;
  h.context.http.post = () => { attempts++; return {status: 0, json: () => ({})}; };
  assert.throws(() => h.api.iteration(), /no reenviar/);
  assert.equal(attempts, 1);
  assert.equal(h.metrics.transfers_unresolved[0], 1);
});
test('terminal rejection cannot count as successful throughput', () => {
  const h = harness();
  h.context.http.get = () => ({status: 200, json: () => ({estado: 'RECHAZADA'})});
  assert.throws(() => h.api.iteration(), /Resultado no exitoso/);
  assert.equal(h.metrics.transfers_completed.length, 0);
});
test('hard per-VU limit sends no more writes', () => {
  const h = harness(); h.context.__ITER = 200; h.api.iteration();
  assert.equal(h.calls.length, 0);
});
test('alternate direction; no account sharing between VUs', () => {
  const h = harness(); h.context.__ITER = 1; h.api.iteration();
  const body = h.calls.find(([method]) => method === 'POST')[2];
  assert.equal(body.idCuentaOrigen, h.fixture.pairs[0].destination);
  assert.equal(body.montoCentavos, 1);
});
test('reject HTTP and duplicated account fixtures', () => {
  assert.throws(() => harness({baseUrl: 'http://bank.example'}), /HTTPS/);
  assert.throws(() => harness({pairs: [{source: 'a', destination: 'a'}]}), /distinto/);
});
test('summary does not export credentials or account IDs', () => {
  const h = harness();
  const result = h.api.handleSummary({metrics: {}, state: {testRunDurationMs: 100}});
  assert.ok(!result.stdout.includes('private'));
  assert.ok(!result.stdout.includes(h.fixture.pairs[0].source));
});
test('capacity requires explicit opt-in and is bounded to 64 VUs / 1600 writes', () => {
  assert.throws(() => harness({}, {PROFILE:'capacity'}), /consentimiento adicional/);
  const pairs = Array.from({length:8}, (_,i) => ({
    source: `00000000-0000-4000-8000-${String(i*2+1).padStart(12,'0')}`,
    destination: `00000000-0000-4000-8000-${String(i*2+2).padStart(12,'0')}`,
  }));
  const h = harness({pairs}, {PROFILE:'capacity', CONFIRM_CAPACITY:'bounded-64'});
  const summary = JSON.parse(h.api.handleSummary({metrics:{},state:{testRunDurationMs:1}}).stdout);
  assert.equal(summary.maxVUs,64); assert.equal(summary.maxTransfers,1600);
  h.context.__ITER = 25; h.api.iteration(); assert.equal(h.calls.length,0);
});
test('no load on an unfinished fixture or a non-test customer', () => {
  assert.throws(() => harness({ready:false}), /fixture/);
  const h=harness();
  h.context.http.get=()=>({status:200,json:()=>({correo:'personal@example.com'})});
  assert.throws(()=>h.api.setup(),/exclusivo/);
});

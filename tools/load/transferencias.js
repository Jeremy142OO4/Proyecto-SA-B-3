// k6 v1.6.0. Sólo transferencias internas entre cuentas exclusivas de carga.
import http from 'k6/http';
import { sleep } from 'k6';
import exec from 'k6/execution';
import { Counter, Rate, Trend } from 'k6/metrics';

const fixture = JSON.parse(open(__ENV.FIXTURE_FILE));
const base = fixture.baseUrl;
const profile = __ENV.PROFILE || 'smoke';
if (__ENV.CONFIRM_LOAD !== 'dedicated-test-accounts' ||
    !/^https:\/\/[^/]+$/.test(base) || fixture.purpose !== 'load-test' || fixture.ready !== true ||
    !/^carga-[a-z0-9-]+$/.test(fixture.runId) || !fixture.token ||
    !Array.isArray(fixture.pairs) || !fixture.pairs.length || fixture.pairs.length > 20) {
  throw new Error('Se requieren HTTPS, consentimiento y fixture exclusivo válido.');
}
if (!['smoke', 'load', 'capacity'].includes(profile)) throw new Error('PROFILE debe ser smoke, load o capacity');
if (profile === 'capacity' && __ENV.CONFIRM_CAPACITY !== 'bounded-64') {
  throw new Error('Capacity requiere consentimiento adicional: CONFIRM_CAPACITY=bounded-64');
}
const ids = fixture.pairs.flatMap(p => [p.source, p.destination]);
if (new Set(ids).size !== ids.length || ids.some(id => !/^[a-f0-9-]{36}$/.test(id))) {
  throw new Error('Cada usuario necesita un par de cuentas distinto.');
}
const n = fixture.pairs.length;
const maxVUs = profile === 'smoke' ? 1 : profile === 'capacity' ? Math.min(n * 8, 64) : n;
const iterationLimit = Math.min(200, Math.floor(1600 / maxVUs));
const sent = new Counter('transfers_attempted');
const accepted = new Counter('transfers_accepted');
const completed = new Counter('transfers_completed');
const failed = new Rate('transfer_failure_rate');
const unresolved = new Counter('transfers_unresolved');
const latency = new Trend('transfer_end_to_end_ms', true);
const terminal = new Set(['COMPLETADA', 'RECHAZADA', 'COMPENSADA', 'COMPENSACION_FALLIDA']);
const headers = { Authorization: `Bearer ${fixture.token}`, 'Content-Type': 'application/json' };
const params = name => ({ headers, timeout: '15s', redirects: 0, tags: { name } });
const parse = response => { try { return response.json(); } catch (_) { return {}; } };
const accounts = payload => Array.isArray(payload) ? payload : payload.cuentas;

export const options = {
  scenarios: profile === 'smoke' ? {
    transfers: { executor: 'shared-iterations', vus: 1, iterations: 3, maxDuration: '3m' },
  } : {
    transfers: {
      executor: 'ramping-vus', startVUs: 1,
      stages: profile === 'capacity' ? [
        { duration: '30s', target: Math.min(8, maxVUs) },
        { duration: '60s', target: Math.min(16, maxVUs) },
        { duration: '60s', target: Math.min(32, maxVUs) },
        { duration: '60s', target: maxVUs },
        { duration: '180s', target: maxVUs },
        { duration: '30s', target: 0 },
      ] : [
        { duration: '30s', target: Math.min(2, n) },
        { duration: '60s', target: Math.min(4, n) },
        { duration: '60s', target: n },
        { duration: '180s', target: n },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '75s', gracefulStop: '75s',
    },
  },
  setupTimeout: '60s', teardownTimeout: '60s',
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
  systemTags: ['status', 'method', 'name', 'scenario', 'expected_response'],
  thresholds: {
    transfer_failure_rate: [{ threshold: 'rate==0', abortOnFail: true, delayAbortEval: '20s' }],
    transfers_completed: ['count>0'],
    transfers_unresolved: ['count==0'],
    'http_req_duration{name:POST /api/transferencias}': ['p(95)<500'],
    transfer_end_to_end_ms: ['p(95)<60000'],
  },
};

export function setup() {
  const who = http.get(`${base}/api/clientes/perfil`, params('GET /api/clientes/perfil'));
  const client = parse(who);
  if (who.status !== 200 || !String(client.correo).startsWith(`${fixture.runId}@`) ||
      !String(client.correo).endsWith('.invalid') || client.kycStatus !== 'VERIFIED' ||
      client.estado !== 'ACTIVO' || client.rol !== 'CLIENTE') {
    throw new Error('El cliente debe ser exclusivo de carga, activo y KYC verificado.');
  }
  const response = http.get(`${base}/api/cuentas?limite=100`, params('GET /api/cuentas'));
  const rows = accounts(parse(response));
  if (response.status !== 200 || !Array.isArray(rows)) throw new Error('No se pudieron validar las cuentas');
  for (const id of ids) {
    const a = rows.find(row => row.idCuenta === id);
    if (!a || a.idCliente !== client.clienteId || a.estado !== 'ACTIVA' ||
        a.tipoCuenta !== 'CORRIENTE' || a.comisionTransaccionCentavos !== 0 ||
        a.saldoReservadoCentavos !== 0 || a.saldoDisponibleCentavos < 1000) {
      throw new Error('Cuenta no apta: propiedad, tipo, estado, comisión, reservas o fondos');
    }
  }
  unresolved.add(0);
  console.log(`RUN ${fixture.runId} profile=${profile} pairs=${n}; sin BANK14 ni reintentos de POST`);
}

export default function () {
  if (__VU > maxVUs) { exec.test.abort('No aumentar VUs por encima del perfil autorizado'); return; }
  // Máximo global 1,600 operaciones de Q0.01 por perfil, incluso en capacity.
  if (__ITER >= iterationLimit) { sleep(1); return; }
  const pair = fixture.pairs[(__VU - 1) % n];
  const reverse = __ITER % 2 === 1;
  const started = Date.now();
  sent.add(1);
  const response = http.post(`${base}/api/transferencias`, JSON.stringify({
    idCuentaOrigen: reverse ? pair.destination : pair.source,
    idCuentaDestino: reverse ? pair.source : pair.destination,
    tipoCuentaDestino: 'CORRIENTE', montoCentavos: 1,
    descripcion: fixture.runId, resultadoExternoSimulado: 'EXITO',
  }), params('POST /api/transferencias'));
  const id = parse(response).operationId;
  if (response.status !== 202 || !id) {
    failed.add(true);
    unresolved.add(1); // Un timeout no demuestra que no se haya creado la operación.
    exec.test.abort(`POST no confirmado (${response.status}); no reenviar; conciliar por ${fixture.runId}`);
    return;
  }
  accepted.add(1);
  // Consulta durable: NO /operaciones, cuyo estado vive en memoria del Gateway.
  while (Date.now() - started < 60000) {
    sleep(0.5);
    const status = http.get(`${base}/api/transferencias/${id}`, params('GET /api/transferencias/:id'));
    const value = parse(status);
    if (status.status === 404) continue; // Consistencia eventual tras el 202.
    if (status.status !== 200) {
      failed.add(true); unresolved.add(1);
      exec.test.abort(`Consulta falló (${status.status}); pendiente de conciliar: ${id}`);
      return;
    }
    if (!terminal.has(value.estado)) continue;
    latency.add(Date.now() - started);
    failed.add(value.estado !== 'COMPLETADA');
    if (value.estado === 'COMPLETADA') completed.add(1);
    else exec.test.abort(`Resultado no exitoso: ${id} ${value.estado}`);
    sleep(0.1);
    return;
  }
  failed.add(true); unresolved.add(1);
  exec.test.abort(`Sin resultado terminal en 60s; conciliar sin reenviar: ${id}`);
}

export function handleSummary(data) {
  const result = { runId: fixture.runId, profile, maxVUs,
    maxTransfers: profile === 'smoke' ? 3 : iterationLimit * maxVUs, amountCentavos: 1,
    stateEndpoint: '/api/transferencias/:id', metrics: data.metrics,
    durationMs: data.state.testRunDurationMs };
  const path = __ENV.SUMMARY_FILE || 'summary.json';
  return { [path]: JSON.stringify(result, null, 2),
    stdout: JSON.stringify(result, null, 2) + '\n' };
}

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import ts from 'typescript';

const source = readFileSync(new URL('../src/hooks/usarConsulta.ts', import.meta.url), 'utf8');
const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
const flush = () => new Promise(resolve => setImmediate(resolve));

function mount(query, options) {
  const states = [], writes = [], effects = [], timers = new Map(), listeners = new Map();
  const document = { visibilityState: 'visible' };
  const react = {
    useState(initial) {
      const index = states.length;
      states.push(initial);
      return [initial, value => { states[index] = value; writes.push([index, value]); }];
    },
    useRef: current => ({ current }),
    useCallback: callback => callback,
    useEffect: callback => effects.push(callback),
  };
  const window = {
    setInterval: callback => { timers.set(1, callback); return 1; },
    clearInterval: id => timers.delete(id),
    addEventListener: (name, callback) => listeners.set(name, callback),
    removeEventListener: name => listeners.delete(name),
  };
  const exports = {};
  runInNewContext(compiled, { exports, Error, require: name => { assert.equal(name, 'react'); return react; }, window, document });
  const result = exports.usarConsulta(query, 15_000, options);
  const cleanup = effects.map(effect => effect());
  return { result, states, writes, timers, listeners, document, cleanup: () => cleanup.forEach(fn => fn?.()) };
}

test('Cuentas conserva los datos al actualizar, no escucha foco y pausa la pestaña oculta', async () => {
  let calls = 0;
  const view = mount(async () => ({ saldo: ++calls }), { recargaSilenciosa: true, actualizarAlEnfocar: false, pausarOculta: true });
  await flush();
  assert.equal(calls, 1);
  assert.equal(view.states[1], false);
  assert.equal(view.listeners.has('focus'), false);
  view.writes.length = 0;
  view.document.visibilityState = 'hidden';
  view.timers.get(1)();
  await flush();
  assert.equal(calls, 1);
  view.document.visibilityState = 'visible';
  view.timers.get(1)();
  await flush();
  assert.equal(calls, 2);
  assert.equal(view.states[0].saldo, 2);
  assert.equal(view.writes.some(([index, value]) => index === 1 && value === true), false);
  await view.result.recargar();
  assert.equal(calls, 3);
  view.cleanup();
  assert.equal(view.timers.size, 0);
});

test('Un error de actualización mantiene los saldos anteriores y permite reintentar', async () => {
  let fail = false;
  const view = mount(async () => { if (fail) throw new Error('Sin conexión'); return [100]; }, { recargaSilenciosa: true });
  await flush();
  fail = true;
  await view.result.recargar();
  assert.deepEqual(view.states[0], [100]);
  assert.equal(view.states[2], 'Sin conexión');
  assert.equal(view.states[1], false);
  fail = false;
  await view.result.recargar();
  assert.equal(view.states[2], '');
  view.cleanup();
});

test('Las pantallas sin opciones conservan el comportamiento previo por foco', async () => {
  const view = mount(async () => 1);
  await flush();
  assert.equal(view.listeners.has('focus'), true);
  view.writes.length = 0;
  view.listeners.get('focus')();
  await flush();
  assert.ok(view.writes.some(([index, value]) => index === 1 && value === true));
  view.cleanup();
  assert.equal(view.listeners.size, 0);
});

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

test('Swagger apunta al mismo despliegue sin guardar claves ni modificar el contrato', async () => {
  const original = { servers: [{ url: 'https://{bankHost}/interbank/v1' }], paths: {} };
  const spec = structuredClone(original);
  let options;
  const status = { textContent: '' };
  const context = {
    fetch: async () => ({ ok: true, json: async () => spec }),
    window: { location: { origin: 'https://bank03.test' } },
    SwaggerUIBundle: (value) => { options = value; },
    document: { getElementById: (id) => { assert.equal(id, 'api-status'); return status; } },
  };
  runInNewContext(readFileSync(new URL('../public/swagger/init.js', import.meta.url), 'utf8'), context);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(options.spec.servers[0].url, 'https://bank03.test/interbank/v1');
  assert.equal(options.persistAuthorization, false);
  assert.equal(options.validatorUrl, null);
  assert.equal(options.queryConfigEnabled, false);
  assert.equal(options.tryItOutEnabled, false);
  assert.match(status.textContent, /API de BANK03 disponible/);
  assert.equal(original.servers[0].url, 'https://{bankHost}/interbank/v1');
});

test('Swagger se muestra incluso si el health de BANK03 devuelve 503', async () => {
  let rendered = false;
  const status = { textContent: '' };
  const context = {
    fetch: async (url) => url.includes('/health')
      ? { ok: false }
      : { ok: true, json: async () => ({ paths: {} }) },
    window: { location: { origin: 'https://bank03.test' } },
    SwaggerUIBundle: () => { rendered = true; },
    document: { getElementById: () => status },
  };
  runInNewContext(readFileSync(new URL('../public/swagger/init.js', import.meta.url), 'utf8'), context);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(rendered, true);
  assert.match(status.textContent, /no disponible/);
});

test('La especificación generada conserva los ocho paths del contrato original', () => {
  const spec = JSON.parse(readFileSync(new URL('../public/swagger/generated/openapi.json', import.meta.url)));
  assert.equal(Object.keys(spec.paths).length, 8);
  assert.ok(spec.paths['/health']);
  assert.equal(spec.components.securitySchemes.ApiKeyAuth.name, 'X-Api-Key');
});

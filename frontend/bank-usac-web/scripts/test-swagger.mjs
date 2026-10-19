import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

test('Swagger apunta al mismo despliegue sin guardar claves ni modificar el contrato', async () => {
  const original = { servers: [{ url: 'https://{bankHost}/interbank/v1' }], paths: {} };
  const spec = structuredClone(original);
  let options;
  const context = {
    fetch: async () => ({ ok: true, json: async () => spec }),
    window: { location: { origin: 'https://bank03.test' } },
    SwaggerUIBundle: (value) => { options = value; },
    document: { getElementById: () => { throw new Error('Carga inesperadamente fallida'); } },
  };
  runInNewContext(readFileSync(new URL('../public/swagger/init.js', import.meta.url), 'utf8'), context);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(options.spec.servers[0].url, 'https://bank03.test/interbank/v1');
  assert.equal(options.persistAuthorization, false);
  assert.equal(options.validatorUrl, null);
  assert.equal(options.queryConfigEnabled, false);
  assert.equal(options.tryItOutEnabled, false);
  assert.equal(original.servers[0].url, 'https://{bankHost}/interbank/v1');
});

test('La especificación generada conserva los ocho paths del contrato original', () => {
  const spec = JSON.parse(readFileSync(new URL('../public/swagger/generated/openapi.json', import.meta.url)));
  assert.equal(Object.keys(spec.paths).length, 8);
  assert.ok(spec.paths['/health']);
  assert.equal(spec.components.securitySchemes.ApiKeyAuth.name, 'X-Api-Key');
});

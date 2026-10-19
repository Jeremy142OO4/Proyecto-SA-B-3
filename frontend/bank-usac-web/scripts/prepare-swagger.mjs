import { readFileSync, writeFileSync, mkdirSync, copyFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'yaml';

const require = createRequire(import.meta.url);
const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const original = readFileSync(resolve(frontend, '../../contracts/interbank/openapi.yaml'));
const expected = '4888b951aa48f95d8d301098c8aa642adfaa0e45a380975b8ce42acf3506df6a';
if (createHash('sha256').update(original).digest('hex') !== expected) {
  throw new Error('El contrato interbancario original no coincide con su SHA-256');
}
const destination = resolve(frontend, 'public/swagger/generated');
mkdirSync(destination, { recursive: true });
writeFileSync(resolve(destination, 'openapi.yaml'), original);
writeFileSync(resolve(destination, 'openapi.json'), JSON.stringify(parse(original.toString('utf8'))));
const assets = dirname(require.resolve('swagger-ui-dist/package.json'));
for (const asset of ['swagger-ui.css', 'swagger-ui-bundle.js', 'LICENSE', 'NOTICE', 'swagger-ui-bundle.js.LICENSE.txt']) {
  copyFileSync(resolve(assets, asset), resolve(destination, asset));
}
console.log('Swagger: contrato original verificado y recursos locales preparados');

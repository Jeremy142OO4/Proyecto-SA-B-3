/* La copia descargable permanece intacta; solo se configura el servidor de esta UI. */
fetch('/interbank/v1/health', { cache: 'no-store' })
  .then((response) => {
    document.getElementById('api-status').textContent = response.ok
      ? 'API de BANK03 disponible. No implica que la comunicación saliente con BANK14 esté configurada.'
      : 'API de BANK03 no disponible en este momento. La documentación permanece accesible.';
  })
  .catch(() => {
    document.getElementById('api-status').textContent = 'No se pudo consultar la disponibilidad de BANK03.';
  });
fetch('./generated/openapi.json', { cache: 'no-store' })
  .then((response) => {
    if (!response.ok) throw new Error('No se pudo cargar el contrato');
    return response.json();
  })
  .then((spec) => {
    spec.servers = [{ url: `${window.location.origin}/interbank/v1`, description: 'API pública de este despliegue' }];
    window.ui = SwaggerUIBundle({
      spec,
      dom_id: '#swagger-ui',
      deepLinking: true,
      persistAuthorization: false,
      validatorUrl: null,
      queryConfigEnabled: false,
      displayRequestDuration: true,
      tryItOutEnabled: false,
    });
  })
  .catch(() => {
    document.getElementById('swagger-ui').textContent = 'No se pudo cargar Swagger. Intentá recargar la página.';
  });

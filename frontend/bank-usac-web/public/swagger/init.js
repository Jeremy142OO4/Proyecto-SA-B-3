/* La copia descargable permanece intacta; solo se configura el servidor de esta UI. */
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

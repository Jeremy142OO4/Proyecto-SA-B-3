const URL = import.meta.env.VITE_API_URL ?? '/api';
const TIEMPO_ESPERA_MS = 25_000;
const MAX_INTENTOS_GET = 2;
const ESPERA_REINTENTO_MS = 500;

export class ErrorApi extends Error {
  constructor(public estado: number, mensaje: string) {
    super(mensaje);
    this.name = 'ErrorApi';
  }
}

function esMetodoGet(opciones: RequestInit) {
  return (opciones.method ?? 'GET').toUpperCase() === 'GET';
}

function esErrorTransitorio(error: unknown) {
  if (error instanceof ErrorApi) {
    return [408, 429, 502, 503, 504].includes(error.estado);
  }
  // Los errores de red tambien se pueden reintentar para consultas GET.
  return error instanceof TypeError;
}

function esperar(ms: number) {
  return new Promise<void>(resolve => globalThis.setTimeout(resolve, ms));
}

async function solicitar<T>(ruta: string, opciones: RequestInit = {}): Promise<T> {
  const token = sessionStorage.getItem('token');
  const encabezados = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...opciones.headers,
  };
  const puedeReintentar = esMetodoGet(opciones);
  const limite = Date.now() + TIEMPO_ESPERA_MS;
  const intentos = puedeReintentar ? MAX_INTENTOS_GET : 1;

  for (let intento = 1; intento <= intentos; intento += 1) {
    const restante = limite - Date.now();
    if (restante <= 0) {
      throw new ErrorApi(408, 'La solicitud tardó demasiado. Intenta nuevamente.');
    }
    const controlador = new AbortController();
    const temporizador = globalThis.setTimeout(() => controlador.abort(), restante);

    try {
      const respuesta = await fetch(`${URL}${ruta}`, {
        ...opciones,
        headers: encabezados,
        signal: controlador.signal,
      });
      const contenido = await respuesta.json().catch(() => ({}));

      if (!respuesta.ok) {
        const mensaje = respuesta.status === 408 || respuesta.status === 504
          ? 'El servicio está tardando en responder. Intenta nuevamente.'
          : contenido.mensaje ?? contenido.error ?? 'No fue posible completar la operación';
        throw new ErrorApi(respuesta.status, mensaje);
      }

      return contenido as T;
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        throw new ErrorApi(408, 'La solicitud tardó demasiado. Intenta nuevamente.');
      }
      if (!puedeReintentar || intento >= intentos || !esErrorTransitorio(error)) {
        throw error;
      }
      const espera = Math.min(ESPERA_REINTENTO_MS, Math.max(0, limite - Date.now()));
      if (espera > 0) await esperar(espera);
    } finally {
      globalThis.clearTimeout(temporizador);
    }
  }

  throw new ErrorApi(408, 'La solicitud tardó demasiado. Intenta nuevamente.');
}

export const clienteApi = {
  obtener: <T>(ruta: string) => solicitar<T>(ruta),
  publicar: <T>(ruta: string, datos: unknown) => solicitar<T>(ruta, {
    method: 'POST',
    body: JSON.stringify(datos),
  }),
  actualizar: <T>(ruta: string, datos: unknown) => solicitar<T>(ruta, {
    method: 'PUT',
    body: JSON.stringify(datos),
  }),
  actualizarParcial: <T>(ruta: string, datos: unknown) => solicitar<T>(ruta, {
    method: 'PATCH',
    body: JSON.stringify(datos),
  }),
};

const URL = import.meta.env.VITE_API_URL ?? '/api';
const TIEMPO_ESPERA_MS = 15_000;

export class ErrorApi extends Error {
  constructor(public estado: number, mensaje: string) {
    super(mensaje);
    this.name = 'ErrorApi';
  }
}

async function solicitar<T>(ruta: string, opciones: RequestInit = {}): Promise<T> {
  const token = sessionStorage.getItem('token');
  const controlador = new AbortController();
  const temporizador = globalThis.setTimeout(() => controlador.abort(), TIEMPO_ESPERA_MS);
  const encabezados = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...opciones.headers,
  };

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
    throw error;
  } finally {
    globalThis.clearTimeout(temporizador);
  }
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

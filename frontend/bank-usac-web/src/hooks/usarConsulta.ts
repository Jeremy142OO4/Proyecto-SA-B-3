/* eslint-disable react-hooks/rules-of-hooks -- El prefijo del hook se mantiene en espanol por convencion del proyecto. */
import { useCallback, useEffect, useRef, useState } from 'react';

/* Evita que StrictMode o una navegación rápida disparen consultas duplicadas. */
export function usarConsulta<T>(consulta: () => Promise<T>, intervaloMs?: number, opciones?: {
  recargaSilenciosa?: boolean;
  actualizarAlEnfocar?: boolean;
  pausarOculta?: boolean;
}) {
  const silenciosa = opciones?.recargaSilenciosa ?? false;
  const alEnfocar = opciones?.actualizarAlEnfocar ?? true;
  const pausarOculta = opciones?.pausarOculta ?? false;
  const [datos, setDatos] = useState<T>();
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const [actualizando, setActualizando] = useState(false);
  const tieneDatos = useRef(false);
  const solicitudEnCurso = useRef<Promise<T> | null>(null);

  const ejecutar = useCallback(async () => {
    if (solicitudEnCurso.current) return;

    setCargando(!silenciosa || !tieneDatos.current);
    setActualizando(true);
    setError('');
    const solicitud = Promise.resolve().then(consulta);
    solicitudEnCurso.current = solicitud;

    try {
      setDatos(await solicitud);
      tieneDatos.current = true;
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Error inesperado');
    } finally {
      solicitudEnCurso.current = null;
      setCargando(false);
      setActualizando(false);
    }
  }, [consulta, silenciosa]);

  useEffect(() => {
    void ejecutar();
  }, [ejecutar]);

  useEffect(() => {
    if (!intervaloMs) return;
    const actualizar = () => {
      if (pausarOculta && document.visibilityState === 'hidden') return;
      void ejecutar();
    };
    const timer = window.setInterval(actualizar, intervaloMs);
    if (alEnfocar) window.addEventListener('focus', actualizar);
    return () => { window.clearInterval(timer); if (alEnfocar) window.removeEventListener('focus', actualizar); };
  }, [ejecutar, intervaloMs, alEnfocar, pausarOculta]);

  return { datos, cargando, actualizando, error, recargar: ejecutar };
}

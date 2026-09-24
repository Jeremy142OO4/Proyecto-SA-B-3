/* eslint-disable react-hooks/rules-of-hooks -- El prefijo del hook se mantiene en espanol por convencion del proyecto. */
import { useCallback, useEffect, useRef, useState } from 'react';

/* Evita que StrictMode o una navegación rápida disparen consultas duplicadas. */
export function usarConsulta<T>(consulta: () => Promise<T>) {
  const [datos, setDatos] = useState<T>();
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const solicitudEnCurso = useRef<Promise<T> | null>(null);

  const ejecutar = useCallback(async () => {
    if (solicitudEnCurso.current) return;

    setCargando(true);
    setError('');
    const solicitud = Promise.resolve().then(consulta);
    solicitudEnCurso.current = solicitud;

    try {
      setDatos(await solicitud);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Error inesperado');
    } finally {
      solicitudEnCurso.current = null;
      setCargando(false);
    }
  }, [consulta]);

  useEffect(() => {
    void ejecutar();
  }, [ejecutar]);

  return { datos, cargando, error, recargar: ejecutar };
}

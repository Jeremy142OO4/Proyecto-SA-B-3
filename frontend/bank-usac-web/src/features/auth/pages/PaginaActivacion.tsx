import { useEffect, useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { servicioAuth } from '../services/servicioAuth';
import { ErrorApi } from '../../../services/clienteApi';

export function PaginaActivacion() {
  const [params] = useSearchParams();
  const token = params.get('token');
  const [cargando, setCargando] = useState(true);
  const [exito, setExito] = useState(false);
  const [mensaje, setMensaje] = useState('');

  useEffect(() => {
    if (!token) {
      setCargando(false);
      setMensaje('Enlace de activación inválido.');
      return;
    }
    servicioAuth.activar(token)
      .then((res) => { setExito(true); setMensaje(res.mensaje ?? 'Cuenta activada correctamente.'); })
      .catch((err) => setMensaje(err instanceof ErrorApi ? err.message : 'No se pudo activar la cuenta.'))
      .finally(() => setCargando(false));
  }, [token]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="max-w-sm text-center bg-slate-900 border border-slate-800 rounded-xl p-6">
        {cargando ? (
          <p className="text-slate-300">Validando cuenta...</p>
        ) : (
          <>
            <h2 className={`text-lg font-bold ${exito ? 'text-emerald-400' : 'text-rose-400'}`}>
              {exito ? 'Activación exitosa' : 'Error de activación'}
            </h2>
            <p className="text-slate-400 text-sm mt-2">{mensaje}</p>
            <Link to="/login" className="block mt-4 text-blue-400 text-sm">Ir a iniciar sesión</Link>
          </>
        )}
      </div>
    </div>
  );
}
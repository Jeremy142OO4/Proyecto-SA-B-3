import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { servicioAuth } from '../services/servicioAuth';
import { useAutenticacion } from '../context/ContextoAutenticacion';
import { ErrorApi } from '../../../services/clienteApi';

export function PaginaLogin() {
  const [usuario, setUsuario] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [cargando, setCargando] = useState(false);

  const { iniciarSesion } = useAutenticacion();
  const navegar = useNavigate();

  const enviar = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setCargando(true);
    try {
      const respuesta = await servicioAuth.login(usuario, password);
      iniciarSesion(respuesta.token, respuesta.cliente);
      navegar('/cuentas');
    } catch (err) {
      setError(err instanceof ErrorApi ? err.message : 'Error al iniciar sesión');
    } finally {
      setCargando(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="w-full max-w-sm bg-slate-900 border border-slate-800 rounded-xl p-6">
        <h1 className="text-xl font-bold text-white mb-4">Bank USAC - Iniciar sesión</h1>
        {error && <p className="text-rose-400 text-sm mb-3">{error}</p>}
        <form onSubmit={enviar} className="space-y-3">
          <input
            className="w-full bg-slate-800 text-white text-sm p-2 rounded"
            placeholder="Usuario"
            value={usuario}
            onChange={(e) => setUsuario(e.target.value)}
            required
          />
          <input
            type="password"
            className="w-full bg-slate-800 text-white text-sm p-2 rounded"
            placeholder="Contraseña"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <button
            disabled={cargando}
            className="w-full bg-blue-600 hover:bg-blue-700 text-white text-sm py-2 rounded disabled:opacity-50"
          >
            {cargando ? 'Ingresando...' : 'Ingresar'}
          </button>
        </form>
        <p className="text-slate-400 text-sm mt-4 text-center">
          ¿No tienes cuenta? <Link to="/registro" className="text-blue-400">Regístrate</Link>
        </p>
      </div>
    </div>
  );
}
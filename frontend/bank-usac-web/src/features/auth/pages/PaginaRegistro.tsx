import { useState } from 'react';
import { Link } from 'react-router-dom';
import { servicioAuth } from '../services/servicioAuth';
import { ErrorApi } from '../../../services/clienteApi';

export function PaginaRegistro() {
  const [form, setForm] = useState({
    nombres: '', apellidos: '', documento: '', fotoDocumentoUrl: '',
    correo: '', fechaNacimiento: '', direccion: '', password: '',
  });
  const [error, setError] = useState<string | null>(null);
  const [exito, setExito] = useState(false);
  const [cargando, setCargando] = useState(false);

  const cambiar = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm({ ...form, [e.target.name]: e.target.value });

  const enviar = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setCargando(true);
    try {
      await servicioAuth.registrar(form);
      setExito(true);
    } catch (err) {
      setError(err instanceof ErrorApi ? err.message : 'Error al registrar');
    } finally {
      setCargando(false);
    }
  };

  if (exito) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950">
        <div className="max-w-sm text-center bg-slate-900 border border-slate-800 rounded-xl p-6">
          <h2 className="text-white text-lg font-bold">¡Registro completado!</h2>
          <p className="text-slate-400 text-sm mt-2">Revisa tu correo ({form.correo}) para activar tu cuenta.</p>
          <Link to="/login" className="block mt-4 text-blue-400 text-sm">Ir a iniciar sesión</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950 py-10">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-xl p-6">
        <h1 className="text-xl font-bold text-white mb-4">Crear cuenta</h1>
        {error && <p className="text-rose-400 text-sm mb-3">{error}</p>}
        <form onSubmit={enviar} className="space-y-3">
          <input name="nombres" required placeholder="Nombres" value={form.nombres} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <input name="apellidos" required placeholder="Apellidos" value={form.apellidos} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <input name="documento" required placeholder="DPI" value={form.documento} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <input type="date" name="fechaNacimiento" required value={form.fechaNacimiento} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <input type="email" name="correo" required placeholder="Correo" value={form.correo} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <input name="fotoDocumentoUrl" placeholder="URL foto documento" value={form.fotoDocumentoUrl} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <textarea name="direccion" required placeholder="Dirección" value={form.direccion} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <input type="password" name="password" required placeholder="Contraseña" value={form.password} onChange={cambiar} className="w-full bg-slate-800 text-white text-sm p-2 rounded" />
          <button disabled={cargando} className="w-full bg-blue-600 hover:bg-blue-700 text-white text-sm py-2 rounded disabled:opacity-50">
            {cargando ? 'Procesando...' : 'Registrarme'}
          </button>
        </form>
        <p className="text-slate-400 text-sm mt-4 text-center">
          ¿Ya tienes cuenta? <Link to="/login" className="text-blue-400">Iniciar sesión</Link>
        </p>
      </div>
    </div>
  );
}
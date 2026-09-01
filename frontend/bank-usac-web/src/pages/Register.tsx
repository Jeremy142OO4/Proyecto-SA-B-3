import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../services/api';
import { Landmark, CheckCircle, AlertCircle } from 'lucide-react';

export const Register: React.FC = () => {
  const [formData, setFormData] = useState({
    firstName: '',
    lastName: '',
    documentId: '',
    documentPhotoUrl: '',
    email: '',
    birthDate: '',
    address: '',
    password: '',
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await api.post('/api/v1/customers/register', formData);
      setSuccess(true);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Error al completar el registro.');
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="min-h-screen bg-slate-950 flex flex-col justify-center items-center px-4">
        <div className="max-w-md w-full bg-slate-900 border border-slate-800 rounded-xl p-8 text-center shadow-2xl">
          <CheckCircle className="mx-auto h-16 w-16 text-emerald-500 mb-4" />
          <h2 className="text-2xl font-bold text-white">¡Registro completado!</h2>
          <p className="mt-3 text-sm text-slate-300">
            Hemos enviado un enlace de activación a tu correo electrónico: <strong>{formData.email}</strong>.
          </p>
          <p className="mt-2 text-xs text-slate-400">
            Por favor, revisa tu bandeja de entrada para activar tu cuenta antes de iniciar sesión.
          </p>
          <Link
            to="/login"
            className="mt-6 block w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition"
          >
            Ir al Inicio de Sesión
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-xl mx-auto">
        <div className="text-center mb-8">
          <Landmark className="mx-auto h-12 w-12 text-blue-500" />
          <h2 className="mt-4 text-3xl font-extrabold text-white">Crea tu cuenta en Bank USAC</h2>
          <p className="mt-2 text-sm text-slate-400">Completa tus datos personales para abrir tu cuenta</p>
        </div>

        <div className="bg-slate-900 py-8 px-6 shadow-2xl rounded-xl border border-slate-800">
          {error && (
            <div className="mb-6 bg-rose-500/10 border border-rose-500/30 text-rose-400 p-3 rounded-lg flex items-center space-x-2 text-sm">
              <AlertCircle className="h-5 w-5 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <form className="space-y-4" onSubmit={handleSubmit}>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-300">Nombres</label>
                <input
                  type="text"
                  name="firstName"
                  required
                  value={formData.firstName}
                  onChange={handleChange}
                  className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  placeholder="Juan Carlos"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-300">Apellidos</label>
                <input
                  type="text"
                  name="lastName"
                  required
                  value={formData.lastName}
                  onChange={handleChange}
                  className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  placeholder="Pérez Gómez"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-300">DPI / Documento</label>
                <input
                  type="text"
                  name="documentId"
                  required
                  value={formData.documentId}
                  onChange={handleChange}
                  className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  placeholder="2999123450101"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-300">Fecha de Nacimiento</label>
                <input
                  type="date"
                  name="birthDate"
                  required
                  value={formData.birthDate}
                  onChange={handleChange}
                  className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300">Correo Electrónico</label>
              <input
                type="email"
                name="email"
                required
                value={formData.email}
                onChange={handleChange}
                className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                placeholder="juan.perez@correo.com"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300">Foto del Documento (URL / Referencia)</label>
              <input
                type="text"
                name="documentPhotoUrl"
                value={formData.documentPhotoUrl}
                onChange={handleChange}
                className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                placeholder="https://storage.bankusac.gt/docs/dpi-123.jpg"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300">Dirección Residencial</label>
              <textarea
                name="address"
                required
                rows={2}
                value={formData.address}
                onChange={handleChange}
                className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                placeholder="Ciudad de Guatemala, Zona 12"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300">Contraseña</label>
              <input
                type="password"
                name="password"
                required
                value={formData.password}
                onChange={handleChange}
                className="mt-1 block w-full bg-slate-800 border border-slate-700 rounded-lg text-white text-sm p-2.5 focus:ring-2 focus:ring-blue-500 focus:outline-none"
                placeholder="••••••••"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="mt-6 w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg shadow-md transition disabled:opacity-50"
            >
              {loading ? 'Procesando registro...' : 'Completar Registro'}
            </button>
          </form>

          <div className="mt-6 text-center">
            <p className="text-sm text-slate-400">
              ¿Ya tienes cuenta?{' '}
              <Link to="/login" className="font-medium text-blue-400 hover:text-blue-300">
                Iniciar Sesión
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
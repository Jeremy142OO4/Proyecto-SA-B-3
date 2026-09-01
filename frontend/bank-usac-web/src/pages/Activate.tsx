import React, { useEffect, useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { api } from '../services/api';
import { CheckCircle2, XCircle, Loader2 } from 'lucide-react';

export const Activate: React.FC = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');

  const [loading, setLoading] = useState(true);
  const [success, setSuccess] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!token) {
      setLoading(false);
      setSuccess(false);
      setMessage('El enlace de activación no contiene un token válido.');
      return;
    }

    const activateAccount = async () => {
      try {
        const res = await api.get(`/api/v1/customers/activate?token=${token}`);
        setSuccess(true);
        setMessage(res.data.message || '¡Tu cuenta ha sido activada exitosamente!');
      } catch (err: any) {
        setSuccess(false);
        setMessage(err.response?.data?.error || 'No se pudo activar la cuenta. El enlace pudo haber expirado.');
      } finally {
        setLoading(false);
      }
    };

    activateAccount();
  }, [token]);

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col justify-center items-center px-4">
      <div className="max-w-md w-full bg-slate-900 border border-slate-800 rounded-xl p-8 text-center shadow-2xl">
        {loading ? (
          <div className="flex flex-col items-center">
            <Loader2 className="h-12 w-12 text-blue-500 animate-spin mb-4" />
            <h3 className="text-xl font-bold text-white">Validando tu cuenta...</h3>
            <p className="text-sm text-slate-400 mt-2">Por favor espera un momento.</p>
          </div>
        ) : success ? (
          <div className="flex flex-col items-center">
            <CheckCircle2 className="h-14 w-14 text-emerald-500 mb-4" />
            <h3 className="text-2xl font-bold text-white">¡Activación Exitosa!</h3>
            <p className="text-sm text-slate-300 mt-3">{message}</p>
            <Link
              to="/login"
              className="mt-6 w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition"
            >
              Iniciar Sesión Ahora
            </Link>
          </div>
        ) : (
          <div className="flex flex-col items-center">
            <XCircle className="h-14 w-14 text-rose-500 mb-4" />
            <h3 className="text-2xl font-bold text-white">Error de Activación</h3>
            <p className="text-sm text-slate-300 mt-3">{message}</p>
            <Link
              to="/login"
              className="mt-6 w-full py-2.5 px-4 bg-slate-800 hover:bg-slate-700 text-white text-sm font-medium rounded-lg transition"
            >
              Volver al Inicio
            </Link>
          </div>
        )}
      </div>
    </div>
  );
};
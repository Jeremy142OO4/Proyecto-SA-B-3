import React from 'react';
import { useAuth } from '../context/AuthContext';
import { Navbar } from '../components/Navbar';
import { Wallet, ArrowLeftRight, CreditCard, ShieldCheck } from 'lucide-react';

export const Dashboard: React.FC = () => {
  const { user } = useAuth();

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto py-10 px-4 sm:px-6 lg:px-8">
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 sm:p-8 mb-8 shadow-xl">
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center">
            <div>
              <h1 className="text-2xl sm:text-3xl font-bold text-white">
                Bienvenido, {user?.fullName}
              </h1>
              <p className="text-slate-400 mt-1">
                Usuario: <span className="text-blue-400 font-mono">@{user?.username}</span> | Rol: {user?.role}
              </p>
            </div>
            <div className="mt-4 sm:mt-0 flex items-center space-x-2 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 px-3 py-1.5 rounded-full text-xs font-semibold">
              <ShieldCheck className="h-4 w-4" />
              <span>Cuenta Activa</span>
            </div>
          </div>
        </div>

        {/* Accesos rápidos de los 5 microservicios */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-lg hover:border-slate-700 transition">
            <div className="bg-blue-600/20 text-blue-400 p-3 rounded-lg w-fit mb-4">
              <Wallet className="h-6 w-6" />
            </div>
            <h3 className="text-lg font-bold text-white">Mis Cuentas</h3>
            <p className="text-slate-400 text-sm mt-2">
              Consulta balances de cuentas monetarias y de ahorro gestionadas por <span className="text-slate-300 font-mono text-xs">account-service</span>.
            </p>
          </div>

          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-lg hover:border-slate-700 transition">
            <div className="bg-emerald-600/20 text-emerald-400 p-3 rounded-lg w-fit mb-4">
              <ArrowLeftRight className="h-6 w-6" />
            </div>
            <h3 className="text-lg font-bold text-white">Transferencias</h3>
            <p className="text-slate-400 text-sm mt-2">
              Realiza transferencias entre cuentas orquestadas con el patrón Saga por <span className="text-slate-300 font-mono text-xs">transaction-service</span>.
            </p>
          </div>

          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-lg hover:border-slate-700 transition">
            <div className="bg-amber-600/20 text-amber-400 p-3 rounded-lg w-fit mb-4">
              <CreditCard className="h-6 w-6" />
            </div>
            <h3 className="text-lg font-bold text-white">Pagos de Servicios</h3>
            <p className="text-slate-400 text-sm mt-2">
              Procesa pagos externos e internos procesados asíncronamente por <span className="text-slate-300 font-mono text-xs">payment-service</span>.
            </p>
          </div>
        </div>
      </main>
    </div>
  );
};
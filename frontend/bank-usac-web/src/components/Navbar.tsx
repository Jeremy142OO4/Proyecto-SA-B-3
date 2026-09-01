import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { Landmark, LogOut, User as UserIcon, ShieldAlert } from 'lucide-react';

export const Navbar: React.FC = () => {
  const { user, logout, hasRole } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <nav className="bg-slate-900 text-white shadow-lg border-b border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center space-x-3">
            <Landmark className="h-8 w-8 text-blue-500" />
            <Link to="/dashboard" className="font-bold text-xl tracking-wider">
              BANK <span className="text-blue-500">USAC</span>
            </Link>
          </div>

          <div className="flex items-center space-x-6">
            {hasRole(['ADMIN']) && (
              <Link
                to="/audit"
                className="flex items-center space-x-1 text-sm text-amber-400 hover:text-amber-300 font-medium transition"
              >
                <ShieldAlert className="h-4 w-4" />
                <span>Auditoría</span>
              </Link>
            )}

            <div className="flex items-center space-x-3 bg-slate-800 px-3 py-1.5 rounded-full border border-slate-700">
              <UserIcon className="h-4 w-4 text-slate-400" />
              <div className="text-sm">
                <span className="font-semibold text-slate-200">{user?.username}</span>
                <span className="ml-2 text-xs bg-blue-600/30 text-blue-400 px-2 py-0.5 rounded-full border border-blue-500/30">
                  {user?.role}
                </span>
              </div>
            </div>

            <button
              onClick={handleLogout}
              className="flex items-center space-x-1 text-sm text-rose-400 hover:text-rose-300 transition"
              title="Cerrar sesión"
            >
              <LogOut className="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
};
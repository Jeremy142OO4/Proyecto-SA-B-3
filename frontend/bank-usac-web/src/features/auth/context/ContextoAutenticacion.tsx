import React, { createContext, useContext, useState } from 'react';
import { Usuario, Rol } from '../types/auth';

interface ContextoAuth {
  usuario: Usuario | null;
  autenticado: boolean;
  iniciarSesion: (token: string, usuario: Usuario) => void;
  cerrarSesion: () => void;
  tieneRol: (roles: Rol[]) => boolean;
}

const ContextoAutenticacion = createContext<ContextoAuth | undefined>(undefined);

export const ProveedorAutenticacion: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [usuario, setUsuario] = useState<Usuario | null>(() => {
    const guardado = sessionStorage.getItem('usuario');
    return guardado ? JSON.parse(guardado) : null;
  });
  const [token, setToken] = useState<string | null>(() => sessionStorage.getItem('token'));

  const iniciarSesion = (nuevoToken: string, nuevoUsuario: Usuario) => {
    sessionStorage.setItem('token', nuevoToken);
    sessionStorage.setItem('usuario', JSON.stringify(nuevoUsuario));
    setToken(nuevoToken);
    setUsuario(nuevoUsuario);
  };

  const cerrarSesion = () => {
    sessionStorage.removeItem('token');
    sessionStorage.removeItem('usuario');
    setToken(null);
    setUsuario(null);
  };

  const tieneRol = (roles: Rol[]) => !!usuario && roles.includes(usuario.rol);

  return (
    <ContextoAutenticacion.Provider value={{ usuario, autenticado: !!token, iniciarSesion, cerrarSesion, tieneRol }}>
      {children}
    </ContextoAutenticacion.Provider>
  );
};

export const useAutenticacion = () => {
  const contexto = useContext(ContextoAutenticacion);
  if (!contexto) throw new Error('useAutenticacion debe usarse dentro de ProveedorAutenticacion');
  return contexto;
};
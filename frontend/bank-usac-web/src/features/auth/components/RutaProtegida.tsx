import { Navigate, Outlet } from 'react-router-dom';
import { useAutenticacion } from '../context/ContextoAutenticacion';
import { Rol } from '../types/auth';

export function RutaProtegida({ rolesPermitidos }: { rolesPermitidos?: Rol[] }) {
  const { autenticado, usuario } = useAutenticacion();

  if (!autenticado || !usuario) return <Navigate to="/login" replace />;
  if (rolesPermitidos && !rolesPermitidos.includes(usuario.rol)) return <Navigate to="/" replace />;

  return <Outlet />;
}

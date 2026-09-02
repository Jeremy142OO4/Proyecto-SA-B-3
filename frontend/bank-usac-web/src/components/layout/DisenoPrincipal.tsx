import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAutenticacion } from '../../features/auth/context/ContextoAutenticacion';

export function DisenoPrincipal() {
  const { usuario, cerrarSesion } = useAutenticacion();
  const navegar = useNavigate();
  function salir() { cerrarSesion(); navegar('/login'); }
  return <div className="aplicacion"><aside className="barra">
    <div className="marca"><span className="marca-escudo">U</span><div><strong>Bank USAC</strong><small>Banca distribuida</small></div></div>
    <nav>
      {usuario?.rol === 'CLIENTE' && <><NavLink to="/cuentas">Cuentas</NavLink><NavLink to="/transferencias">Transferencias</NavLink><NavLink to="/transferencias/nueva">Nueva transferencia</NavLink><NavLink to="/pagos">Pagos</NavLink><NavLink to="/pagos/nuevo">Nuevo pago</NavLink></>}
      {usuario?.rol === 'TELLER' && <><NavLink to="/cajero/clientes/nuevo">Registrar cliente</NavLink><NavLink to="/cajero/cuentas/nueva">Crear cuenta</NavLink></>}
      {usuario?.rol === 'ADMIN' && <><NavLink to="/administracion/clientes">Administrar clientes</NavLink><NavLink to="/auditoria">Auditoría</NavLink></>}
    </nav>
    <div className="usuario"><span>{usuario?.rol}</span><strong>{usuario?.nombreCompleto}</strong><button className="boton-salir" onClick={salir}>Cerrar sesión</button></div>
  </aside><main><header className="encabezado"><div><p>Portal bancario</p><h1>{usuario?.rol === 'CLIENTE' ? 'Mis finanzas' : usuario?.rol === 'TELLER' ? 'Operaciones de cajero' : 'Administración'}</h1></div><span className="entorno">Sesión protegida</span></header><section className="contenido"><Outlet /></section></main></div>;
}

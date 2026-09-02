import { Navigate, Route, Routes } from 'react-router-dom';
import { DisenoPrincipal } from '../components/layout/DisenoPrincipal';
import { PaginaCuentas } from '../features/accounts/pages/PaginaCuentas';
import { PaginaCrearCuentaCliente } from '../features/accounts/pages/PaginaCrearCuentaCliente';
import { PaginaDetalleCuenta } from '../features/accounts/pages/PaginaDetalleCuenta';
import { PaginaAdministracionClientes } from '../features/admin/pages/PaginaAdministracionClientes';
import { PaginaAuditoria } from '../features/audit/pages/PaginaAuditoria';
import { RutaProtegida } from '../features/auth/components/RutaProtegida';
import { useAutenticacion } from '../features/auth/context/ContextoAutenticacion';
import { PaginaActivacion } from '../features/auth/pages/PaginaActivacion';
import { PaginaLogin } from '../features/auth/pages/PaginaLogin';
import { PaginaRegistro } from '../features/auth/pages/PaginaRegistro';
import { PaginaNuevoPago } from '../features/payments/pages/PaginaNuevoPago';
import { PaginaPagos } from '../features/payments/pages/PaginaPagos';
import { PaginaDetalleTransferencia } from '../features/transfers/pages/PaginaDetalleTransferencia';
import { PaginaEstadoTransferencia } from '../features/transfers/pages/PaginaEstadoTransferencia';
import { PaginaNuevaTransferencia } from '../features/transfers/pages/PaginaNuevaTransferencia';
import { PaginaTransferencias } from '../features/transfers/pages/PaginaTransferencias';

function InicioPorRol() {
  const { usuario } = useAutenticacion();
  if (!usuario) return <Navigate to="/login" replace />;
  return <Navigate to={usuario.rol === 'CLIENTE' ? '/cuentas' : usuario.rol === 'TELLER' ? '/cajero/clientes/nuevo' : '/administracion/clientes'} replace />;
}

export function App() {
  return <Routes>
    <Route path="/login" element={<PaginaLogin />} />
    <Route path="/activar" element={<PaginaActivacion />} />
    <Route path="/" element={<InicioPorRol />} />

    <Route element={<RutaProtegida rolesPermitidos={['CLIENTE']} />}><Route element={<DisenoPrincipal />}>
      <Route path="/cuentas" element={<PaginaCuentas />} /><Route path="/cuentas/:idCuenta" element={<PaginaDetalleCuenta />} />
      <Route path="/pagos" element={<PaginaPagos />} /><Route path="/pagos/nuevo" element={<PaginaNuevoPago />} />
      <Route path="/transferencias" element={<PaginaTransferencias />} /><Route path="/transferencias/nueva" element={<PaginaNuevaTransferencia />} />
      <Route path="/transferencias/estado/:idOperacion" element={<PaginaEstadoTransferencia />} /><Route path="/transferencias/:idTransferencia" element={<PaginaDetalleTransferencia />} />
    </Route></Route>

    <Route element={<RutaProtegida rolesPermitidos={['TELLER']} />}><Route element={<DisenoPrincipal />}>
      <Route path="/cajero/clientes/nuevo" element={<PaginaRegistro />} /><Route path="/cajero/cuentas/nueva" element={<PaginaCrearCuentaCliente />} />
    </Route></Route>

    <Route element={<RutaProtegida rolesPermitidos={['ADMIN']} />}><Route element={<DisenoPrincipal />}>
      <Route path="/administracion/clientes" element={<PaginaAdministracionClientes />} /><Route path="/auditoria" element={<PaginaAuditoria />} />
    </Route></Route>
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes>;
}

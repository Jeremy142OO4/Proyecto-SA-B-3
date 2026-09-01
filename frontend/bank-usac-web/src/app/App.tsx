import { Navigate, Route, Routes } from 'react-router-dom';
import { DisenoPrincipal } from '../components/layout/DisenoPrincipal';
import { PaginaCuentas } from '../features/accounts/pages/PaginaCuentas';
import { PaginaDetalleCuenta } from '../features/accounts/pages/PaginaDetalleCuenta';
import { PaginaPagos } from '../features/payments/pages/PaginaPagos';
import { PaginaNuevoPago } from '../features/payments/pages/PaginaNuevoPago';
import { PaginaTransferencias } from '../features/transfers/pages/PaginaTransferencias';
import { PaginaNuevaTransferencia } from '../features/transfers/pages/PaginaNuevaTransferencia';
import { PaginaEstadoTransferencia } from '../features/transfers/pages/PaginaEstadoTransferencia';
import { PaginaDetalleTransferencia } from '../features/transfers/pages/PaginaDetalleTransferencia';

import { PaginaLogin } from '../features/auth/pages/PaginaLogin';
import { PaginaRegistro } from '../features/auth/pages/PaginaRegistro';
import { PaginaActivacion } from '../features/auth/pages/PaginaActivacion';
import { RutaProtegida } from '../features/auth/components/RutaProtegida';
import { PaginaAuditoria } from '../features/audit/pages/PaginaAuditoria';

export function App() {
  return (
    <Routes>
      {/* Rutas públicas */}
      <Route path="/login" element={<PaginaLogin />} />
      <Route path="/registro" element={<PaginaRegistro />} />
      <Route path="/activar" element={<PaginaActivacion />} />

      {/* Rutas protegidas existentes, ahora requieren sesión */}
      <Route element={<RutaProtegida />}>
        <Route element={<DisenoPrincipal />}>
          <Route index element={<Navigate to="/cuentas" replace />} />
          <Route path="/cuentas" element={<PaginaCuentas />} />
          <Route path="/cuentas/:idCuenta" element={<PaginaDetalleCuenta />} />
          <Route path="/pagos" element={<PaginaPagos />} />
          <Route path="/pagos/nuevo" element={<PaginaNuevoPago />} />
          <Route path="/transferencias" element={<PaginaTransferencias />} />
          <Route path="/transferencias/nueva" element={<PaginaNuevaTransferencia />} />
          <Route path="/transferencias/estado/:idOperacion" element={<PaginaEstadoTransferencia />} />
          <Route path="/transferencias/:idTransferencia" element={<PaginaDetalleTransferencia />} />
        </Route>
      </Route>

      {/* Solo ADMIN */}
      <Route element={<RutaProtegida rolesPermitidos={['ADMIN']} />}>
        <Route path="/auditoria" element={<PaginaAuditoria />} />
      </Route>
    </Routes>
  );
}
import { Navigate, Route, Routes } from 'react-router-dom';
import { DisenoPrincipal } from '../components/layout/DisenoPrincipal';
import { PaginaCuentas } from '../features/accounts/pages/PaginaCuentas';
import { PaginaDetalleCuenta } from '../features/accounts/pages/PaginaDetalleCuenta';
import { PaginaPagos } from '../features/payments/pages/PaginaPagos';
import { PaginaNuevoPago } from '../features/payments/pages/PaginaNuevoPago';
export function App(){return <Routes><Route element={<DisenoPrincipal/>}><Route index element={<Navigate to="/cuentas" replace/>}/><Route path="/cuentas" element={<PaginaCuentas/>}/><Route path="/cuentas/:idCuenta" element={<PaginaDetalleCuenta/>}/><Route path="/pagos" element={<PaginaPagos/>}/><Route path="/pagos/nuevo" element={<PaginaNuevoPago/>}/></Route></Routes>}

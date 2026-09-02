import { clienteApi } from '../../../services/clienteApi';
import type { Cuenta, MovimientoCuenta, OperacionAceptada } from '../types/cuenta';
export const servicioCuentas={listar:()=>clienteApi.obtener<Cuenta[]>('/cuentas'),buscar:(id:string)=>clienteApi.obtener<Cuenta>(`/cuentas/${id}`),movimientos:(id:string)=>clienteApi.obtener<MovimientoCuenta[]>(`/cuentas/${id}/movimientos`),crear:(datos:{tipoCuenta:string;idCliente:string})=>clienteApi.publicar<OperacionAceptada>('/cuentas',datos)};

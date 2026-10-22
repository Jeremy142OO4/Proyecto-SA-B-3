import { clienteApi } from '../../../services/clienteApi';
import type { Usuario } from '../../auth/types/auth';
export interface DepositoInterbancario { interbankTransferId:string; status:string; reasonCode?:string }
export const servicioInterbancario={
 perfil:()=>clienteApi.obtener<Usuario>('/clientes/perfil'),
 consultar:(id:string)=>clienteApi.obtener<DepositoInterbancario>(`/interbancarias/${id}`),
 enviar:(id:string,datos:{originAccount:string;destinationAccount:string;amount:string;description:string})=>clienteApi.publicar<DepositoInterbancario>('/interbancarias/',{interbankTransferId:id,currency:'GTQ',...datos},{'Idempotency-Key':id}),
};

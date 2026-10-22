import { clienteApi } from '../../../services/clienteApi';
import type { Usuario } from '../../auth/types/auth';
export interface DepositoInterbancario { interbankTransferId:string; status:string; reasonCode?:string; amount?:string; destinationAccount?:string; simulation?:boolean; createdAt?:string }
export const servicioInterbancario={
 listar:()=>clienteApi.obtener<DepositoInterbancario[]>('/interbancarias/'),
 perfil:()=>clienteApi.obtener<Usuario>('/clientes/perfil'),
 consultar:(id:string)=>clienteApi.obtener<DepositoInterbancario>(`/interbancarias/${id}`),
 enviar:(id:string,datos:{originAccount:string;destinationAccount:string;amount:string;description:string;scenario?:string})=>clienteApi.publicar<DepositoInterbancario>(datos.scenario?'/interbancarias/escenarios':'/interbancarias/',{interbankTransferId:id,currency:'GTQ',...datos},{'Idempotency-Key':id}),
};

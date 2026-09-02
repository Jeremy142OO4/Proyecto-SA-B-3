import { clienteApi } from '../../../services/clienteApi';
import type { EstadoCliente } from '../../auth/types/auth';
import type { ClienteAdministrado } from '../types/clienteAdministrado';

export const servicioAdministracion = {
  listarClientes: () => clienteApi.obtener<ClienteAdministrado[]>('/administracion/clientes'),
  cambiarEstado: (idCliente: string, estado: EstadoCliente) =>
    clienteApi.actualizarParcial<ClienteAdministrado>(`/administracion/clientes/${idCliente}/estado`, { estado }),
};

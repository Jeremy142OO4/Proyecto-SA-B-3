import type { EstadoCliente, Rol } from '../../auth/types/auth';

export interface ClienteAdministrado {
  customerId: string;
  fullName: string;
  email: string;
  username: string;
  role: Rol;
  status: EstadoCliente;
  createdAt: string;
}

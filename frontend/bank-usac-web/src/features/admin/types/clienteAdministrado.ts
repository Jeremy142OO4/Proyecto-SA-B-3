import type { EstadoCliente, Rol } from '../../auth/types/auth';

export interface ClienteAdministrado {
  customerId: string;
  fullName: string;
  documentId: string;
  email: string;
  username: string;
  role: Rol;
  status: EstadoCliente;
  kycStatus: EstadoKYC;
  createdAt: string;
}

export type EstadoKYC = 'PENDING' | 'VERIFIED' | 'REJECTED';

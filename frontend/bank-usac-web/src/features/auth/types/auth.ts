export type Rol = 'ADMIN' | 'TELLER' | 'CLIENTE';
export type EstadoCliente = 'PENDIENTE_ACTIVACION' | 'ACTIVO' | 'BLOQUEADO';
export type EstadoKYC = 'PENDING' | 'VERIFIED' | 'REJECTED';

export interface Usuario {
  clienteId: string;
  nombreCompleto: string;
  documento: string;
  correo: string;
  usuario: string;
  rol: Rol;
  estado: EstadoCliente;
  kycStatus: EstadoKYC;
}

export interface RespuestaLogin {
  token: string;
  cliente: Usuario;
}

export type Rol = 'ADMIN' | 'TELLER' | 'CLIENTE';
export type EstadoCliente = 'PENDIENTE_ACTIVACION' | 'ACTIVO' | 'BLOQUEADO';

export interface Usuario {
  clienteId: string;
  nombreCompleto: string;
  documento: string;
  correo: string;
  usuario: string;
  rol: Rol;
  estado: EstadoCliente;
}

export interface RespuestaLogin {
  token: string;
  cliente: Usuario;
}
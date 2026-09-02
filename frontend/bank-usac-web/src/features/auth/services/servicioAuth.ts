import { clienteApi } from '../../../services/clienteApi';
import { RespuestaLogin } from '../types/auth';

export const servicioAuth = {
  login: (usuario: string, password: string) =>
    clienteApi.publicar<RespuestaLogin>('/clientes/login', { usuario, password }),

  registrar: (datos: {
    nombres: string;
    apellidos: string;
    documento: string;
    fotoDocumentoUrl?: string;
    correo: string;
    fechaNacimiento: string;
    direccion: string;
    password: string;
  }) => clienteApi.publicar<{ mensaje: string }>('/clientes/registro', datos),

  activar: (token: string) =>
    clienteApi.obtener<{ mensaje: string }>(`/clientes/activacion?token=${token}`),
};

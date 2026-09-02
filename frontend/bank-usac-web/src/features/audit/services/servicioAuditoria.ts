import { clienteApi } from '../../../services/clienteApi';
import { RegistroAuditoria } from '../types/auditoria';

export const servicioAuditoria = {
  listar: (limite = 50) =>
    clienteApi.obtener<RegistroAuditoria[]>(`/auditoria/registros?limite=${limite}`),
};

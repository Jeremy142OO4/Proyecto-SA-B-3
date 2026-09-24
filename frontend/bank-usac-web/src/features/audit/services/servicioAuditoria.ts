import { clienteApi } from '../../../services/clienteApi';
import { RegistroAuditoria } from '../types/auditoria';

export const servicioAuditoria = {
  listar: async (limite = 50): Promise<RegistroAuditoria[]> => {
    const respuesta = await clienteApi.obtener<RegistroAuditoria[] | null>(`/auditoria/registros?limite=${limite}`);
    return Array.isArray(respuesta) ? respuesta : [];
  },
};

import { clienteApi } from '../../../services/clienteApi';
import type { Cuenta } from '../../accounts/types/cuenta';
import type { ClienteAdministrado } from '../types/clienteAdministrado';

export interface ConfiguracionMonitoreo {
  maxOperaciones: number;
  ventanaMinutos: number;
  version: number;
  origen: 'CONFIGMAP' | 'ADMINISTRACION';
}
export const servicioCuentasAdministracion = {
  listar: (pagina: number, cuenta = '') => clienteApi.obtener<{ cuentas: Cuenta[]; total: number }>(
    `/administracion/cuentas?limite=25&desplazamiento=${pagina * 25}${cuenta ? `&cuenta=${encodeURIComponent(cuenta)}` : ''}`,
  ),
  configuracion: () => clienteApi.obtener<ConfiguracionMonitoreo>('/administracion/cuentas/monitoreo'),
  configurar: (maxOperaciones: number, ventanaMinutos: number) =>
    clienteApi.actualizar<ConfiguracionMonitoreo>('/administracion/cuentas/monitoreo', { maxOperaciones, ventanaMinutos }),
  bloquear: (id: string, motivo: string) =>
    clienteApi.actualizarParcial<Cuenta>(`/administracion/cuentas/${id}/bloqueo`, { motivo }),
  propietarios: async () => {
    const clientes = new Map<string, ClienteAdministrado>();
    for (let desplazamiento = 0; ; desplazamiento += 100) {
      const lista = await clienteApi.obtener<ClienteAdministrado[]>(`/administracion/clientes?limite=100&desplazamiento=${desplazamiento}`);
      const anteriores = clientes.size;
      lista.forEach(cliente => clientes.set(cliente.customerId, cliente));
      if (lista.length < 100) return clientes;
      if (anteriores === clientes.size) throw new Error('No fue posible completar la consulta de propietarios');
    }
  },
};

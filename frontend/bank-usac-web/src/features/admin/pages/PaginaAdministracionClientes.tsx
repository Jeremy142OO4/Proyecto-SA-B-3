import { useEffect, useState } from 'react';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import type { EstadoCliente } from '../../auth/types/auth';
import { servicioAdministracion } from '../services/servicioAdministracion';
import type { ClienteAdministrado } from '../types/clienteAdministrado';

const estados: EstadoCliente[] = ['PENDIENTE_ACTIVACION', 'ACTIVO', 'BLOQUEADO'];

export function PaginaAdministracionClientes() {
  const [clientes, setClientes] = useState<ClienteAdministrado[]>([]);
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const [actualizando, setActualizando] = useState('');
  useEffect(() => { servicioAdministracion.listarClientes().then(setClientes).catch(e => setError(e instanceof Error ? e.message : 'No fue posible consultar los clientes')).finally(() => setCargando(false)); }, []);
  async function cambiar(cliente: ClienteAdministrado, estado: EstadoCliente) {
    setActualizando(cliente.customerId); setError('');
    try {
      const actualizado = await servicioAdministracion.cambiarEstado(cliente.customerId, estado);
      setClientes(lista => lista.map(item => item.customerId === actualizado.customerId ? actualizado : item));
    } catch (e) { setError(e instanceof Error ? e.message : 'No fue posible cambiar el estado'); }
    finally { setActualizando(''); }
  }
  if (cargando) return <EstadoCarga />;
  if (error && !clientes.length) return <EstadoError mensaje={error} />;
  return <>
    <div className="titulo-seccion"><div><p>Administración</p><h2>Clientes y usuarios</h2></div></div>
    {error && <div className="alerta error">{error}</div>}
    {!clientes.length ? <EstadoVacio mensaje="No hay clientes registrados." /> : <div className="lista tabla-clientes">
      {clientes.map(cliente => <article key={cliente.customerId}>
        <div><strong>{cliente.fullName}</strong><small>{cliente.username} · {cliente.email}</small><small>{cliente.role}</small></div>
        <label>Estado<select value={cliente.status} disabled={actualizando === cliente.customerId} onChange={e => void cambiar(cliente, e.target.value as EstadoCliente)}>{estados.map(estado => <option key={estado}>{estado}</option>)}</select></label>
      </article>)}
    </div>}
  </>;
}

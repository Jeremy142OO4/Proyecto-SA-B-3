import { useEffect, useState, type ChangeEvent } from 'react';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import type { EstadoCliente } from '../../auth/types/auth';
import { servicioAdministracion } from '../services/servicioAdministracion';
import type { ClienteAdministrado, EstadoKYC } from '../types/clienteAdministrado';
import './administracion.css';

const estados: EstadoCliente[] = ['PENDIENTE_ACTIVACION', 'ACTIVO', 'BLOQUEADO'];
const estadosKYC: EstadoKYC[] = ['PENDING', 'VERIFIED', 'REJECTED'];

export function PaginaAdministracionClientes() {
  const [clientes, setClientes] = useState<ClienteAdministrado[]>([]);
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const [actualizando, setActualizando] = useState('');
  const [copiado, setCopiado] = useState('');
  const [modal, setModal] = useState<{ titulo: string; mensaje: string } | null>(null);

  useEffect(() => {
    servicioAdministracion.listarClientes()
      .then(setClientes)
      .catch(e => setError(e instanceof Error ? e.message : 'No fue posible consultar los clientes'))
      .finally(() => setCargando(false));
  }, []);

  async function cambiar(cliente: ClienteAdministrado, estado: EstadoCliente) {
    setActualizando(cliente.customerId);
    setError('');
    try {
      const actualizado = await servicioAdministracion.cambiarEstado(cliente.customerId, estado);
      setClientes(lista => lista.map(item => item.customerId === actualizado.customerId ? actualizado : item));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'No fue posible cambiar el estado');
    } finally {
      setActualizando('');
    }
  }

  async function cambiarKYC(cliente: ClienteAdministrado, estadoKYC: EstadoKYC) {
    setActualizando(cliente.customerId);
    setError('');
    try {
      const actualizado = await servicioAdministracion.cambiarEstadoKYC(cliente.customerId, estadoKYC);
      setClientes(lista => lista.map(item => item.customerId === actualizado.customerId ? actualizado : item));
      setModal({
        titulo: 'Actualización realizada',
        mensaje: `El estado KYC de ${actualizado.fullName} se actualizó correctamente a ${actualizado.kycStatus}.`,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'No fue posible cambiar el estado KYC');
    } finally {
      setActualizando('');
    }
  }

  async function copiarDpi(cliente: ClienteAdministrado) {
    if (cliente.role !== 'CLIENTE' || !cliente.documentId) {
      setError('Solo se puede copiar el DPI de clientes con rol CLIENTE');
      return;
    }
    try {
      await navigator.clipboard.writeText(cliente.documentId);
      setCopiado(cliente.customerId);
      window.setTimeout(() => setCopiado(actual => actual === cliente.customerId ? '' : actual), 1800);
    } catch {
      setError('No fue posible copiar el DPI del cliente');
    }
  }

  if (cargando) return <EstadoCarga />;
  if (error && !clientes.length) return <EstadoError mensaje={error} />;

  return <>
    <div className="titulo-seccion">
      <div><p>Administración</p><h2>Clientes y usuarios</h2></div>
    </div>
    {error && <div className="alerta error">{error}</div>}
    {!clientes.length ? <EstadoVacio mensaje="No hay clientes registrados." /> : <div className="lista tabla-clientes">
      {clientes.map(cliente => <article key={cliente.customerId}>
        <div>
          <strong>{cliente.fullName}</strong>
          <small>{cliente.username} · {cliente.email}</small>
          <small>{cliente.role}</small>
          <small>ID del cliente: {cliente.customerId}</small>
          {cliente.role === 'CLIENTE' ? <>
            <small>DPI: {cliente.documentId || 'No disponible'}</small>
            <button type="button" className="secundario" disabled={!cliente.documentId} onClick={() => void copiarDpi(cliente)}>
              {copiado === cliente.customerId ? 'DPI copiado' : 'Copiar DPI'}
            </button>
          </> : <small className="dato-restringido">Copia de DPI no disponible para este rol</small>}
        </div>
        <label>Estado<select value={cliente.status} disabled={actualizando === cliente.customerId} onChange={(e: ChangeEvent<HTMLSelectElement>) => void cambiar(cliente, e.target.value as EstadoCliente)}>
          {estados.map(estado => <option key={estado}>{estado}</option>)}
        </select></label>
        <label>Estado KYC<select value={cliente.kycStatus} disabled={actualizando === cliente.customerId} onChange={(e: ChangeEvent<HTMLSelectElement>) => void cambiarKYC(cliente, e.target.value as EstadoKYC)}>
          {estadosKYC.map(estado => <option key={estado}>{estado}</option>)}
        </select></label>
      </article>)}
    </div>}
    {modal && <div className="modal-fondo" role="presentation" onClick={() => setModal(null)}>
      <section className="modal-confirmacion" role="dialog" aria-modal="true" aria-labelledby="modal-titulo" onClick={e => e.stopPropagation()}>
        <span className="modal-icono" aria-hidden="true">✓</span>
        <h3 id="modal-titulo">{modal.titulo}</h3>
        <p>{modal.mensaje}</p>
        <button type="button" onClick={() => setModal(null)}>Aceptar</button>
      </section>
    </div>}
  </>;
}

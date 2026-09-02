import { useEffect, useState } from 'react';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { servicioAuditoria } from '../services/servicioAuditoria';
import type { RegistroAuditoria } from '../types/auditoria';

export function PaginaAuditoria() {
  const [registros, setRegistros] = useState<RegistroAuditoria[]>([]);
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  useEffect(() => { servicioAuditoria.listar().then(setRegistros).catch(e => setError(e instanceof Error ? e.message : 'No fue posible consultar la auditoría')).finally(() => setCargando(false)); }, []);
  if (cargando) return <EstadoCarga />;
  if (error) return <EstadoError mensaje={error} />;
  return <><div className="titulo-seccion"><div><p>Trazabilidad distribuida</p><h2>Auditoría</h2></div></div>
    {!registros.length ? <EstadoVacio mensaje="No hay registros de auditoría." /> : <div className="lista">{registros.map(registro => <article key={registro.id}>
      <div><strong>{registro.eventType}</strong><small>{registro.producer} · {new Date(registro.occurredAt).toLocaleString('es-GT')}</small></div>
      <small>{registro.correlationId}</small>
    </article>)}</div>}
  </>;
}

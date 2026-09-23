import { useEffect, useMemo, useState } from 'react';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { servicioAuditoria } from '../services/servicioAuditoria';
import type { RegistroAuditoria } from '../types/auditoria';

type FiltroSeveridad = 'TODOS' | RegistroAuditoria['severity'];

const severidades: Array<{ valor: FiltroSeveridad; etiqueta: string }> = [
  { valor: 'TODOS', etiqueta: 'Todos' },
  { valor: 'INFO', etiqueta: 'Información' },
  { valor: 'WARNING', etiqueta: 'Advertencias' },
  { valor: 'ERROR', etiqueta: 'Errores' },
];

const iconosSeveridad: Record<RegistroAuditoria['severity'], string> = {
  INFO: 'ℹ️',
  WARNING: '⚠️',
  ERROR: '❌',
};

export function PaginaEventos() {
  const [registros, setRegistros] = useState<RegistroAuditoria[]>([]);
  const [filtro, setFiltro] = useState<FiltroSeveridad>('TODOS');
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    servicioAuditoria.listar(100)
      .then(setRegistros)
      .catch(e => setError(e instanceof Error ? e.message : 'No fue posible consultar los eventos'))
      .finally(() => setCargando(false));
  }, []);

  const visibles = useMemo(
    () => filtro === 'TODOS' ? registros : registros.filter(registro => registro.severity === filtro),
    [filtro, registros],
  );
  const totalPorSeveridad = (severidad: FiltroSeveridad) =>
    severidad === 'TODOS' ? registros.length : registros.filter(registro => registro.severity === severidad).length;

  if (cargando) return <EstadoCarga />;
  if (error) return <EstadoError mensaje={error} />;

  return <>
    <div className="titulo-seccion"><div><p>Notification &amp; Audit Service</p><h2>Eventos</h2></div></div>
    <section className="panel-eventos">
      <div className="cabecera-eventos"><div><strong>Historial de eventos</strong><small>Clasificación operativa del sistema</small></div><span className="contador-eventos">{visibles.length} registros</span></div>
      <div className="filtros-eventos" role="tablist" aria-label="Filtrar eventos por severidad">
        {severidades.map(opcion => <button key={opcion.valor} type="button" className={filtro === opcion.valor ? 'filtro-evento activo' : 'filtro-evento'} onClick={() => setFiltro(opcion.valor)}>
          {opcion.etiqueta}<span>{totalPorSeveridad(opcion.valor)}</span>
        </button>)}
      </div>
    </section>
    {!visibles.length ? <EstadoVacio mensaje="No hay eventos para la clasificación seleccionada." /> : <div className="lista-eventos">
      {visibles.map(registro => <article className={`evento evento-${registro.severity.toLowerCase()}`} key={registro.id}>
        <div className="evento-indicador" aria-hidden="true" />
        <div className="evento-contenido"><div className="evento-cabecera"><strong>{registro.eventType}</strong><span className={`severidad severidad-${registro.severity.toLowerCase()}`}>{iconosSeveridad[registro.severity]} {registro.severity}</span></div>
          <small>{registro.producer} · {new Date(registro.occurredAt).toLocaleString('es-GT')}</small><span className="evento-correlacion">CorrelationId: {registro.correlationId}</span>
        </div>
      </article>)}
    </div>}
  </>;
}

import { useEffect, useMemo, useState } from 'react';
import SvgIcon, { type SvgIconProps } from '@mui/material/SvgIcon';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { servicioAuditoria } from '../services/servicioAuditoria';
import type { RegistroAuditoria } from '../types/auditoria';

type FiltroSeveridad = 'TODOS' | RegistroAuditoria['severity'];
type DatosEvento = Record<string, unknown>;

const severidades: Array<{ valor: FiltroSeveridad; etiqueta: string }> = [
  { valor: 'TODOS', etiqueta: 'Todos' },
  { valor: 'INFO', etiqueta: 'Información' },
  { valor: 'WARNING', etiqueta: 'Advertencias' },
  { valor: 'ERROR', etiqueta: 'Errores' },
];

const IconoInfo = (props: SvgIconProps) => <SvgIcon {...props}><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zM11 17h2v-6h-2v6zm0-8h2V7h-2v2z" /></SvgIcon>;
const IconoAdvertencia = (props: SvgIconProps) => <SvgIcon {...props}><path d="M1 21h22L12 2 1 21zm12-3h-2v2h2v-2zm0-2h-2v-4h2v4z" /></SvgIcon>;
const IconoError = (props: SvgIconProps) => <SvgIcon {...props}><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z" /></SvgIcon>;

const iconosSeveridad = {
  INFO: IconoInfo,
  WARNING: IconoAdvertencia,
  ERROR: IconoError,
};

const nombresEventos: Record<string, string> = {
  'cuenta.consultada': 'Cuenta consultada',
  'cuenta.historial.consultado': 'Historial de cuentas consultado',
  'cuenta.movimientos.consultados': 'Movimientos de cuenta consultados',
  'cuenta.creada': 'Cuenta creada',
  'cuenta.debitada': 'Cuenta debitada',
  'cuenta.acreditada': 'Cuenta acreditada',
  'cuenta.transferencia.validada': 'Transferencia validada',
  'cuenta.transferencia.rechazada': 'Transferencia rechazada',
  'pago.solicitado': 'Pago solicitado',
  'pago.completado': 'Pago completado',
  'pago.rechazado': 'Pago rechazado',
};

const esObjeto = (valor: unknown): valor is DatosEvento =>
  typeof valor === 'object' && valor !== null && !Array.isArray(valor);

const esNumero = (valor: unknown): valor is number =>
  typeof valor === 'number' && Number.isFinite(valor);

const formatearMonto = (centavos: number) =>
  new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(centavos / 100);

const formatearNombreEvento = (tipo: string) => {
  const nombre = nombresEventos[tipo] ?? tipo
    .split('.')
    .map(parte => parte.charAt(0).toUpperCase() + parte.slice(1))
    .join(' ');
  return `Evento: ${nombre}`;
};

const etiquetaCampo: Record<string, string> = {
  idCuenta: 'ID de cuenta',
  numeroCuenta: 'Número de cuenta',
  tipoCuenta: 'Tipo de cuenta',
  saldoCentavos: 'Saldo actual',
  saldoMinimoCentavos: 'Saldo mínimo',
  montoCentavos: 'Monto',
  montoTotalCentavos: 'Monto total',
  moneda: 'Moneda',
  estado: 'Estado',
  idCliente: 'ID del cliente',
  idOperacion: 'ID de operación',
  idTransferencia: 'ID de transferencia',
  codigo: 'Código',
  motivo: 'Motivo',
  descripcion: 'Descripción',
  beneficiario: 'Beneficiario',
  concepto: 'Concepto',
};

const camposDetalle = [
  'idCuenta', 'numeroCuenta', 'tipoCuenta', 'saldoCentavos', 'saldoMinimoCentavos',
  'montoCentavos', 'montoTotalCentavos', 'moneda', 'estado', 'idCliente',
  'idOperacion', 'idTransferencia', 'codigo', 'motivo', 'descripcion',
  'beneficiario', 'concepto',
];

function formatearCampo(clave: string, valor: unknown) {
  if (clave.endsWith('Centavos') && esNumero(valor)) return formatearMonto(valor);
  if (typeof valor === 'boolean') return valor ? 'Sí' : 'No';
  return String(valor);
}

function detalleEvento(registro: RegistroAuditoria) {
  const payload = esObjeto(registro.payload) ? registro.payload : {};
  const detalles = camposDetalle
    .filter(clave => payload[clave] !== undefined && payload[clave] !== null && payload[clave] !== '')
    .map(clave => ({ etiqueta: etiquetaCampo[clave], valor: formatearCampo(clave, payload[clave]) }));
  const cuentas = Array.isArray(payload.cuentas) ? payload.cuentas.filter(esObjeto) : [];
  return { detalles, cuentas };
}

export function PaginaEventos() {
  const [registros, setRegistros] = useState<RegistroAuditoria[]>([]);
  const [filtro, setFiltro] = useState<FiltroSeveridad>('TODOS');
  const [eventoAbierto, setEventoAbierto] = useState<string | null>(null);
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    servicioAuditoria.listar(100)
      .then(setRegistros)
      .catch(e => setError(e instanceof Error ? e.message : 'No fue posible consultar los eventos'))
      .finally(() => setCargando(false));
  }, []);

  const ultimosPorSeveridad = useMemo(() => ({
    INFO: registros.filter(registro => registro.severity === 'INFO').slice(0, 50),
    WARNING: registros.filter(registro => registro.severity === 'WARNING').slice(0, 50),
    ERROR: registros.filter(registro => registro.severity === 'ERROR').slice(0, 50),
  }), [registros]);

  const visibles = useMemo(() => {
    if (filtro !== 'TODOS') return ultimosPorSeveridad[filtro];
    const idsVisibles = new Set([
      ...ultimosPorSeveridad.INFO,
      ...ultimosPorSeveridad.WARNING,
      ...ultimosPorSeveridad.ERROR,
    ].map(registro => registro.id));
    return registros.filter(registro => idsVisibles.has(registro.id));
  }, [filtro, registros, ultimosPorSeveridad]);

  const totalPorSeveridad = (severidad: FiltroSeveridad) =>
    severidad === 'TODOS'
      ? ultimosPorSeveridad.INFO.length + ultimosPorSeveridad.WARNING.length + ultimosPorSeveridad.ERROR.length
      : ultimosPorSeveridad[severidad].length;

  if (cargando) return <EstadoCarga />;
  if (error) return <EstadoError mensaje={error} />;

  return <>
    <div className="titulo-seccion"><div><h2>Evento</h2></div></div>
    <section className="panel-eventos">
      <div className="cabecera-eventos"><div><strong>Historial de eventos</strong><small>Clasificación operativa del sistema</small></div><span className="contador-eventos">{visibles.length} registros</span></div>
      <div className="filtros-eventos" role="tablist" aria-label="Filtrar eventos por severidad">
        {severidades.map(opcion => <button key={opcion.valor} type="button" className={filtro === opcion.valor ? 'filtro-evento activo' : 'filtro-evento'} onClick={() => setFiltro(opcion.valor)}>
          {opcion.etiqueta}<span>{totalPorSeveridad(opcion.valor)}</span>
        </button>)}
      </div>
    </section>
    {!visibles.length ? <EstadoVacio mensaje="No hay eventos para la clasificación seleccionada." /> : <div className="lista-eventos">
      {visibles.map(registro => {
        const Icono = iconosSeveridad[registro.severity];
        const detalle = detalleEvento(registro);
        const abierto = eventoAbierto === registro.id;
        return <article
          className={`evento evento-${registro.severity.toLowerCase()}${abierto ? ' evento-expandido' : ''}`}
          key={registro.id}
          role="button"
          tabIndex={0}
          aria-expanded={abierto}
          onClick={() => setEventoAbierto(abierto ? null : registro.id)}
          onKeyDown={evento => {
            if (evento.key === 'Enter' || evento.key === ' ') {
              evento.preventDefault();
              setEventoAbierto(abierto ? null : registro.id);
            }
          }}
        >
          <div className="evento-indicador" aria-hidden="true" />
          <div className="evento-contenido">
            <div className="evento-cabecera"><strong>{formatearNombreEvento(registro.eventType)}</strong><span className={`severidad severidad-${registro.severity.toLowerCase()}`}><span className="icono-severidad" aria-hidden="true"><Icono fontSize="small" /></span>{registro.severity}</span></div>
            <small>{registro.producer} · {new Date(registro.occurredAt).toLocaleString('es-GT')}</small>
            <span className="evento-correlacion">CorrelationId: {registro.correlationId}</span>
            {abierto && <div className="evento-detalle" onClick={evento => evento.stopPropagation()}>
              <div className="detalle-titulo">Información detallada</div>
              <div className="detalle-grid">
                <div className="detalle-item"><span>Evento</span><strong>{formatearNombreEvento(registro.eventType)}</strong></div>
                <div className="detalle-item"><span>Fecha</span><strong>{new Date(registro.occurredAt).toLocaleString('es-GT')}</strong></div>
                {detalle.detalles.map(item => <div className="detalle-item" key={`${registro.id}-${item.etiqueta}`}><span>{item.etiqueta}</span><strong>{item.valor}</strong></div>)}
              </div>
              {detalle.cuentas.length > 0 && <div className="detalle-cuentas"><strong>Cuentas consultadas</strong>{detalle.cuentas.map((cuenta, indice) => <div className="cuenta-detalle" key={String(cuenta.idCuenta ?? indice)}><span>{String(cuenta.idCuenta ?? 'Cuenta sin identificador')}</span><strong>{esNumero(cuenta.saldoCentavos) ? formatearMonto(cuenta.saldoCentavos) : 'Saldo no disponible'}</strong></div>)}</div>}
            </div>}
          </div>
        </article>;
      })}
    </div>}
  </>;
}

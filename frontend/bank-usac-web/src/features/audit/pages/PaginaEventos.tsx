import { useEffect, useMemo, useState } from 'react';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Chip from '@mui/material/Chip';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import WarningAmberOutlinedIcon from '@mui/icons-material/WarningAmberOutlined';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CloseIcon from '@mui/icons-material/Close';
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

const iconosSeveridad = {
  INFO: InfoOutlinedIcon,
  WARNING: WarningAmberOutlinedIcon,
  ERROR: ErrorOutlineIcon,
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
  const [eventoSeleccionado, setEventoSeleccionado] = useState<RegistroAuditoria | null>(null);
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

  const detalleSeleccionado = eventoSeleccionado ? detalleEvento(eventoSeleccionado) : null;
  const IconoSeleccionado = eventoSeleccionado ? iconosSeveridad[eventoSeleccionado.severity] : null;

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
        return <article
          className={`evento evento-${registro.severity.toLowerCase()}`}
          key={registro.id}
          role="button"
          tabIndex={0}
          aria-label={`Ver detalle de ${formatearNombreEvento(registro.eventType)}`}
          onClick={() => setEventoSeleccionado(registro)}
          onKeyDown={evento => {
            if (evento.key === 'Enter' || evento.key === ' ') {
              evento.preventDefault();
              setEventoSeleccionado(registro);
            }
          }}
        >
          <div className="evento-indicador" aria-hidden="true" />
          <div className="evento-contenido">
            <div className="evento-cabecera"><strong>{formatearNombreEvento(registro.eventType)}</strong><span className={`severidad severidad-${registro.severity.toLowerCase()}`}><span className="icono-severidad" aria-hidden="true"><Icono fontSize="small" /></span>{registro.severity}</span></div>
            <small>{registro.producer} · {new Date(registro.occurredAt).toLocaleString('es-GT')}</small>
            <span className="evento-correlacion">CorrelationId: {registro.correlationId}</span>
          </div>
        </article>;
      })}
    </div>}
    <Dialog
      open={eventoSeleccionado !== null}
      onClose={() => setEventoSeleccionado(null)}
      fullWidth
      maxWidth="md"
      aria-labelledby="detalle-evento-titulo"
    >
      {eventoSeleccionado && detalleSeleccionado && IconoSeleccionado && <>
        <DialogTitle id="detalle-evento-titulo" className="detalle-dialogo-titulo">
          <span>Información del evento</span>
          <Button onClick={() => setEventoSeleccionado(null)} aria-label="Cerrar detalle" color="inherit" size="small" startIcon={<CloseIcon />}>
            Cerrar
          </Button>
        </DialogTitle>
        <DialogContent dividers className="detalle-dialogo-contenido">
          <div className="detalle-dialogo-resumen">
            <div>
              <strong>{formatearNombreEvento(eventoSeleccionado.eventType)}</strong>
              <small>{eventoSeleccionado.producer} · {new Date(eventoSeleccionado.occurredAt).toLocaleString('es-GT')}</small>
            </div>
            <Chip
              icon={<IconoSeleccionado />}
              label={eventoSeleccionado.severity}
              className={`severidad-chip severidad-chip-${eventoSeleccionado.severity.toLowerCase()}`}
            />
          </div>
          <div className="detalle-grid">
            <div className="detalle-item"><span>Identificador del evento</span><strong>{eventoSeleccionado.eventId}</strong></div>
            <div className="detalle-item"><span>CorrelationId</span><strong>{eventoSeleccionado.correlationId}</strong></div>
            {eventoSeleccionado.causationId && <div className="detalle-item"><span>CausationId</span><strong>{eventoSeleccionado.causationId}</strong></div>}
            {eventoSeleccionado.version !== undefined && <div className="detalle-item"><span>Versión</span><strong>{eventoSeleccionado.version}</strong></div>}
            <div className="detalle-item"><span>Fecha del evento</span><strong>{new Date(eventoSeleccionado.occurredAt).toLocaleString('es-GT')}</strong></div>
            {eventoSeleccionado.recordedAt && <div className="detalle-item"><span>Fecha registrada</span><strong>{new Date(eventoSeleccionado.recordedAt).toLocaleString('es-GT')}</strong></div>}
            {detalleSeleccionado.detalles.map(item => <div className="detalle-item" key={`${eventoSeleccionado.id}-${item.etiqueta}`}><span>{item.etiqueta}</span><strong>{item.valor}</strong></div>)}
          </div>
          {detalleSeleccionado.cuentas.length > 0 && <div className="detalle-cuentas"><strong>Cuentas consultadas</strong>{detalleSeleccionado.cuentas.map((cuenta, indice) => <div className="cuenta-detalle" key={String(cuenta.idCuenta ?? indice)}><span>{String(cuenta.idCuenta ?? 'Cuenta sin identificador')}</span><strong>{esNumero(cuenta.saldoCentavos) ? formatearMonto(cuenta.saldoCentavos) : 'Saldo no disponible'}</strong></div>)}</div>}
          <details className="detalle-payload">
            <summary>Payload completo del evento</summary>
            <pre>{JSON.stringify(eventoSeleccionado.payload, null, 2)}</pre>
          </details>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEventoSeleccionado(null)} variant="contained">Cerrar</Button>
        </DialogActions>
      </>}
    </Dialog>
  </>;
}

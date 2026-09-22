import { FormEvent, useCallback, useState } from 'react';
import { Link } from 'react-router-dom';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { servicioCuentas } from '../../accounts/services/servicioCuentas';
import { servicioTransferencias, type FiltrosTransferencias } from '../services/servicioTransferencias';

const dinero = (v: number) => new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(v / 100);

export function PaginaTransferencias() {
  const [filtros, setFiltros] = useState<FiltrosTransferencias>({});
  const [aplicados, setAplicados] = useState<FiltrosTransferencias>({});
  const consultarCuentas = useCallback(() => servicioCuentas.listar(), []);
  const cuentas = usarConsulta(consultarCuentas);
  const consultarTransferencias = useCallback(() => servicioTransferencias.listar(aplicados), [aplicados]);
  const transferencias = usarConsulta(consultarTransferencias);

  function aplicar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    setAplicados({ ...filtros });
  }

  function limpiar() {
    setFiltros({});
    setAplicados({});
  }

  if (transferencias.cargando && !transferencias.datos) return <EstadoCarga />;
  if (transferencias.error) return <EstadoError mensaje={transferencias.error} />;

  return <>
    <div className="titulo-seccion">
      <div><p>Operaciones entre cuentas</p><h2>Transferencias</h2></div>
      <Link className="boton" to="/transferencias/nueva">Nueva transferencia</Link>
    </div>
    <form className="panel-formulario filtros-transferencias" onSubmit={aplicar}>
      <div className="filtros-encabezado"><div><p>Consulta avanzada</p><h3>Filtrar historial</h3></div><button type="button" className="secundario" onClick={limpiar}>Limpiar</button></div>
      <div className="rejilla-filtros">
        <label>Cuenta
          <select value={filtros.idCuenta ?? ''} onChange={e => setFiltros({ ...filtros, idCuenta: e.target.value || undefined })}>
            <option value="">Todas mis cuentas</option>
            {cuentas.datos?.map(c => <option key={c.idCuenta} value={c.idCuenta}>{c.numeroCuenta} · {c.tipoCuenta}</option>)}
          </select>
        </label>
        <label>Desde<input type="date" value={filtros.fechaDesde ?? ''} onChange={e => setFiltros({ ...filtros, fechaDesde: e.target.value || undefined })} /></label>
        <label>Hasta<input type="date" value={filtros.fechaHasta ?? ''} onChange={e => setFiltros({ ...filtros, fechaHasta: e.target.value || undefined })} /></label>
        <label>Estado
          <select value={filtros.estado ?? ''} onChange={e => setFiltros({ ...filtros, estado: e.target.value || undefined })}>
            <option value="">Todos los estados</option>
            <option value="PENDIENTE">Pendiente</option><option value="PROCESANDO">Procesando</option>
            <option value="COMPLETADA">Completada</option><option value="RECHAZADA">Rechazada</option>
            <option value="COMPENSANDO">Compensando</option><option value="COMPENSADA">Compensada</option>
            <option value="COMPENSACION_FALLIDA">Compensación fallida</option>
          </select>
        </label>
      </div>
      <button disabled={transferencias.cargando}>Aplicar filtros</button>
    </form>
    {!transferencias.datos?.length ? <EstadoVacio mensaje="No hay transferencias que coincidan con los filtros." /> : <div className="lista">{transferencias.datos.map(t => <Link className="fila-transferencia" key={t.idTransferencia} to={`/transferencias/${t.idTransferencia}`}><div><strong>{t.descripcion || 'Transferencia bancaria'}</strong><small>{new Date(t.fechaCreacion).toLocaleString('es-GT')} · Destino •••• {t.idCuentaDestino.slice(-4)}</small></div><div className="alineado-derecha"><strong>{dinero(t.montoCentavos)}</strong><span className={`estado ${t.estado.toLowerCase()}`}>{t.estado.replaceAll('_', ' ')}</span></div></Link>)}</div>}
  </>;
}

import { useEffect, useState, type FormEvent } from 'react';
import { useSearchParams } from 'react-router-dom';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import type { Cuenta } from '../../accounts/types/cuenta';
import type { ClienteAdministrado } from '../types/clienteAdministrado';
import { servicioCuentasAdministracion as servicio, type ConfiguracionMonitoreo } from '../services/servicioCuentasAdministracion';
import './cuentas-administracion.css';

const monto = (centavos: number) => new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(centavos / 100);
const mensajeError = (e: unknown) => e instanceof Error ? e.message : 'No fue posible completar la operación';

export function PaginaAdministracionCuentas() {
 const [parametros, setParametros] = useSearchParams();
 const cuentaFiltrada = parametros.get('cuenta') ?? '';
  const [cuentas, setCuentas] = useState<Cuenta[]>([]);
  const [propietarios, setPropietarios] = useState(new Map<string, ClienteAdministrado>());
  const [avisoPropietarios, setAvisoPropietarios] = useState('');
  const [total, setTotal] = useState(0);
  const [pagina, setPagina] = useState(0);
 const paginaVisible = cuentaFiltrada ? 0 : pagina;
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    const actualizar = () => setRevision(v => v + 1);
    window.addEventListener('bank-usac:cuenta-bloqueada', actualizar);
    return () => window.removeEventListener('bank-usac:cuenta-bloqueada', actualizar);
  }, []);
  const [cargando, setCargando] = useState(true);
  const [errorCuentas, setErrorCuentas] = useState('');
  const [configuracion, setConfiguracion] = useState<ConfiguracionMonitoreo>();
  const [maximo, setMaximo] = useState('');
  const [minutos, setMinutos] = useState('');
  const [errorConfiguracion, setErrorConfiguracion] = useState('');
  const [guardando, setGuardando] = useState(false);
  const [confirmacion, setConfirmacion] = useState<Cuenta | null>(null);
  const [motivo, setMotivo] = useState('');
  const [bloqueando, setBloqueando] = useState(false);
  const [errorBloqueo, setErrorBloqueo] = useState('');
  const [exito, setExito] = useState('');

  useEffect(() => {
    let vigente = true;
    setCargando(true);
    setErrorCuentas('');
    servicio.listar(cuentaFiltrada ? 0 : pagina, cuentaFiltrada).then(resultado => {
      if (vigente) { setCuentas(resultado.cuentas); setTotal(resultado.total); }
    }).catch(e => { if (vigente) setErrorCuentas(mensajeError(e)); })
      .finally(() => { if (vigente) setCargando(false); });
    return () => { vigente = false; };
  }, [pagina, revision, cuentaFiltrada]);

  useEffect(() => {
    let vigente = true;
    servicio.propietarios().then(lista => { if (vigente) { setPropietarios(lista); setAvisoPropietarios(''); } })
      .catch(() => { if (vigente) setAvisoPropietarios('No se pudieron consultar los nombres de los propietarios. Puedes identificar cada cuenta por el ID de su cliente.'); });
    return () => { vigente = false; };
  }, [revision]);

  useEffect(() => {
    let vigente = true;
    servicio.configuracion().then(c => {
      if (vigente) { setConfiguracion(c); setMaximo(String(c.maxOperaciones)); setMinutos(String(c.ventanaMinutos)); setErrorConfiguracion(''); }
    }).catch(e => { if (vigente) setErrorConfiguracion(mensajeError(e)); });
    return () => { vigente = false; };
  }, [revision]);

  async function guardar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const n = Number(maximo), m = Number(minutos);
    if (!Number.isInteger(n) || n < 1 || n > 10000 || !Number.isInteger(m) || m < 1 || m > 1440) {
      setErrorConfiguracion('Indica entre 1 y 10000 operaciones y entre 1 y 1440 minutos.'); return;
    }
    setGuardando(true); setErrorConfiguracion(''); setExito('');
    try { const c = await servicio.configurar(n, m); setConfiguracion(c); setExito('Los nuevos límites se aplicarán a las siguientes operaciones.'); }
    catch (e) { setErrorConfiguracion(mensajeError(e)); }
    finally { setGuardando(false); }
  }

  async function bloquear() {
    if (!confirmacion || motivo.trim().length < 5) return;
    setBloqueando(true); setErrorBloqueo(''); setExito('');
    try {
      const cuenta = await servicio.bloquear(confirmacion.idCuenta, motivo.trim());
      setCuentas(lista => lista.map(item => item.idCuenta === cuenta.idCuenta ? cuenta : item));
      setConfirmacion(null); setExito(`La cuenta ${cuenta.numeroCuenta} quedó bloqueada.`);
    } catch (e) { setErrorBloqueo(mensajeError(e)); }
    finally { setBloqueando(false); }
  }

  return <>
    <div className="titulo-seccion"><div><p>Account Service</p><h2>Administrar cuentas</h2></div>
      <button type="button" className="secundario" disabled={cargando || guardando || bloqueando} onClick={() => setRevision(r => r + 1)}>Actualizar</button>
    </div>
    {exito && <p className="alerta" role="status">{exito}</p>}
    <section className="panel-formulario monitoreo-cuentas" aria-labelledby="monitoreo-titulo">
      <p>Detección de patrones</p><h3 id="monitoreo-titulo">Alta frecuencia de transacciones</h3>
      {errorConfiguracion && <p className="alerta error" role="alert">{errorConfiguracion}</p>}
      {!configuracion && !errorConfiguracion && <p role="status">Consultando configuración…</p>}
      <form onSubmit={e => void guardar(e)} className="formulario-monitoreo">
        <label htmlFor="max-operaciones">Número de operaciones (N)<input id="max-operaciones" type="number" min="1" max="10000" step="1" required value={maximo} disabled={!configuracion || guardando} onChange={e => setMaximo(e.target.value)} /></label>
        <label htmlFor="ventana-minutos">Ventana en minutos (M)<input id="ventana-minutos" type="number" min="1" max="1440" step="1" required value={minutos} disabled={!configuracion || guardando} onChange={e => setMinutos(e.target.value)} /></label>
        <button type="submit" disabled={!configuracion || guardando}>{guardando ? 'Guardando…' : 'Guardar límites'}</button>
      </form>
      {configuracion && <small>Configuración vigente: más de {configuracion.maxOperaciones} operaciones en {configuracion.ventanaMinutos} minutos. Se emite como máximo una alerta por cuenta y ventana con la misma configuración.</small>}
    </section>
    {avisoPropietarios && <p className="alerta">{avisoPropietarios}</p>}
    <section aria-labelledby="cuentas-titulo">
      <h3 id="cuentas-titulo">Cuentas y propietarios</h3>
 {cuentaFiltrada && <p className="alerta">Cuenta seleccionada desde la alerta. <button type="button" className="secundario" onClick={() => { setPagina(0); setParametros({}); }}>Ver todas las cuentas</button></p>}
      {cargando ? <p role="status">Consultando cuentas…</p> : errorCuentas ? <p className="alerta error" role="alert">{errorCuentas}</p> : !cuentas.length ? <p className="estado-vacio">No hay cuentas en esta página.</p> :
        <div className="cuentas-admin-tabla"><table><thead><tr><th>Cuenta</th><th>Propietario</th><th>Tipo</th><th>Saldo</th><th>Estado</th><th>Acción</th></tr></thead>
          <tbody>{cuentas.map(cuenta => <tr key={cuenta.idCuenta}>
            <td><strong>{cuenta.numeroCuenta}</strong><small>{cuenta.idCuenta}</small></td>
            <td>{propietarios.get(cuenta.idCliente)?.fullName ?? 'Nombre no disponible'}<small>ID: {cuenta.idCliente}</small></td>
            <td>{cuenta.tipoCuenta}</td><td>{monto(cuenta.saldoCentavos)}</td><td><span className={`estado ${cuenta.estado.toLowerCase()}`}>{cuenta.estado}</span></td>
            <td><button type="button" className="secundario" disabled={bloqueando || cuenta.estado === 'BLOQUEADA' || cuenta.estado === 'CERRADA'} onClick={() => { setConfirmacion(cuenta); setMotivo(''); setErrorBloqueo(''); }}>{cuenta.estado === 'BLOQUEADA' ? 'Bloqueada' : 'Bloquear cuenta'}</button></td>
          </tr>)}</tbody></table></div>}
      <div className="paginacion-cuentas"><button type="button" className="secundario" disabled={paginaVisible === 0 || cargando} onClick={() => setPagina(p => p - 1)}>Anterior</button>
        <span>Página {paginaVisible + 1} · {total} cuentas</span><button type="button" className="secundario" disabled={(paginaVisible + 1) * 25 >= total || cargando} onClick={() => setPagina(p => p + 1)}>Siguiente</button></div>
    </section>
    <Dialog open={Boolean(confirmacion)} onClose={() => { if (!bloqueando) setConfirmacion(null); }} fullWidth maxWidth="sm" aria-labelledby="bloquear-cuenta-titulo">
      <DialogTitle id="bloquear-cuenta-titulo">Bloquear cuenta {confirmacion?.numeroCuenta}</DialogTitle>
      <DialogContent><p>Se impedirán nuevos débitos y créditos. Las compensaciones de operaciones anteriores seguirán permitidas para devolver fondos.</p>
        <label className="motivo-bloqueo" htmlFor="motivo-bloqueo">Motivo del bloqueo<textarea id="motivo-bloqueo" minLength={5} maxLength={300} value={motivo} disabled={bloqueando} onChange={e => setMotivo(e.target.value)} /></label>
        {errorBloqueo && <p className="alerta error" role="alert">{errorBloqueo}</p>}
      </DialogContent><DialogActions><button type="button" className="secundario" disabled={bloqueando} onClick={() => setConfirmacion(null)}>Cancelar</button><button type="button" disabled={bloqueando || motivo.trim().length < 5} onClick={() => void bloquear()}>{bloqueando ? 'Bloqueando…' : 'Confirmar bloqueo'}</button></DialogActions>
    </Dialog>
  </>;
}

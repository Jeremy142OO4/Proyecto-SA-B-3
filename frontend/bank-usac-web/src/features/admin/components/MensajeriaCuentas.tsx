import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import Button from '@mui/material/Button';
import { servicioCuentasAdministracion } from '../services/servicioCuentasAdministracion';
import Badge from '@mui/material/Badge';
import IconButton from '@mui/material/IconButton';
import Popover from '@mui/material/Popover';
import ChatBubbleOutlineIcon from '@mui/icons-material/ChatBubbleOutline';
import { clienteApi } from '../../../services/clienteApi';
import './mensajeria-cuentas.css';

interface AlertaCuenta {
  id: string;
  accountId?: string;
  severity: 'WARNING';
  subject: string;
  bodySummary: string;
  sentAt: string;
  correlationId: string;
  details: { idCuenta?: string; tipoCuenta?: string; operaciones?: number; maxOperaciones?: number; ventanaMinutos?: number };
}

function leerIds(clave: string): string[] {
  try { const valor: unknown = JSON.parse(localStorage.getItem(clave) ?? '[]'); return Array.isArray(valor) ? valor.filter((id): id is string => typeof id === 'string') : []; }
  catch { return []; }
}

export function MensajeriaCuentas({ usuarioId }: { usuarioId: string }) {
  const clave = `bank-usac:alertas-cuentas-leidas:${usuarioId}`;
  const [leidas, setLeidas] = useState<string[]>(() => leerIds(clave));
  const [alertas, setAlertas] = useState<AlertaCuenta[]>([]);
  const [ancla, setAncla] = useState<HTMLElement | null>(null);
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const [cuentaBloquear, setCuentaBloquear] = useState('');
  const [bloqueando, setBloqueando] = useState(false);
  const [bloqueadas, setBloqueadas] = useState<string[]>([]);
  const [errorBloqueo, setErrorBloqueo] = useState('');
  const motivo = 'Se han detectado múltiples transacciones en poco tiempo.';
  async function bloquear() {
    setBloqueando(true); setErrorBloqueo('');
    try {
      await servicioCuentasAdministracion.bloquear(cuentaBloquear, motivo);
      setBloqueadas(ids => [...ids, cuentaBloquear]);
      window.dispatchEvent(new Event('bank-usac:cuenta-bloqueada'));
      setCuentaBloquear('');
    } catch (e) { setErrorBloqueo(e instanceof Error ? e.message : 'No fue posible bloquear la cuenta.'); }
    finally { setBloqueando(false); }
  }

  useEffect(() => {
    let vigente = true;
 setCargando(true);
    let timer: ReturnType<typeof setTimeout>;
    async function consultar() {
      try {
        const lista = await clienteApi.obtener<AlertaCuenta[] | null>('/auditoria/notificaciones?tipo=ACCOUNT_ANOMALY_DETECTED&nivel=WARNING&limite=50');
        if (vigente) { setAlertas(Array.isArray(lista) ? lista : []); setError(''); }
      } catch { if (vigente) setError('No se pudieron actualizar las alertas.'); }
      finally { if (vigente) { setCargando(false); timer = setTimeout(() => void consultar(), 15_000); } }
    }
    void consultar();
    return () => { vigente = false; clearTimeout(timer); };
  }, [revision]);

  const sinLeer = alertas.filter(alerta => !leidas.includes(alerta.id)).length;
  function marcarLeidas() {
    const ids = alertas.map(alerta => alerta.id);
    setLeidas(ids);
    try { localStorage.setItem(clave, JSON.stringify(ids)); } catch { /* La sesión actual conserva el estado. */ }
  }

  return <>
    <IconButton color="inherit" aria-label={`Mensajería de cuentas, ${sinLeer} alertas sin leer`} aria-haspopup="dialog" aria-expanded={Boolean(ancla)} onClick={e => setAncla(e.currentTarget)}>
      <Badge badgeContent={sinLeer} color="warning"><ChatBubbleOutlineIcon /></Badge>
    </IconButton>
    <Popover open={Boolean(ancla)} anchorEl={ancla} onClose={() => setAncla(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }} transformOrigin={{ vertical: 'top', horizontal: 'right' }}>
      <section className="mensajeria-cuentas" role="dialog" aria-label="Alertas de cuentas">
        <header><div><strong>Mensajería de cuentas</strong></div><button type="button" className="secundario" onClick={() => setAncla(null)} aria-label="Cerrar mensajería">Cerrar</button></header>
        <div className="mensajeria-acciones"><button type="button" className="secundario" disabled={cargando} onClick={() => setRevision(v => v + 1)}>Actualizar</button><button type="button" className="secundario" disabled={sinLeer === 0} onClick={marcarLeidas}>Marcar como leídas</button></div>
        {error && <p className="alerta error" role="alert">{error} Se conservan las últimas alertas consultadas.</p>}
        {cargando ? <p role="status">Consultando notificaciones…</p> : !alertas.length ? <p>No hay alertas de alta frecuencia disponibles.</p> :
          <ul>{alertas.map(alerta => <li key={alerta.id} className={leidas.includes(alerta.id) ? '' : 'alerta-no-leida'}>
            <strong>Alta frecuencia de operaciones</strong>
            <p>Cuenta: {alerta.accountId ?? alerta.details?.idCuenta ?? 'Identificador no disponible'}</p>
            {typeof alerta.details?.operaciones === 'number' ? <p>{alerta.details.operaciones} operaciones en {alerta.details.ventanaMinutos} minutos. Límite: {alerta.details.maxOperaciones}.</p> : <p>{alerta.bodySummary}</p>}
            {alerta.details?.tipoCuenta && <small>Tipo de cuenta: {alerta.details.tipoCuenta}</small>}
            <time dateTime={alerta.sentAt}>{new Date(alerta.sentAt).toLocaleString('es-GT')}</time>
            <small>Correlación: {alerta.correlationId}</small>
            <button type="button" disabled={! (alerta.accountId ?? alerta.details?.idCuenta) || bloqueadas.includes(alerta.accountId ?? alerta.details?.idCuenta ?? '')} onClick={() => { setCuentaBloquear(alerta.accountId ?? alerta.details?.idCuenta ?? ''); setErrorBloqueo(''); setAncla(null); }}>{bloqueadas.includes(alerta.accountId ?? alerta.details?.idCuenta ?? '') ? 'Cuenta bloqueada' : 'Bloquear cuenta'}</button>
            <Link to={`/administracion/cuentas?cuenta=${encodeURIComponent(alerta.accountId ?? alerta.details?.idCuenta ?? '')}`} onClick={() => setAncla(null)}>Revisar cuenta</Link>
          </li>)}</ul>}
        <footer>Últimas 50 alertas. Se actualizan cada 15 segundos. La lectura se guarda en este navegador.</footer>
      </section>
    </Popover>
    <Dialog open={Boolean(cuentaBloquear)} onClose={() => { if (!bloqueando) setCuentaBloquear(''); }} aria-labelledby="titulo-bloqueo-alerta">
      <DialogTitle id="titulo-bloqueo-alerta">Bloquear cuenta</DialogTitle>
      <DialogContent><p>Cuenta: {cuentaBloquear}</p><p>Motivo: {motivo}</p><p>Se notificará al propietario. Podrá consultar su historial, pero no realizar nuevas transacciones con esta cuenta.</p>{errorBloqueo && <p role="alert">{errorBloqueo}</p>}</DialogContent>
      <DialogActions><Button disabled={bloqueando} onClick={() => setCuentaBloquear('')}>Cancelar</Button><Button color="error" variant="contained" disabled={bloqueando} onClick={() => void bloquear()}>{bloqueando ? 'Bloqueando…' : 'Confirmar bloqueo'}</Button></DialogActions>
    </Dialog>
  </>;
}

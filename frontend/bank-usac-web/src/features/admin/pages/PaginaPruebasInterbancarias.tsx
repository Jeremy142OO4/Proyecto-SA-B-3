import { useEffect, useState } from 'react';
import { clienteApi } from '../../../services/clienteApi';
import './pruebas-interbancarias.css';

type Escenario = 'accept' | 'reject' | 'unavailable' | 'no_callback' | 'delayed';
interface Estado { enabled: boolean; target: string; environment?: string; scenario?: Escenario; delaySeconds?: number; accounts?: string[]; updatedAt?: string; callbackConfigured?: boolean; grafanaUrl?: string; mensaje?: string }
const ruta = '/administracion/pruebas-interbancarias/';
const escenarios: { id: Escenario; titulo: string; detalle: string; verificar: string }[] = [
  { id: 'accept', titulo: 'Operación normal', detalle: 'Acepta el depósito y envía su resultado.', verificar: 'Confirmar ACCEPTED y un único débito.' },
  { id: 'reject', titulo: 'Rechazo', detalle: 'Devuelve REJECTED / PROCESSING_ERROR.', verificar: 'Confirmar liberación de la reserva, sin débito.' },
  { id: 'unavailable', titulo: 'Error 503', detalle: 'Las nuevas solicitudes reciben SERVICE_UNAVAILABLE.', verificar: 'Observar reintentos con el mismo ID y recuperación.' },
  { id: 'no_callback', titulo: 'Sin callback', detalle: 'Procesa el depósito, pero no envía el resultado.', verificar: 'Verificar recuperación por reintento idempotente o consulta tras TIMEOUT, sin duplicar el débito.' },
  { id: 'delayed', titulo: 'Respuesta demorada', detalle: 'Programa el procesamiento para después de 181 s.', verificar: 'Observar TIMEOUT y cancelación o rechazo por vencimiento; no es un callback tardío aceptado.' },
];
const finales = ['ACCEPTED', 'REJECTED', 'REVERSED', 'CANCELLED'];
const explicarEstado = (status: string) => ({
  ACCEPTED: 'Prueba completada: el destino aceptó el depósito. Verificá que exista un solo débito y que la reserva sea cero.',
  REVERSED: 'Fondos recuperados: el depósito no se completó y la reserva fue liberada.',
  REJECTED: 'Depósito rechazado. Revisá el motivo y confirmá que no quedó dinero reservado.',
  CANCELLED: 'Depósito cancelado. Revisá la liberación de la reserva.',
  TIMEOUT: 'El banco todavía no conoce el resultado. La reserva se conserva: no significa que se haya perdido el dinero. Esperá la recuperación automática.',
  PROCESSING: 'El depósito está en proceso. Este panel consultará el mismo ID automáticamente.',
  RESPUESTA_INCIERTA: 'No se confirmó la respuesta. Consultaremos el mismo ID; no crearemos otro depósito.',
  CONSULTAR: 'Recuperamos el ID de tu última prueba. Consultá su resultado antes de enviar otra.',
  ENVIANDO: 'Enviando una única solicitud al banco de pruebas…',
} as Record<string, string>)[status] ?? 'Esperando que el procesamiento termine. Consultá el mismo ID para conocer el resultado.';

export function PaginaPruebasInterbancarias() {
  const [estado, setEstado] = useState<Estado>();
  const [ocupado, setOcupado] = useState(false);
  const [error, setError] = useState('');
  const [exito, setExito] = useState('');
  const [monto, setMonto] = useState('100.00');
  const [deposito, setDeposito] = useState<{ id: string; status: string; reasonCode?: string } | undefined>(() => {
    const id = sessionStorage.getItem('sandbox-deposit-id');
    return id && /^[0-9a-f-]{36}$/.test(id) ? { id, status: 'CONSULTAR' } : undefined;
  });
  const [enviando, setEnviando] = useState(false);
  const [saldo, setSaldo] = useState<{ saldoCentavos: number; saldoReservadoCentavos: number; saldoDisponibleCentavos: number }>();
  const dinero = (centavos: number) => new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(centavos / 100);
  useEffect(() => {
    let vigente = true;
    clienteApi.obtener<Estado>(ruta).then(e => { if (vigente) setEstado(e); }).catch(() => { if (vigente) setError('No se pudo consultar el entorno. Los controles permanecen deshabilitados.'); });
    return () => { vigente = false; };
  }, []);
  useEffect(() => {
    if (!estado?.enabled) return;
    let vigente = true;
    clienteApi.obtener<{ cuentas: { numeroCuenta: string; saldoCentavos: number; saldoReservadoCentavos: number; saldoDisponibleCentavos: number }[] }>(`${ruta}saldo`)
      .then(r => { if (vigente) setSaldo(r.cuentas.find(c => c.numeroCuenta === '900000000003')); })
      .catch(() => { if (vigente) setSaldo(undefined); });
    return () => { vigente = false; };
  }, [estado?.enabled, deposito]);
  async function consultarDeposito(id: string) {
    try {
      const resultado = await clienteApi.obtener<{ status: string; reasonCode?: string }>(`${ruta}depositos/${id}`);
      setDeposito({ id, ...resultado });
    } catch { setError('No se pudo consultar el depósito. Conservá su ID y no lo reenvíes.'); }
  }
  useEffect(() => {
    if (!deposito || finales.includes(deposito.status)) return;
    const id = deposito.id;
    const timer = window.setTimeout(() => { if (!document.hidden) void consultarDeposito(id); }, 5000);
    const volver = () => { if (!document.hidden && !enviando) void consultarDeposito(id); };
    document.addEventListener('visibilitychange', volver);
    return () => { window.clearTimeout(timer); document.removeEventListener('visibilitychange', volver); };
  }, [deposito, enviando]);
  async function enviarDeposito() {
    if (!estado?.enabled || enviando || ocupado || (deposito && !finales.includes(deposito.status))) return;
    const numero = Number(monto);
    if (!Number.isFinite(numero) || numero < .01 || numero > 10000 || !/^\d{1,5}(\.\d{1,2})?$/.test(monto)) { setError('Monto válido: Q0.01 a Q10,000.00, con máximo dos decimales.'); return; }
    const id = crypto.randomUUID();
    sessionStorage.setItem('sandbox-deposit-id', id);
    setDeposito({ id, status: 'ENVIANDO' }); setEnviando(true); setError('');
    try {
      const resultado = await clienteApi.publicar<{ status: string; reasonCode?: string }>(`${ruta}depositos`, { id, amount: numero.toFixed(2) });
      setDeposito({ id, ...resultado });
    } catch {
      setDeposito({ id, status: 'RESPUESTA_INCIERTA' });
      setError('No se confirmó el envío. Consultá este mismo ID antes de intentar otra operación.');
    } finally { setEnviando(false); }
  }
  async function cambiar(scenario: Escenario) {
    if (!estado?.enabled || ocupado) return;
    setOcupado(true); setError(''); setExito('');
    try {
      const nuevo = await clienteApi.actualizar<Estado>(ruta, { scenario });
      setEstado(nuevo);
      setExito('Configuración confirmada. Se aplicará a los siguientes depósitos; no se ha enviado ninguna transferencia.');
    } catch {
      setEstado(undefined);
      setError('No se pudo confirmar el cambio. No envíes una prueba hasta volver a consultar el entorno.');
    } finally { setOcupado(false); }
  }
  async function consultar() {
    setOcupado(true); setError('');
    try { setEstado(await clienteApi.obtener<Estado>(ruta)); }
    catch { setEstado(undefined); setError('El entorno de pruebas no está disponible.'); }
    finally { setOcupado(false); }
  }
  return <div className="pruebas-banco">
    <header className="pruebas-titulo"><div><p>BANK03 / CENTRO DE DEMOSTRACIÓN</p><h2>Simulador de escenarios</h2><span>Provocá un fallo controlado y comprobá cómo se recupera el banco.</span></div>
      <button type="button" onClick={consultar} disabled={ocupado}>Consultar entorno</button></header>
    <div className="pruebas-pasos"><span><b>1</b> Elegí el escenario</span><span><b>2</b> Enviá una prueba</span><span><b>3</b> Observá la recuperación</span></div>
    <section className={`pruebas-entorno ${estado?.enabled ? 'aislado' : ''}`} aria-live="polite">
      <strong>{estado?.enabled ? `Laboratorio conectado · ${estado.environment || 'Entorno aislado'}` : 'Verificando disponibilidad del laboratorio'}</strong>
      <p>{estado?.enabled ? 'Este panel usa un mock del banco externo y una cuenta ficticia independiente. No modifica BANK14 ni el dinero de las cuentas del portal. Los depósitos que el mock ya recibió conservan su escenario original.' : estado?.mensaje ?? 'Los controles se habilitan cuando el laboratorio está disponible. No se cambiará la conexión con el banco pareja.'}</p>
      {estado?.enabled && <><p>Cuenta ficticia de destino: <code>{estado.accounts?.join(', ')}</code></p><p>Callbacks: {estado.callbackConfigured ? 'configurados' : 'NO configurados: la aceptación se resolverá por consulta, no por callback'}</p></>}
    </section>
    {error && <p className="pruebas-error" role="alert">{error}</p>}
    {exito && <p className="pruebas-exito" role="status">{exito}</p>}
    <h3 className="pruebas-seccion">1. ¿Qué querés demostrar?</h3>
    <div className="pruebas-escenarios">{escenarios.map(e => <article key={e.id} className={estado?.enabled && estado.scenario === e.id ? 'seleccionado' : ''}>
      <span className="pruebas-etiqueta">{estado?.enabled && estado.scenario === e.id ? 'ACTIVO' : 'ESCENARIO'}</span><h3>{e.titulo}</h3><p>{e.detalle}</p><small>{e.verificar}</small>
      <button type="button" aria-pressed={estado?.enabled && estado.scenario === e.id} disabled={!estado?.enabled || ocupado || estado.scenario === e.id} onClick={() => cambiar(e.id)}>{estado?.scenario === e.id ? '✓ Escenario seleccionado' : 'Usar este escenario'}</button>
    </article>)}</div>
    <section className="pruebas-guion"><h3>Cómo mostrar cada prueba al auxiliar</h3><ol><li>Seleccioná el escenario y esperá la confirmación.</li><li>Enviá un depósito desde el formulario de abajo.</li><li>Mostrá su ID, estado y reserva. No repitas un envío pendiente.</li><li>Para el error 503: mientras el depósito sigue pendiente, pulsá «Restablecer comportamiento normal» y observá el reintento del mismo depósito.</li><li>En Grafana buscá el ID y el entorno <code>interbank-cloud-tests</code> para distinguir estas pruebas del tráfico del banco real.</li></ol>
      <p>Cambiar un escenario no crea depósitos, no reinicia servicios y no cambia URLs de integración.</p>
      <div className="pruebas-acciones"><button type="button" disabled={!estado?.enabled || ocupado} onClick={() => cambiar('accept')}>Restablecer comportamiento normal del mock</button>
        {estado?.enabled && estado.grafanaUrl && <a href={estado.grafanaUrl} target="_blank" rel="noopener noreferrer">Abrir Grafana ↗</a>}</div>
      {estado?.enabled && !estado.grafanaUrl && <p>El enlace de observabilidad todavía no está configurado para este entorno.</p>}
    </section>
    {estado?.enabled && <section className="pruebas-guion"><h3>2. Enviá un depósito de demostración</h3>
      <p>Escenario activo: <strong>{escenarios.find(e => e.id === estado.scenario)?.titulo}</strong>. Cuenta ficticia <code>900000000003</code> → banco mock <code>TEST-BANK14-001</code>.</p>
      {saldo ? <div className="pruebas-saldos"><span>Saldo total <strong>{dinero(saldo.saldoCentavos)}</strong></span><span>Reservado <strong>{dinero(saldo.saldoReservadoCentavos)}</strong></span><span>Disponible <strong>{dinero(saldo.saldoDisponibleCentavos)}</strong></span></div> : <p>Saldo de la cuenta ficticia: no disponible todavía. Consultá Administrar cuentas antes de enviar.</p>}
      <label>Monto en quetzales <input type="number" min="0.01" max="10000" step="0.01" value={monto} onChange={e => setMonto(e.target.value)} /></label>
      <button type="button" disabled={ocupado || enviando || !!(deposito && !['ACCEPTED', 'REJECTED', 'REVERSED', 'CANCELLED'].includes(deposito.status))} onClick={enviarDeposito}>{enviando ? 'Enviando…' : 'Enviar nuevo depósito de prueba'}</button>
      {deposito && <div className={`pruebas-resultado ${finales.includes(deposito.status) ? 'finalizado' : ''}`} aria-live="polite"><h3>3. Resultado de la prueba</h3><p className="pruebas-estado">{deposito.status}{deposito.reasonCode && ` / ${deposito.reasonCode}`}</p><p>{explicarEstado(deposito.status)}</p><p>ID para consultar y buscar en Grafana: <code>{deposito.id}</code></p><button type="button" disabled={enviando} onClick={() => consultarDeposito(deposito.id)}>Actualizar resultado y saldo</button></div>}
    </section>}
  </div>;
}

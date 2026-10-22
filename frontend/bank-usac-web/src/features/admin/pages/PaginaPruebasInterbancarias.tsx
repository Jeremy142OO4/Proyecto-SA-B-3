import { useEffect, useState } from 'react';
import { clienteApi } from '../../../services/clienteApi';
import './pruebas-interbancarias.css';

type Escenario = 'accept' | 'reject' | 'unavailable' | 'no_callback' | 'delayed';
interface Estado { enabled: boolean; target: string; environment?: string; scenario?: Escenario; delaySeconds?: number; accounts?: string[]; updatedAt?: string; callbackConfigured?: boolean; grafanaUrl?: string; mensaje?: string }
const ruta = '/administracion/pruebas-interbancarias/';
const escenarios: { id: Escenario; titulo: string; detalle: string; verificar: string }[] = [
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
  const escenarioActivo = estado?.scenario === 'accept'
    ? 'Funcionamiento normal'
    : escenarios.find(e => e.id === estado?.scenario)?.titulo ?? 'Sin confirmar';
  const pendiente = !!deposito && !finales.includes(deposito.status);
  const nombreEstado = (status: string) => ({
    ACCEPTED: 'Depósito aceptado', REVERSED: 'Fondos recuperados', REJECTED: 'Depósito rechazado',
    CANCELLED: 'Depósito cancelado', TIMEOUT: 'Esperando confirmación', PROCESSING: 'Depósito en proceso',
    RESPUESTA_INCIERTA: 'Respuesta por confirmar', CONSULTAR: 'Prueba pendiente de consulta', ENVIANDO: 'Enviando depósito',
  } as Record<string, string>)[status] ?? 'Procesamiento en curso';

  return <div className="pruebas-banco">
    <header className="pruebas-titulo">
      <div><p className="pruebas-eyebrow">CENTRO DE DEMOSTRACIÓN</p><h2>Simulador de escenarios</h2>
        <p className="pruebas-intro">Elegí una condición de prueba, enviá un depósito y seguí la recuperación del banco.</p></div>
      <button className="pruebas-boton-secundario" type="button" onClick={consultar} disabled={ocupado}>
        {ocupado ? 'Consultando…' : 'Consultar entorno'}
      </button>
    </header>

    <ol className="pruebas-pasos" aria-label="Pasos de la demostración">
      <li className="actual"><span>1</span><div><strong>Elegí un escenario</strong><small>Configurá la respuesta</small></div></li>
      <li className={estado?.enabled ? 'disponible' : ''}><span>2</span><div><strong>Enviá un depósito</strong><small>Usá la cuenta de prueba</small></div></li>
      <li className={deposito ? 'disponible' : ''}><span>3</span><div><strong>Revisá el resultado</strong><small>Comprobá estado y saldo</small></div></li>
    </ol>

    <section className={`pruebas-entorno ${estado?.enabled ? 'aislado' : ''}`} aria-live="polite" aria-label="Estado del laboratorio">
      <div className="pruebas-conexion"><span className="pruebas-indicador" aria-hidden="true" /><div>
        <strong>{estado?.enabled ? 'Laboratorio conectado' : estado || error ? 'Laboratorio no disponible' : 'Consultando laboratorio…'}</strong>
        <small>{estado?.enabled ? estado.environment || 'Entorno aislado' : 'Los controles se habilitarán al confirmar la conexión.'}</small>
      </div></div>
      {estado?.enabled && <span className="pruebas-escenario-actual">Escenario activo <strong>{escenarioActivo}</strong></span>}
    </section>
    {error && <p className="pruebas-error" role="alert">{error}</p>}
    {exito && <p className="pruebas-exito" role="status">{exito}</p>}

    <section aria-labelledby="pruebas-escenarios-titulo">
      <div className="pruebas-seccion"><div><p className="pruebas-eyebrow">PASO 01</p><h3 id="pruebas-escenarios-titulo">¿Qué querés poner a prueba?</h3></div>
        <span>El cambio aplica a los siguientes depósitos.</span></div>
      <div className="pruebas-escenarios">{escenarios.map((e, index) => {
        const seleccionado = !!estado?.enabled && estado.scenario === e.id;
        return <article key={e.id} className={seleccionado ? 'seleccionado' : ''}>
          <div className="pruebas-tarjeta-cabecera"><span className="pruebas-numero" aria-hidden="true">0{index + 1}</span>
            <span className="pruebas-etiqueta">{seleccionado ? '✓ ACTIVO' : 'ESCENARIO DE PRUEBA'}</span></div>
          <h4>{e.titulo}</h4><p>{e.detalle}</p>
          <div className="pruebas-verificar"><strong>Qué verificar</strong><p>{e.verificar}</p></div>
          <button type="button" aria-pressed={seleccionado} disabled={!estado?.enabled || ocupado || seleccionado} onClick={() => cambiar(e.id)}>
            {seleccionado ? 'Escenario seleccionado' : 'Usar este escenario'}
          </button>
        </article>;
      })}</div>
      <div className="pruebas-restablecer"><div><strong>¿Terminaste la prueba?</strong><p>Volvé al comportamiento normal para los siguientes depósitos.</p></div>
        <button className="pruebas-boton-secundario" type="button" disabled={!estado?.enabled || ocupado || estado.scenario === 'accept'} onClick={() => cambiar('accept')}>
          {estado?.scenario === 'accept' ? 'Comportamiento normal activo' : 'Restablecer comportamiento normal'}
        </button>
      </div>
    </section>

    {estado?.enabled && <div className="pruebas-operacion">
      <section className="pruebas-guion" aria-labelledby="pruebas-envio-titulo">
        <p className="pruebas-eyebrow">PASO 02</p><h3 id="pruebas-envio-titulo">Enviá un depósito de prueba</h3>
        <p className="pruebas-ayuda">El depósito usará el escenario <strong>{escenarioActivo.toLowerCase()}</strong>.</p>
        <div className="pruebas-ruta"><div><small>CUENTA FICTICIA · ORIGEN</small><code>900000000003</code></div><span aria-hidden="true">→</span>
          <div><small>BANCO DE PRUEBA · DESTINO</small><code>TEST-BANK14-001</code></div></div>
        {saldo ? <div className="pruebas-saldos"><div><span>Saldo total</span><strong>{dinero(saldo.saldoCentavos)}</strong></div>
          <div><span>Reservado</span><strong>{dinero(saldo.saldoReservadoCentavos)}</strong></div>
          <div className="pruebas-disponible"><span>Disponible</span><strong>{dinero(saldo.saldoDisponibleCentavos)}</strong></div></div>
          : <p className="pruebas-aviso">Saldo no disponible todavía. Consultá Administrar cuentas antes de enviar.</p>}
        <form className="pruebas-formulario" onSubmit={e => { e.preventDefault(); void enviarDeposito(); }}>
          <label htmlFor="pruebas-monto">Monto en quetzales</label>
          <div className="pruebas-campo-monto"><span aria-hidden="true">Q</span><input id="pruebas-monto" type="number" min="0.01" max="10000" step="0.01" inputMode="decimal" value={monto} onChange={e => setMonto(e.target.value)} aria-describedby="pruebas-monto-ayuda" /></div>
          <small id="pruebas-monto-ayuda">De Q0.01 a Q10,000.00 · máximo dos decimales.</small>
          <button className="pruebas-boton-primario" type="submit" disabled={ocupado || enviando || pendiente}>
            {enviando ? 'Enviando…' : 'Enviar depósito de prueba'}
          </button>
          {pendiente && <p className="pruebas-ayuda">Esperá el resultado de la prueba actual antes de enviar otra.</p>}
        </form>
      </section>

      <section className={`pruebas-guion pruebas-resultado ${deposito && finales.includes(deposito.status) ? 'finalizado' : ''}`} data-status={deposito?.status} aria-labelledby="pruebas-resultado-titulo" aria-live="polite">
        <p className="pruebas-eyebrow">PASO 03</p><h3 id="pruebas-resultado-titulo">Resultado de la prueba</h3>
        {deposito ? <>
          <span className="pruebas-estado">{nombreEstado(deposito.status)}</span>
          <p>{explicarEstado(deposito.status)}</p>
          <dl className="pruebas-detalles"><div><dt>Estado</dt><dd><code>{deposito.status}</code></dd></div>
            {deposito.reasonCode && <div><dt>Motivo</dt><dd><code>{deposito.reasonCode}</code></dd></div>}
            <div className="pruebas-identificador"><dt>ID de la prueba</dt><dd><code>{deposito.id}</code></dd></div></dl>
          <button className="pruebas-boton-secundario" type="button" disabled={enviando} onClick={() => consultarDeposito(deposito.id)}>Actualizar resultado y saldo</button>
        </> : <div className="pruebas-vacio"><span aria-hidden="true">↗</span><strong>Tu próxima prueba aparecerá aquí</strong>
          <p>Enviá un depósito para seguir su estado y comprobar cómo se recupera el banco.</p></div>}
        {estado.grafanaUrl && <a className="pruebas-enlace" href={estado.grafanaUrl} target="_blank" rel="noopener noreferrer">Ver observabilidad en Grafana <span aria-hidden="true">↗</span></a>}
      </section>
    </div>}
  </div>;
}

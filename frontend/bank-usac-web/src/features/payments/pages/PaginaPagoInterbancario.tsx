import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAutenticacion } from '../../auth/context/ContextoAutenticacion';
import { servicioCuentas } from '../../accounts/services/servicioCuentas';
import type { Cuenta } from '../../accounts/types/cuenta';
import type { Usuario } from '../../auth/types/auth';
import { ErrorApi } from '../../../services/clienteApi';
import { servicioInterbancario, type DepositoInterbancario } from '../services/servicioInterbancario';
import './interbancario.css';

const terminales = ['ACCEPTED', 'REJECTED', 'REVERSED', 'CANCELLED'];
const escenarios = [
  ['reject', 'Rechazado', 'El banco simulado rechaza el depósito y se libera la reserva.'],
  ['unavailable', 'Banco no disponible', 'Respuesta 503 para comprobar reintentos y recuperación.'],
  ['no_callback', 'Sin callback', 'Se consulta el resultado con el mismo ID antes de decidir la compensación.'],
  ['delayed', 'Respuesta lenta', 'La confirmación supera el tiempo de espera y se comprueba la recuperación.'],
];
const dinero = (centavos: number) => new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(centavos / 100);
export function PaginaPagoInterbancario({ simulacion = false }: { simulacion?: boolean }) {
  const { usuario } = useAutenticacion();
  const navegar = useNavigate();
  const storageKey = `interbank-pending-${usuario?.clienteId}`;
  const [cuentas, setCuentas] = useState<Cuenta[]>([]);
  const [perfil, setPerfil] = useState<Usuario>();
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState('');
  const [enviando, setEnviando] = useState(false);

  const [deposito, setDeposito] = useState<DepositoInterbancario>();

  const [cuentaId, setCuentaId] = useState('');
  const [escenario, setEscenario] = useState('reject');
  const envioEnCurso = useRef(false);

  const cuenta = cuentas.find(c => c.idCuenta === cuentaId);
  const kycVerificado = perfil?.estado === 'ACTIVO' && perfil.kycStatus === 'VERIFIED';
  const pendiente = !!deposito && !terminales.includes(deposito.status);

  const cargar = useCallback(async () => {
    setCargando(true); setError('');
    try {
      const [lista, cliente] = await Promise.all([servicioCuentas.listar(), servicioInterbancario.perfil()]);
      setCuentas(lista.filter(c => c.estado === 'ACTIVA' && c.idCliente === cliente.clienteId));
      setPerfil(cliente);
    } catch { setPerfil(undefined); setError('No se pudieron consultar tus cuentas o el estado KYC. Vuelve a cargar los datos.'); }
    finally { setCargando(false); }
  }, []);

  useEffect(() => {
    void cargar();

  }, [cargar, storageKey]);

  async function enviar(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (envioEnCurso.current || pendiente || !kycVerificado) return;
    const data = new FormData(event.currentTarget);
    const monto = String(data.get('monto'));
    const destino = simulacion ? 'TEST-BANK14-001' : String(data.get('destino')).trim();
    if (!cuenta || !/^\d{1,5}(\.\d{1,2})?$/.test(monto) || Number(monto) <= 0 || Number(monto) > 10000 || !/^[A-Za-z0-9-]{1,34}$/.test(destino)) {
      setError('Selecciona una cuenta activa, un destino válido y un monto de Q0.01 a Q10,000.00.'); return;
    }
    envioEnCurso.current = true; setEnviando(true); setError('');
    let id: string | undefined;
    try {
      // Recheck current server values instead of trusting the login snapshot.
      const [actual, cliente] = await Promise.all([servicioCuentas.buscar(cuenta.idCuenta), servicioInterbancario.perfil()]);
      setPerfil(cliente);
      if (cliente.estado !== 'ACTIVO' || cliente.kycStatus !== 'VERIFIED') throw new Error('Tu cliente necesita estar activo y tener KYC verificado.');
      if (actual.estado !== 'ACTIVA' || actual.idCliente !== cliente.clienteId) throw new Error('La cuenta de origen no está activa o no te pertenece.');
      id = crypto.randomUUID();
      sessionStorage.setItem(storageKey, id);
      setDeposito({ interbankTransferId: id, status: 'ENVIANDO' });
      const resultado = await servicioInterbancario.enviar(id, {
        originAccount: actual.numeroCuenta, destinationAccount: destino,
        amount: Number(monto).toFixed(2), description: String(data.get('descripcion')).trim(),
        ...(simulacion ? { scenario: escenario } : {}),
      });
      setDeposito(resultado);
      navegar('/pagos');
      if (terminales.includes(resultado.status)) sessionStorage.removeItem(storageKey);
    } catch (e) {
      if (id) {
        // An uncertain POST is never retried automatically with a new deposit ID.
        setDeposito({ interbankTransferId: id, status: 'RESPUESTA_INCIERTA' });
        setError(`No se confirmó el envío. ID: ${id}. Consulta su estado en el historial de pagos antes de reenviar.`);
      } else { setError(e instanceof Error ? e.message : 'No se pudo validar el depósito.'); }
    } finally { envioEnCurso.current = false; setEnviando(false); }
  }

  return <div className="interbancario">
    <div className="titulo-seccion"><div><p>Procesamiento seguro</p><h2>{simulacion ? 'Escenarios interbancarios' : 'Pago interbancario'}</h2></div><Link className="boton secundario" to="/pagos">Volver a pagos</Link></div>
    <div className="interbancario-columnas">
      <section className="panel-formulario formulario-ancho">
        <span className="interbancario-distintivo">{simulacion ? 'ESCENARIOS CON MOCK' : 'DEPÓSITO AL BANCO PAREJA'}</span>
        <h3>{simulacion ? 'Elige qué comprobar' : 'Envía a una cuenta de otro banco'}</h3>
        <p>{simulacion ? 'Estos escenarios requieren tu cuenta y saldo reales. La respuesta externa será simulada.' : 'Ingresa el número de cuenta del banco pareja. El monto se reserva mientras llega la confirmación.'}</p>
        {error && <div className="alerta error" role="alert">{error}</div>}
        {!cargando && perfil && !kycVerificado && <div className="alerta" role="status">Tu cliente debe estar activo y tener KYC verificado antes de enviar un depósito.</div>}
        <form onSubmit={enviar}>
          <label>Cuenta de origen<select required value={cuentaId} onChange={e => setCuentaId(e.target.value)} disabled={cargando || enviando || pendiente}><option value="">{cargando ? 'Cargando cuentas…' : 'Selecciona tu cuenta'}</option>{cuentas.map(c => <option key={c.idCuenta} value={c.idCuenta}>{c.numeroCuenta} · {c.tipoCuenta}</option>)}</select></label>
          {!cargando && !cuentas.length && <p className="interbancario-ayuda">No tienes cuentas activas disponibles.</p>}
          {simulacion ? <fieldset className="interbancario-escenarios"><legend>Respuesta del banco simulado</legend>{escenarios.map(([id, titulo, detalle]) => <label key={id} className={escenario === id ? 'seleccionado' : ''}><input type="checkbox" name="escenario" checked={escenario === id} onChange={() => setEscenario(id)} /><span><strong>{titulo}</strong><small>{detalle}</small></span></label>)}</fieldset>
            : <label>Cuenta destino del banco pareja<input name="destino" required maxLength={34} pattern="[A-Za-z0-9-]{1,34}" placeholder="Número de cuenta del otro banco" disabled={enviando || pendiente} /></label>}
          <label>Monto en quetzales<input name="monto" required type="number" min="0.01" max="10000" step="0.01" placeholder="0.00" disabled={enviando || pendiente} /></label>
          <label>Descripción<input name="descripcion" maxLength={140} placeholder="Motivo del depósito" disabled={enviando || pendiente} /></label>
          <p className="interbancario-ayuda">Límite actual: Q10,000.00 por operación. Moneda: GTQ.</p>
          <button disabled={cargando || !kycVerificado || !cuenta || enviando || pendiente}>{enviando ? 'Enviando…' : simulacion ? 'Ejecutar escenario con mi saldo' : 'Confirmar pago interbancario'}</button>

        </form>
      </section>
      <aside className="interbancario-lateral">
        <section className="panel-formulario interbancario-resultado"><p>Tu cuenta</p><h3>{cuenta?.numeroCuenta ?? 'Selecciona una cuenta'}</h3><p>Saldo total</p><strong className="interbancario-importe">{cuenta ? dinero(cuenta.saldoCentavos) : '—'}</strong><small>El saldo disponible para enviar se valida en el servidor e incluye reservas, mínimos y comisiones.</small><span className="interbancario-distintivo">KYC: {cargando ? 'CONSULTANDO' : perfil?.kycStatus ?? 'SIN CONFIRMAR'}</span></section>

      </aside>
    </div>
  </div>;
}

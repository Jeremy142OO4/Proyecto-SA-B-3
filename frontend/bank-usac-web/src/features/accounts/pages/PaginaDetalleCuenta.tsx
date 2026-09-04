import { useCallback, useState, type FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { servicioCuentas } from '../services/servicioCuentas';

const dinero = (v: number) => new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(v / 100);

export function PaginaDetalleCuenta() {
  const { idCuenta = '' } = useParams();
  const [monto, setMonto] = useState('');
  const [mensaje, setMensaje] = useState('');
  const [errorDeposito, setErrorDeposito] = useState('');
  const [depositando, setDepositando] = useState(false);
  const consultarCuenta = useCallback(() => servicioCuentas.buscar(idCuenta), [idCuenta]);
  const consultarMovimientos = useCallback(() => servicioCuentas.movimientos(idCuenta), [idCuenta]);
  const cuenta = usarConsulta(consultarCuenta);
  const movimientos = usarConsulta(consultarMovimientos);

  async function depositar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    setMensaje(''); setErrorDeposito(''); setDepositando(true);
    try {
      await servicioCuentas.depositar(idCuenta, Math.round(Number(monto) * 100));
      setMensaje('Depósito enviado. El saldo se actualizará en unos segundos.');
      setMonto('');
    } catch (e) { setErrorDeposito(e instanceof Error ? e.message : 'No fue posible realizar el depósito'); }
    finally { setDepositando(false); }
  }

  if (cuenta.cargando) return <EstadoCarga />;
  if (cuenta.error) return <EstadoError mensaje={cuenta.error} />;
  return <>
    <div className="hero-saldo"><span>{cuenta.datos?.tipoCuenta}</span><h2>{dinero(cuenta.datos?.saldoCentavos ?? 0)}</h2><p>Cuenta •••• {cuenta.datos?.numeroCuenta.slice(-4)}</p></div>
    <div className="panel-formulario">
      <p>Operación de prueba</p><h3>Agregar fondos</h3>
      <form className="acciones" onSubmit={depositar}>
        <label>Monto en quetzales<input required min="0.01" step="0.01" type="number" value={monto} onChange={e => setMonto(e.target.value)} placeholder="100.00" /></label>
        <button disabled={depositando}>{depositando ? 'Procesando…' : 'Depositar fondos'}</button>
      </form>
      {mensaje && <div className="alerta">{mensaje}</div>}
      {errorDeposito && <div className="alerta error">{errorDeposito}</div>}
    </div>
    <div className="titulo-seccion"><h2>Movimientos recientes</h2></div>
    {movimientos.cargando ? <EstadoCarga /> : movimientos.error ? <EstadoError mensaje={movimientos.error} /> : !movimientos.datos?.length ? <EstadoVacio mensaje="Esta cuenta aún no tiene movimientos." /> : <div className="lista">{movimientos.datos.map(m => <article key={m.idMovimiento}><div><strong>{m.descripcion || m.tipoMovimiento}</strong><small>{new Date(m.fechaCreacion).toLocaleString('es-GT')}</small></div><span className={m.tipoMovimiento === 'DEBITO' ? 'negativo' : 'positivo'}>{m.tipoMovimiento === 'DEBITO' ? '-' : '+'}{dinero(m.montoCentavos)}</span></article>)}</div>}
  </>;
}

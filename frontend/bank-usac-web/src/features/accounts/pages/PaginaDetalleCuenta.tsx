import { useCallback, useState, type FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { servicioCuentas } from '../services/servicioCuentas';

const dinero = (centavos: number) => new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(centavos / 100);

export function PaginaDetalleCuenta() {
  const { idCuenta = '' } = useParams();
  const [monto, setMonto] = useState('');
  const [mensaje, setMensaje] = useState('');
  const [errorDeposito, setErrorDeposito] = useState('');
  const [depositando, setDepositando] = useState(false);
  const consultarCuenta = useCallback(() => servicioCuentas.buscar(idCuenta), [idCuenta]);
  const consultarMovimientos = useCallback(() => servicioCuentas.movimientos(idCuenta), [idCuenta]);
  const cuenta = usarConsulta(consultarCuenta, 15_000, {
    recargaSilenciosa: true, actualizarAlEnfocar: false, pausarOculta: true,
  });
  const movimientos = usarConsulta(consultarMovimientos);

  async function depositar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    setMensaje(''); setErrorDeposito(''); setDepositando(true);
    try {
      if ((await servicioCuentas.buscar(idCuenta)).estado !== 'ACTIVA') { void cuenta.recargar(); throw new Error('Esta cuenta no permite transacciones. Puedes consultar su historial.'); }
      await servicioCuentas.depositar(idCuenta, monto);
      setMensaje('Fondos agregados. El saldo se actualizará en unos segundos.');
      setMonto('');
      void cuenta.recargar();
      void movimientos.recargar();
      window.setTimeout(() => { void cuenta.recargar(); void movimientos.recargar(); }, 1000);
      window.setTimeout(() => { void cuenta.recargar(); void movimientos.recargar(); }, 2500);
    } catch (e) {
      setErrorDeposito(e instanceof Error ? e.message : 'No fue posible realizar el depósito');
    } finally {
      setDepositando(false);
    }
  }

  if (cuenta.cargando) return <EstadoCarga />;
  if (cuenta.error && !cuenta.datos) return <EstadoError mensaje={cuenta.error} />;
  return <>
    {cuenta.error && <div className="alerta error" role="status">No se pudo actualizar el saldo. Se muestran los últimos datos disponibles.</div>}
    <div className="hero-saldo">
      <span>{cuenta.datos?.tipoCuenta}</span>
      <h2>{dinero(cuenta.datos?.saldoCentavos ?? 0)}</h2>
      <p>Número de cuenta: {cuenta.datos?.numeroCuenta}</p>
      <p>ID de cuenta: {cuenta.datos?.idCuenta}</p>
    </div>
    <p className="alerta" role="status">Estado: {cuenta.datos?.estado}. {cuenta.datos?.estado === 'BLOQUEADA' && 'Cuenta bloqueada. Puedes consultar tu historial, pero no realizar transacciones.'}</p>
    {cuenta.datos?.estado === 'ACTIVA' && <div className="panel-formulario">
      <p>Gestión de cuenta</p><h3>Agregar fondos</h3>
      <form className="formulario-deposito" onSubmit={depositar}>
        <label>Monto a agregar (Q)<input required min="0.01" step="0.01" type="number" value={monto} onChange={e => setMonto(e.target.value)} placeholder="100.00" /></label>
        <button disabled={depositando}>{depositando ? 'Procesando…' : 'Agregar fondos'}</button>
      </form>
      {mensaje && <div className="alerta">{mensaje}</div>}
      {errorDeposito && <div className="alerta error">{errorDeposito}</div>}
    </div>
    }
    <div className="titulo-seccion"><h2>Movimientos recientes</h2></div>
    {movimientos.cargando ? <EstadoCarga /> : movimientos.error ? <EstadoError mensaje={movimientos.error} /> : !movimientos.datos?.length ? <EstadoVacio mensaje="Esta cuenta aún no tiene movimientos." /> : <div className="lista">{movimientos.datos.map(m => <article key={m.idMovimiento}><div><strong>{m.descripcion || m.tipoMovimiento}</strong><small>{new Date(m.fechaCreacion).toLocaleString('es-GT')}</small></div><span className={m.tipoMovimiento === 'DEBITO' ? 'negativo' : 'positivo'}>{m.tipoMovimiento === 'DEBITO' ? '-' : '+'}{dinero(m.montoCentavos)}</span></article>)}</div>}
  </>;
}

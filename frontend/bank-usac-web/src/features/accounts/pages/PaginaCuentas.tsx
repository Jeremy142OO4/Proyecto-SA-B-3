import { useCallback } from 'react';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { TarjetaCuenta } from '../components/TarjetaCuenta';
import { servicioCuentas } from '../services/servicioCuentas';

export function PaginaCuentas() {
  const consulta = useCallback(() => servicioCuentas.listar(), []);
  const { datos, cargando, actualizando, error, recargar } = usarConsulta(consulta, 15_000, {
    recargaSilenciosa: true, actualizarAlEnfocar: false, pausarOculta: true,
  });
  return <>
    <div className="titulo-seccion"><div><p>Productos activos</p><h2>Mis cuentas bancarias</h2></div><button type="button" disabled={actualizando} onClick={() => void recargar()}>{actualizando ? 'Actualizando…' : 'Actualizar saldo'}</button></div>
    {error && datos !== undefined && <div className="alerta error" role="status">No se pudo actualizar el saldo. Se muestran los últimos datos disponibles. {error}</div>}
    {cargando ? <EstadoCarga /> : error && datos === undefined ? <EstadoError mensaje={error} /> : !datos?.length ?
      <EstadoVacio mensaje="No tienes cuentas registradas." /> :
      <div className="rejilla">{datos.map(c => <TarjetaCuenta key={c.idCuenta} cuenta={c} />)}</div>}
  </>;
}

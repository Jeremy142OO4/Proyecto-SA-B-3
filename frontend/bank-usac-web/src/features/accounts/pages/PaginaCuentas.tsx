import { useCallback } from 'react';
import { EstadoCarga, EstadoError, EstadoVacio } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { TarjetaCuenta } from '../components/TarjetaCuenta';
import { servicioCuentas } from '../services/servicioCuentas';

export function PaginaCuentas() {
  const consulta = useCallback(() => servicioCuentas.listar(), []);
  const { datos, cargando, error } = usarConsulta(consulta);
  return <>
    <div className="titulo-seccion"><div><p>Productos activos</p><h2>Mis cuentas bancarias</h2></div></div>
    {cargando ? <EstadoCarga /> : error ? <EstadoError mensaje={error} /> : !datos?.length ?
      <EstadoVacio mensaje="No tienes cuentas registradas." /> :
      <div className="rejilla">{datos.map(c => <TarjetaCuenta key={c.idCuenta} cuenta={c} />)}</div>}
  </>;
}

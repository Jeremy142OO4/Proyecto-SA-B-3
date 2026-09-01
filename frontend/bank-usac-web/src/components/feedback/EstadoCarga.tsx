export function EstadoCarga(){return <div className="estado-vacio">Cargando información…</div>}
export function EstadoError({mensaje}:{mensaje:string}){return <div className="alerta error">{mensaje}</div>}
export function EstadoVacio({mensaje}:{mensaje:string}){return <div className="estado-vacio">{mensaje}</div>}

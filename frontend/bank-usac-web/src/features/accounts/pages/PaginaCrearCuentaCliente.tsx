import { FormEvent, useState } from 'react';
import { servicioCuentas } from '../services/servicioCuentas';

export function PaginaCrearCuentaCliente() {
  const [enviando, setEnviando] = useState(false);
  const [mensaje, setMensaje] = useState('');
  const [error, setError] = useState('');
  async function enviar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    if (enviando) return;
    const datos = new FormData(evento.currentTarget);
    setEnviando(true); setMensaje(''); setError('');
    try {
      const respuesta = await servicioCuentas.crear({idCliente: String(datos.get('idCliente')), tipoCuenta: String(datos.get('tipoCuenta'))});
      setMensaje(`Solicitud aceptada: ${respuesta.operationId}`);
      evento.currentTarget.reset();
    } catch (e) { setError(e instanceof Error ? e.message : 'No fue posible solicitar la cuenta'); }
    finally { setEnviando(false); }
  }
  return <div className="panel-formulario formulario-ancho">
    <p>Operación de cajero</p><h2>Crear cuenta para un cliente</h2>
    {mensaje && <div className="alerta">{mensaje}</div>}{error && <div className="alerta error">{error}</div>}
    <form onSubmit={enviar}>
      <label>Identificador del cliente<input name="idCliente" required minLength={36} maxLength={36} placeholder="UUID del cliente" /></label>
      <label>Tipo de cuenta<select name="tipoCuenta" defaultValue="MONETARIA"><option value="MONETARIA">Monetaria</option><option value="AHORRO">Ahorro</option></select></label>
      <button disabled={enviando}>{enviando ? 'Enviando…' : 'Solicitar creación'}</button>
    </form>
  </div>;
}

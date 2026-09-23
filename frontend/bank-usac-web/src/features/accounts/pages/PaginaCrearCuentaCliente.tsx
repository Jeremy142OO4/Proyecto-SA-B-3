import { FormEvent, useState } from 'react';
import { servicioCuentas } from '../services/servicioCuentas';

export function PaginaCrearCuentaCliente() {
  const [enviando, setEnviando] = useState(false);
  const [mensaje, setMensaje] = useState('');
  const [error, setError] = useState('');
  async function enviar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    if (enviando) return;
    const formulario = evento.currentTarget;
    const datos = new FormData(evento.currentTarget);
    setEnviando(true); setMensaje(''); setError('');
    try {
      const respuesta = await servicioCuentas.crear({documento: String(datos.get('documento')).trim(), tipoCuenta: String(datos.get('tipoCuenta')), saldoMinimoQuetzales: String(datos.get('saldoMinimoQuetzales') || '0'), comisionTransaccionQuetzales: String(datos.get('comisionTransaccionQuetzales') || '0')});
      setMensaje(`Solicitud aceptada: ${respuesta.operationId}`);
      formulario.reset();
    } catch (e) { setError(e instanceof Error ? e.message : 'No fue posible solicitar la cuenta'); }
    finally { setEnviando(false); }
  }
  return <div className="panel-formulario formulario-ancho">
    <p>Operación de cajero</p><h2>Crear cuenta para un cliente</h2>
    {mensaje && <div className="alerta">{mensaje}</div>}{error && <div className="alerta error">{error}</div>}
    <form onSubmit={enviar}>
      <label>DPI del cliente<input name="documento" required minLength={1} maxLength={30} placeholder="DPI del cliente" /></label>
      <label>Tipo de cuenta<select name="tipoCuenta" defaultValue="CORRIENTE"><option value="CORRIENTE">Corriente</option><option value="AHORRO">Ahorro</option><option value="MONETARIA">Monetaria (compatibilidad)</option></select></label>
      <label>Saldo mínimo (Q)<input name="saldoMinimoQuetzales" type="number" min="0" step="0.01" defaultValue="0.00" /></label>
      <label>Comisión por transacción (Q)<input name="comisionTransaccionQuetzales" type="number" min="0" step="0.01" defaultValue="0.00" /></label>
      <button disabled={enviando}>{enviando ? 'Enviando…' : 'Solicitar creación'}</button>
    </form>
  </div>;
}

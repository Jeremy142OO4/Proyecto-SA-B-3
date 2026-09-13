import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { servicioPagos } from '../services/servicioPagos';
import type { ResultadoSimulado } from '../types/pago';

export function PaginaNuevoPago() {
  const navegar = useNavigate();
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState('');
  const [tipoPago, setTipoPago] = useState<'INTERNO' | 'EXTERNO'>('INTERNO');

  async function enviar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    setEnviando(true);
    setError('');
    const formulario = new FormData(evento.currentTarget);
    try {
      await servicioPagos.crear({
        idCuentaOrigen: String(formulario.get('cuenta')),
        beneficiario: String(formulario.get('beneficiario')),
        concepto: String(formulario.get('concepto')),
        montoCentavos: Math.round(Number(formulario.get('monto')) * 100),
        tipoPago,
        resultadoSimulado: tipoPago === 'EXTERNO'
          ? String(formulario.get('resultadoSimulado')) as ResultadoSimulado
          : 'EXITO',
      });
      navegar('/pagos');
    } catch (excepcion) {
      setError(excepcion instanceof Error ? excepcion.message : 'No fue posible enviar el pago');
    } finally {
      setEnviando(false);
    }
  }

  return <div className="panel-formulario formulario-ancho">
    <p>Procesamiento seguro</p>
    <h2>Nuevo pago</h2>
    {error && <div className="alerta error">{error}</div>}
    <form onSubmit={enviar}>
      <label>Cuenta de origen<input name="cuenta" required placeholder="Identificador de la cuenta" /></label>
      <label>Beneficiario<input name="beneficiario" required maxLength={150} /></label>
      <label>Concepto<input name="concepto" required maxLength={255} /></label>
      <div className="dos-columnas">
        <label>Monto en quetzales<input name="monto" required type="number" min="0.01" step="0.01" /></label>
        <label>Tipo
          <select name="tipo" value={tipoPago} onChange={evento => setTipoPago(evento.target.value as 'INTERNO' | 'EXTERNO')}>
            <option value="INTERNO">Interno</option>
            <option value="EXTERNO">Externo</option>
          </select>
        </label>
      </div>
      {tipoPago === 'EXTERNO' && <label>Respuesta simulada del proveedor
        <select name="resultadoSimulado" defaultValue="EXITO">
          <option value="EXITO">Éxito</option>
          <option value="FALLO">Fallo</option>
          <option value="TIMEOUT">Timeout</option>
        </select>
      </label>}
      <button disabled={enviando}>{enviando ? 'Enviando…' : 'Confirmar pago'}</button>
    </form>
  </div>;
}

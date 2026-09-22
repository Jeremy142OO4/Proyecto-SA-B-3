import { useCallback, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { EstadoCarga, EstadoError } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { servicioCuentas } from '../../accounts/services/servicioCuentas';
import { servicioTransferencias } from '../services/servicioTransferencias';
import type { ResultadoExternoSimulado } from '../types/transferencia';

const etiqueta = (numero: string, saldo: number) => `•••• ${numero.slice(-4)} · ${new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(saldo / 100)}`;

export function PaginaNuevaTransferencia() {
  const navegar = useNavigate();
  const consulta = useCallback(() => servicioCuentas.listar(), []);
  const cuentas = usarConsulta(consulta);
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState('');

  async function enviar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    if (enviando) return;
    setEnviando(true);
    setError('');
    const formulario = new FormData(evento.currentTarget);
    const origen = String(formulario.get('origen'));
    const destino = String(formulario.get('destino')).trim();
    if (origen === destino) {
      setError('La cuenta destino debe ser diferente.');
      setEnviando(false);
      return;
    }
    try {
      const respuesta = await servicioTransferencias.crear({
        idCuentaOrigen: origen,
        idCuentaDestino: destino,
        montoCentavos: Math.round(Number(formulario.get('monto')) * 100),
        descripcion: String(formulario.get('descripcion')).trim(),
        resultadoExternoSimulado: String(formulario.get('resultadoExternoSimulado')) as ResultadoExternoSimulado,
      });
      navegar(`/operaciones/${respuesta.operationId}`);
    } catch (excepcion) {
      setError(excepcion instanceof Error ? excepcion.message : 'No fue posible enviar la transferencia');
    } finally {
      setEnviando(false);
    }
  }

  if (cuentas.cargando) return <EstadoCarga />;
  return <div className="panel-formulario formulario-ancho">
    <p>Transferencia bancaria</p><h2>Nueva transferencia</h2>
    {cuentas.error && <EstadoError mensaje={cuentas.error} />}{error && <div className="alerta error">{error}</div>}

    <form onSubmit={enviar}>
      <label>Cuenta de origen<select name="origen" required>{cuentas.datos?.map(cuenta => <option key={cuenta.idCuenta} value={cuenta.idCuenta}>{etiqueta(cuenta.numeroCuenta, cuenta.saldoCentavos)}</option>)}</select></label>
      <label>UUID de la cuenta destino<input name="destino" required maxLength={36} placeholder="UUID de la cuenta destino" /></label>
      <label>Monto en quetzales<input name="monto" required type="number" min="0.01" step="0.01" /></label>
      <label>Descripción<input name="descripcion" maxLength={255} placeholder="Ej. Pago de alquiler" /></label>
      <label>Resultado externo para demostración
        <select name="resultadoExternoSimulado" defaultValue="EXITO">
          <option value="EXITO">Éxito</option><option value="FALLO">Fallo</option><option value="TIMEOUT">Timeout</option>
        </select>
      </label>
      <small>La Saga validará KYC y los tipos de cuenta antes de mover fondos.</small>
      <button disabled={enviando || !cuentas.datos?.length}>{enviando ? 'Enviando…' : 'Confirmar transferencia'}</button>
    </form>
  </div>;
}

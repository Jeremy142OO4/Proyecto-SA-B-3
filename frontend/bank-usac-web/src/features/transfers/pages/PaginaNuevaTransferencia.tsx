import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '@mui/material/Button';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import WarningAmberOutlinedIcon from '@mui/icons-material/WarningAmberOutlined';
import { ErrorApi } from '../../../services/clienteApi';
import { EstadoCarga, EstadoError } from '../../../components/feedback/EstadoCarga';
import { usarConsulta } from '../../../hooks/usarConsulta';
import { useAutenticacion } from '../../auth/context/ContextoAutenticacion';
import { servicioCuentas } from '../../accounts/services/servicioCuentas';
import { servicioTransferencias } from '../services/servicioTransferencias';

const etiqueta = (numero: string, saldo: number) => `•••• ${numero.slice(-4)} · ${new Intl.NumberFormat('es-GT', { style: 'currency', currency: 'GTQ' }).format(saldo / 100)}`;

export function PaginaNuevaTransferencia() {
  const navegar = useNavigate();
  const consulta = useCallback(() => servicioCuentas.listar(), []);
  const cuentas = usarConsulta(consulta);
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState('');
  const [advertencia, setAdvertencia] = useState('');

  async function enviar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault();
    if (enviando) return;
    setError('');
    const formulario = new FormData(evento.currentTarget);
    const origen = String(formulario.get('origen'));
    const destino = String(formulario.get('destino')).trim();
    const tipoCuentaDestino = String(formulario.get('tipoCuentaDestino'));
    if (!cuentas.datos?.some(cuenta => cuenta.idCuenta === origen) || !tipoCuentaDestino) {
      setError('Selecciona la cuenta de origen y el tipo de cuenta destino.');
      return;
    }
    if (origen === destino) {
      setError('La cuenta destino debe ser diferente.');
      return;
    }
    setEnviando(true);
    setError('');
    try {
      const respuesta = await servicioTransferencias.crear({
        idCuentaOrigen: origen,
        idCuentaDestino: destino,
        tipoCuentaDestino,
        montoCentavos: Math.round(Number(formulario.get('monto')) * 100),
        descripcion: String(formulario.get('descripcion')).trim(),
      });
      navegar(`/operaciones/${respuesta.operationId}`);
    } catch (excepcion) {
      if (excepcion instanceof ErrorApi && excepcion.estado === 409) {
        setAdvertencia(excepcion.message);
      } else {
        setError(excepcion instanceof Error ? excepcion.message : 'No fue posible enviar la transferencia');
      }
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
      <label>Tipo de cuenta destino<select name="tipoCuentaDestino" required defaultValue=""><option value="" disabled>Selecciona el tipo de cuenta destino</option><option value="MONETARIA">Monetaria</option><option value="AHORRO">Ahorro</option><option value="CORRIENTE">Corriente</option></select></label>
      <label>Monto en quetzales<input name="monto" required type="number" min="0.01" step="0.01" /></label>
      <label>Descripción<input name="descripcion" maxLength={255} placeholder="Ej. Pago de alquiler" /></label>
      <button disabled={enviando || !cuentas.datos?.length}>{enviando ? 'Enviando…' : 'Confirmar transferencia'}</button>
      </form>
      <Dialog
        open={advertencia !== ''}
        onClose={() => setAdvertencia('')}
        aria-labelledby="titulo-advertencia-transferencia"
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle id="titulo-advertencia-transferencia" sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#153b62', pb: 1 }}>
          <WarningAmberOutlinedIcon sx={{ color: '#c48713', fontSize: 30 }} />
          Tipo de cuenta destino no válido
        </DialogTitle>
        <DialogContent dividers sx={{ pt: 2 }}>
          <Alert
            severity="warning"
            variant="outlined"
            icon={<WarningAmberOutlinedIcon fontSize="inherit" />}
            sx={{ alignItems: 'flex-start', borderRadius: 2 }}
          >
            <AlertTitle sx={{ fontWeight: 700 }}>El tipo seleccionado no coincide</AlertTitle>
            {advertencia}
          </Alert>
          <DialogContentText sx={{ mt: 2, color: 'text.secondary' }}>
            Verifica el UUID de la cuenta destino y selecciona el tipo de cuenta que realmente le corresponde. Las transferencias entre tipos de cuenta diferentes sí están permitidas.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setAdvertencia('')} variant="contained" autoFocus>
            Entendido
          </Button>
        </DialogActions>
      </Dialog>
  </div>;
}

export function PaginaNuevaTransferenciaProtegida() {
  const { usuario } = useAutenticacion();
  const navegar = useNavigate();
  const bloqueada = usuario?.rol === 'CLIENTE' && usuario.kycStatus !== 'VERIFIED';
  const [mostrarModal, setMostrarModal] = useState(bloqueada);

  useEffect(() => {
    setMostrarModal(bloqueada);
  }, [bloqueada]);

  if (!bloqueada) return <PaginaNuevaTransferencia />;

  return <div className="modal-kyc-fondo" role="presentation">
    {mostrarModal && <section className="modal-kyc" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-kyc">
      <div className="modal-kyc-icono">!</div>
      <p className="modal-kyc-etiqueta">Transferencia no disponible</p>
      <h2 id="titulo-modal-kyc">No es válida la realización de transferencias</h2>
      <p>Tu estado KYC todavía no está verificado. Debes tener el estado <strong>VERIFIED</strong> para realizar transferencias.</p>
      <button type="button" onClick={() => { setMostrarModal(false); navegar(-1); }}>Volver a la página anterior</button>
    </section>}
  </div>;
}

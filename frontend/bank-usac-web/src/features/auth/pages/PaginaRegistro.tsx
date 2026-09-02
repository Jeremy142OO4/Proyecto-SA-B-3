import { FormEvent, useState } from 'react';
import { ErrorApi } from '../../../services/clienteApi';
import { servicioAuth } from '../services/servicioAuth';

export function PaginaRegistro() {
  const [form, setForm] = useState({nombres: '', apellidos: '', documento: '', fotoDocumentoUrl: '', correo: '', fechaNacimiento: '', direccion: '', password: ''});
  const [error, setError] = useState(''); const [mensaje, setMensaje] = useState(''); const [cargando, setCargando] = useState(false);
  async function enviar(evento: FormEvent<HTMLFormElement>) {
    evento.preventDefault(); if (cargando) return; setError(''); setMensaje(''); setCargando(true);
    try { await servicioAuth.registrar(form); setMensaje(`Cliente registrado. Se envió la activación a ${form.correo}.`); setForm({nombres:'',apellidos:'',documento:'',fotoDocumentoUrl:'',correo:'',fechaNacimiento:'',direccion:'',password:''}); }
    catch (e) { setError(e instanceof ErrorApi ? e.message : 'No fue posible registrar al cliente'); }
    finally { setCargando(false); }
  }
  const cambiar = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setForm({...form, [e.target.name]: e.target.value});
  return <div className="panel-formulario formulario-ancho"><p>Operación de cajero</p><h2>Registrar cliente</h2>
    {mensaje && <div className="alerta">{mensaje}</div>}{error && <div className="alerta error">{error}</div>}
    <form onSubmit={enviar}>
      <div className="dos-columnas"><label>Nombres<input name="nombres" required value={form.nombres} onChange={cambiar}/></label><label>Apellidos<input name="apellidos" required value={form.apellidos} onChange={cambiar}/></label></div>
      <label>DPI<input name="documento" required value={form.documento} onChange={cambiar}/></label>
      <label>Fecha de nacimiento<input type="date" name="fechaNacimiento" required value={form.fechaNacimiento} onChange={cambiar}/></label>
      <label>Correo<input type="email" name="correo" required value={form.correo} onChange={cambiar}/></label>
      <label>URL de fotografía del documento<input name="fotoDocumentoUrl" value={form.fotoDocumentoUrl} onChange={cambiar}/></label>
      <label>Dirección<input name="direccion" required value={form.direccion} onChange={cambiar}/></label>
      <label>Contraseña temporal<input type="password" name="password" required minLength={8} value={form.password} onChange={cambiar}/></label>
      <button disabled={cargando}>{cargando ? 'Registrando…' : 'Registrar cliente'}</button>
    </form>
  </div>;
}

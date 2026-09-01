import { useEffect, useState } from 'react';
import { servicioAuditoria } from '../services/servicioAuditoria';
import { RegistroAuditoria } from '../types/auditoria';

export function PaginaAuditoria() {
  const [registros, setRegistros] = useState<RegistroAuditoria[]>([]);
  const [cargando, setCargando] = useState(true);

  useEffect(() => {
    servicioAuditoria.listar().then(setRegistros).finally(() => setCargando(false));
  }, []);

  return (
    <div className="p-6">
      <h1 className="text-xl font-bold text-white mb-4">Auditoría distribuida</h1>
      {cargando ? (
        <p className="text-slate-400">Cargando...</p>
      ) : (
        <table className="w-full text-sm text-left text-slate-300">
          <thead className="text-slate-500">
            <tr>
              <th className="p-2">Fecha</th>
              <th className="p-2">Evento</th>
              <th className="p-2">Productor</th>
              <th className="p-2">CorrelationId</th>
            </tr>
          </thead>
          <tbody>
            {registros.map((r) => (
              <tr key={r.id} className="border-t border-slate-800">
                <td className="p-2">{new Date(r.occurredAt).toLocaleString()}</td>
                <td className="p-2">{r.eventType}</td>
                <td className="p-2">{r.producer}</td>
                <td className="p-2 font-mono text-xs">{r.correlationId}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
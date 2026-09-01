import React, { useEffect, useState } from 'react';
import { Navbar } from '../components/Navbar';
import { api } from '../services/api';
import { AuditLog } from '../types';
import { ShieldAlert, RefreshCw, Eye } from 'lucide-react';

export const AuditLogs: React.FC = () => {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const res = await api.get('/api/v1/audit/logs?limit=50');
      setLogs(res.data);
    } catch (err) {
      console.error('Error cargando auditoría:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, []);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto py-10 px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center mb-6">
          <div className="flex items-center space-x-3">
            <ShieldAlert className="h-8 w-8 text-amber-500" />
            <div>
              <h1 className="text-2xl font-bold text-white">Auditoría Distribuida</h1>
              <p className="text-xs text-slate-400 font-mono">Trazabilidad SRE - notification-audit-service</p>
            </div>
          </div>

          <button
            onClick={fetchLogs}
            disabled={loading}
            className="flex items-center space-x-2 bg-slate-800 hover:bg-slate-700 px-3.5 py-2 rounded-lg text-sm transition"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            <span>Actualizar</span>
          </button>
        </div>

        {/* Tabla de Logs */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-2xl">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-800 text-left text-sm">
              <thead className="bg-slate-800/60 text-slate-400 font-mono text-xs">
                <tr>
                  <th className="px-4 py-3">Fecha (UTC)</th>
                  <th className="px-4 py-3">Tipo de Evento</th>
                  <th className="px-4 py-3">Productor</th>
                  <th className="px-4 py-3">Correlation ID</th>
                  <th className="px-4 py-3 text-right">Detalle</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800 font-mono text-xs">
                {logs.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                      No hay eventos de auditoría registrados.
                    </td>
                  </tr>
                ) : (
                  logs.map((log) => (
                    <tr key={log.id} className="hover:bg-slate-800/40 transition">
                      <td className="px-4 py-3 text-slate-400">{new Date(log.occurredAt).toLocaleString()}</td>
                      <td className="px-4 py-3">
                        <span className="bg-blue-950 text-blue-400 border border-blue-800/40 px-2 py-0.5 rounded-full">
                          {log.eventType}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-slate-300">{log.producer}</td>
                      <td className="px-4 py-3 text-slate-400">{log.correlationId}</td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => setSelectedLog(log)}
                          className="text-blue-400 hover:text-blue-300 p-1 rounded hover:bg-slate-800 transition"
                        >
                          <Eye className="h-4 w-4 inline" />
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Modal para ver el Payload JSON */}
        {selectedLog && (
          <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
            <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-2xl w-full p-6 shadow-2xl">
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-bold text-white">Detalle del Evento</h3>
                <button
                  onClick={() => setSelectedLog(null)}
                  className="text-slate-400 hover:text-white text-sm px-2 py-1 bg-slate-800 rounded"
                >
                  Cerrar
                </button>
              </div>
              <div className="space-y-2 text-xs font-mono mb-4 text-slate-300">
                <p><span className="text-slate-500">Event ID:</span> {selectedLog.eventId}</p>
                <p><span className="text-slate-500">Correlation ID:</span> {selectedLog.correlationId}</p>
                <p><span className="text-slate-500">Tipo:</span> {selectedLog.eventType}</p>
              </div>
              <div className="bg-slate-950 p-4 rounded-lg border border-slate-800 overflow-auto max-h-80 text-xs font-mono text-emerald-400">
                <pre>{JSON.stringify(selectedLog.payload, null, 2)}</pre>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
};
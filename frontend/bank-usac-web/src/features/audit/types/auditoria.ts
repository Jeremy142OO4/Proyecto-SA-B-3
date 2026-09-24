export interface RegistroAuditoria {
  id: string;
  eventId: string;
  correlationId: string;
  causationId?: string;
  eventType: string;
  severity: 'INFO' | 'WARNING' | 'ERROR';
  producer: string;
  version?: number;
  payload: unknown;
  occurredAt: string;
  recordedAt?: string;
}

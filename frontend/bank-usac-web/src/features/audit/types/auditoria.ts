export interface RegistroAuditoria {
  id: string;
  eventId: string;
  correlationId: string;
  eventType: string;
  producer: string;
  payload: unknown;
  occurredAt: string;
}
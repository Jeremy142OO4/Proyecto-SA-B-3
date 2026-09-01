export type Role = 'ADMIN' | 'TELLER' | 'CUSTOMER';
export type CustomerStatus = 'PENDING_ACTIVATION' | 'ACTIVE' | 'BLOCKED';

export interface User {
  customerId: string;
  firstName: string;
  lastName: string;
  fullName: string;
  documentId: string;
  email: string;
  username: string;
  role: Role;
  status: CustomerStatus;
  documentPhotoUrl?: string;
  address?: string;
}

export interface AuditLog {
  id: string;
  eventId: string;
  correlationId: string;
  eventType: string;
  producer: string;
  version: number;
  payload: any;
  occurredAt: string;
  recordedAt: string;
}
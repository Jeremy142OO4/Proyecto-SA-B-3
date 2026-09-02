export type EstadoTransferencia='PENDIENTE'|'PROCESANDO'|'COMPLETADA'|'RECHAZADA'|'COMPENSANDO'|'COMPENSADA'|'COMPENSACION_FALLIDA';
export interface Transferencia{idTransferencia:string;idCliente:string;idCuentaOrigen:string;idCuentaDestino:string;idCorrelacion:string;montoCentavos:number;moneda:string;descripcion?:string;estado:EstadoTransferencia;codigoError?:string;fechaCreacion:string;fechaActualizacion:string}
export interface NuevaTransferencia{idCuentaOrigen:string;idCuentaDestino:string;montoCentavos:number;descripcion?:string}
export interface OperacionAceptada{operationId:string;correlationId:string;status:string;statusUrl:string}
export interface EstadoOperacion{operationId:string;correlationId:string;type:string;status:string;updatedAt:string;errorCode?:string}

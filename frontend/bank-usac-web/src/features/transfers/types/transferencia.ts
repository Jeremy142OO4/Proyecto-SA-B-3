export type EstadoTransferencia='VALIDANDO_KYC'|'VALIDANDO_CUENTAS'|'PENDIENTE'|'PROCESANDO'|'COMPLETADA'|'RECHAZADA'|'COMPENSANDO'|'COMPENSADA'|'COMPENSACION_FALLIDA';
export type ResultadoExternoSimulado='EXITO'|'FALLO'|'TIMEOUT';
export interface Transferencia{idTransferencia:string;idCliente:string;idCuentaOrigen:string;idCuentaDestino:string;idCorrelacion:string;montoCentavos:number;moneda:string;descripcion?:string;estado:EstadoTransferencia;codigoError?:string;resultadoExternoSimulado?:ResultadoExternoSimulado;fechaCreacion:string;fechaActualizacion:string}
export interface NuevaTransferencia{idCuentaOrigen:string;idCuentaDestino:string;tipoCuentaDestino:string;montoCentavos:number;descripcion?:string;resultadoExternoSimulado?:ResultadoExternoSimulado}
export interface OperacionAceptada{operationId:string;correlationId:string;status:string;statusUrl:string}
export interface EstadoOperacion{operationId:string;correlationId:string;type:string;status:string;updatedAt:string;errorCode?:string}

export type ResultadoSimulado = 'EXITO' | 'FALLO' | 'TIMEOUT';
export interface Pago{idPago:string;beneficiario:string;concepto:string;montoCentavos:number;moneda:string;tipoPago:'INTERNO'|'EXTERNO';resultadoSimulado?:ResultadoSimulado;estado:'PENDIENTE'|'PROCESANDO'|'COMPENSANDO'|'COMPLETADO'|'RECHAZADO';motivoRechazo?:string;fechaCreacion:string}
export interface NuevoPago{idCuentaOrigen:string;beneficiario:string;concepto:string;montoCentavos:number;tipoPago:'INTERNO'|'EXTERNO';resultadoSimulado:ResultadoSimulado}

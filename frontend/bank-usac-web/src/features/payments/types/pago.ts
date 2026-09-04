export interface Pago{idPago:string;beneficiario:string;concepto:string;montoCentavos:number;moneda:string;tipoPago:'INTERNO'|'EXTERNO';estado:'PENDIENTE'|'PROCESANDO'|'COMPENSANDO'|'COMPLETADO'|'RECHAZADO';motivoRechazo?:string;fechaCreacion:string}
export interface NuevoPago{idCuentaOrigen:string;beneficiario:string;concepto:string;montoCentavos:number;tipoPago:'INTERNO'|'EXTERNO'}

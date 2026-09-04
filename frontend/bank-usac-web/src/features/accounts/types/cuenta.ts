export type EstadoCuenta='ACTIVA'|'INACTIVA'|'BLOQUEADA'|'CERRADA';
export interface Cuenta{ idCuenta:string;idCliente:string;numeroCuenta:string;tipoCuenta:'MONETARIA'|'AHORRO';saldoCentavos:number;moneda:string;estado:EstadoCuenta;ultimaActividad?:string }
export interface MovimientoCuenta{idMovimiento:string;tipoMovimiento:'DEBITO'|'CREDITO'|'COMPENSACION';montoCentavos:number;saldoNuevoCentavos:number;descripcion?:string;fechaCreacion:string}
export interface OperacionAceptada{operationId:string;correlationId:string;status:string;statusUrl:string}

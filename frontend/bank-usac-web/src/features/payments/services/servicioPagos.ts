import{clienteApi}from'../../../services/clienteApi';import type{OperacionAceptada}from'../../accounts/types/cuenta';import type{NuevoPago,Pago}from'../types/pago';
export const servicioPagos={listar:()=>clienteApi.obtener<Pago[]>('/pagos'),crear:(p:NuevoPago)=>clienteApi.publicar<OperacionAceptada>('/pagos',p)};

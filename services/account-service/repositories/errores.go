package repositories

import "errors"

var (
	ErrCuentaNoEncontrada     = errors.New("cuenta no encontrada")
	ErrCuentaNoActiva         = errors.New("la cuenta no esta activa")
	ErrFondosInsuficientes    = errors.New("fondos insuficientes")
	ErrMovimientoNoEncontrado = errors.New("movimiento original no encontrado")
	ErrSolicitudNoEncontrada  = errors.New("solicitud de cuenta no encontrada")
	ErrSolicitudNoPendiente   = errors.New("la solicitud de cuenta ya tiene un estado final")
)

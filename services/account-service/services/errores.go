package services

import "errors"

var (
	ErrClienteInvalido    = errors.New("el identificador del cliente es obligatorio")
	ErrTipoCuentaInvalido = errors.New("el tipo de cuenta no es valido")
	ErrMontoInvalido      = errors.New("el monto debe ser mayor que cero")
	ErrMensajeInvalido    = errors.New("el mensaje requiere identificadores validos")
)

package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/messaging"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
	"github.com/Proyecto-SA-B-3/account-service/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

const nombreConsumidor = "account-service.comandos"

type ConsumidorCuentas struct {
	canal             *amqp.Channel
	servicio          services.ServicioCuentas
	servicioCreacion  services.ServicioCreacionCuentas
	repositorioSalida repositories.RepositorioSalida
	publicador        *messaging.Publicador
	maximoReintentos  int
}

func NuevoConsumidorCuentas(
	conexion *amqp.Connection,
	servicio services.ServicioCuentas,
	servicioCreacion services.ServicioCreacionCuentas,
	repositorioSalida repositories.RepositorioSalida,
	publicador *messaging.Publicador,
	maximoReintentos int,
) (*ConsumidorCuentas, error) {
	canal, err := conexion.Channel()
	if err != nil {
		return nil, fmt.Errorf("crear canal consumidor: %w", err)
	}
	if err := canal.Qos(10, 0, false); err != nil {
		_ = canal.Close()
		return nil, fmt.Errorf("configurar limite del consumidor: %w", err)
	}
	return &ConsumidorCuentas{
		canal: canal, servicio: servicio, servicioCreacion: servicioCreacion, repositorioSalida: repositorioSalida,
		publicador: publicador, maximoReintentos: maximoReintentos,
	}, nil
}

func (c *ConsumidorCuentas) Cerrar() error { return c.canal.Close() }

func (c *ConsumidorCuentas) Ejecutar(ctx context.Context) error {
	entregas, err := c.canal.ConsumeWithContext(ctx, messaging.ColaComandosCuenta, nombreConsumidor, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("iniciar consumidor de cuentas: %w", err)
	}
	for entrega := range entregas {
		if err := c.procesar(ctx, entrega); err != nil {
			slog.Error("no se pudo resolver el comando", "clave", entrega.RoutingKey, "error", err)
		}
	}
	return nil
}

func (c *ConsumidorCuentas) procesar(ctx context.Context, entrega amqp.Delivery) error {
	var mensaje events.SobreMensaje
	if err := json.Unmarshal(entrega.Body, &mensaje); err != nil {
		return c.enviarFallidoYAceptar(ctx, entrega, fmt.Errorf("mensaje JSON invalido: %w", err))
	}
	if mensaje.Tipo == "" {
		mensaje.Tipo = entrega.RoutingKey
	}

	var err error
	switch entrega.RoutingKey {
	case events.ComandoConsultarCuenta:
		var solicitud events.SolicitudConsultarCuenta
		if errorJSON := json.Unmarshal(mensaje.Contenido, &solicitud); errorJSON != nil {
			return c.enviarFallidoYAceptar(ctx, entrega, errorJSON)
		}
		cuenta, errorConsulta := c.servicio.ConsultarCuenta(ctx, solicitud.IDCuenta)
		if errorConsulta != nil {
			err = errorConsulta
		} else {
			_, err = c.repositorioSalida.RegistrarRespuesta(ctx, mensaje, nombreConsumidor, events.EventoCuentaConsultada, cuenta)
		}
	case events.ComandoListarMovimientos:
		var solicitud events.SolicitudListarMovimientos
		if errorJSON := json.Unmarshal(mensaje.Contenido, &solicitud); errorJSON != nil {
			return c.enviarFallidoYAceptar(ctx, entrega, errorJSON)
		}
		lista, errorConsulta := c.servicio.ListarMovimientos(ctx, solicitud.IDCuenta, solicitud.Limite, solicitud.Desplazamiento)
		if errorConsulta != nil {
			err = errorConsulta
		} else {
			_, err = c.repositorioSalida.RegistrarRespuesta(ctx, mensaje, nombreConsumidor, events.EventoMovimientosConsultados, map[string]any{"idCuenta": solicitud.IDCuenta, "movimientos": lista})
		}
	case events.ComandoCrearCuenta:
		var solicitud events.SolicitudCrearCuenta
		if errorJSON := json.Unmarshal(mensaje.Contenido, &solicitud); errorJSON != nil {
			return c.enviarFallidoYAceptar(ctx, entrega, fmt.Errorf("contenido de creacion invalido: %w", errorJSON))
		}
		err = c.servicioCreacion.SolicitarCreacion(ctx, mensaje, solicitud)
	case events.ComandoSolicitarDebito, events.ComandoSolicitarCredito, events.ComandoSolicitarCompensacion:
		var solicitud events.SolicitudMovimiento
		if errorJSON := json.Unmarshal(mensaje.Contenido, &solicitud); errorJSON != nil {
			return c.enviarFallidoYAceptar(ctx, entrega, fmt.Errorf("contenido de movimiento invalido: %w", errorJSON))
		}
		switch entrega.RoutingKey {
		case events.ComandoSolicitarDebito:
			err = c.servicio.ProcesarDebito(ctx, mensaje, solicitud)
		case events.ComandoSolicitarCredito:
			err = c.servicio.ProcesarCredito(ctx, mensaje, solicitud)
		case events.ComandoSolicitarCompensacion:
			err = c.servicio.ProcesarCompensacion(ctx, mensaje, solicitud)
		}
	default:
		return c.enviarFallidoYAceptar(ctx, entrega, fmt.Errorf("comando no soportado: %s", entrega.RoutingKey))
	}

	if err == nil {
		return entrega.Ack(false)
	}
	if esErrorPermanente(err) {
		return c.registrarRechazoYAceptar(ctx, entrega, mensaje, err)
	}
	return c.reintentarOEnviarFallido(ctx, entrega, err)
}

func esErrorPermanente(err error) bool {
	return errors.Is(err, services.ErrMontoInvalido) ||
		errors.Is(err, services.ErrMensajeInvalido) ||
		errors.Is(err, repositories.ErrCuentaNoEncontrada) ||
		errors.Is(err, repositories.ErrCuentaNoActiva) ||
		errors.Is(err, repositories.ErrFondosInsuficientes) ||
		errors.Is(err, repositories.ErrMovimientoNoEncontrado)
}

func (c *ConsumidorCuentas) registrarRechazoYAceptar(ctx context.Context, entrega amqp.Delivery, mensaje events.SobreMensaje, causa error) error {
	tipoEvento := eventoRechazo(entrega.RoutingKey)
	contenido := map[string]any{"codigo": codigoError(causa), "mensaje": causa.Error()}
	var solicitud events.SolicitudMovimiento
	if json.Unmarshal(mensaje.Contenido, &solicitud) == nil {
		contenido["idOperacion"] = solicitud.IDOperacion
		contenido["idCuenta"] = solicitud.IDCuenta
		contenido["montoCentavos"] = solicitud.MontoCentavos
	}
	_, err := c.repositorioSalida.RegistrarRechazo(ctx, mensaje, nombreConsumidor, tipoEvento, map[string]any{
		"codigo": contenido["codigo"], "mensaje": contenido["mensaje"],
		"idOperacion": contenido["idOperacion"], "idCuenta": contenido["idCuenta"],
		"montoCentavos": contenido["montoCentavos"],
	})
	if err != nil {
		return c.reintentarOEnviarFallido(ctx, entrega, err)
	}
	return entrega.Ack(false)
}

func (c *ConsumidorCuentas) reintentarOEnviarFallido(ctx context.Context, entrega amqp.Delivery, causa error) error {
	intentos := obtenerIntentos(entrega.Headers)
	encabezados := copiarEncabezados(entrega.Headers)
	encabezados["x-intentos"] = int32(intentos + 1)
	encabezados["x-ultimo-error"] = causa.Error()

	var err error
	if intentos < c.maximoReintentos {
		err = c.publicador.ReintentarComando(ctx, entrega.RoutingKey, entrega.Body, encabezados)
	} else {
		err = c.publicador.EnviarFallido(ctx, entrega.Body, encabezados)
	}
	if err != nil {
		_ = entrega.Nack(false, true)
		return err
	}
	return entrega.Ack(false)
}

func (c *ConsumidorCuentas) enviarFallidoYAceptar(ctx context.Context, entrega amqp.Delivery, causa error) error {
	encabezados := copiarEncabezados(entrega.Headers)
	encabezados["x-ultimo-error"] = causa.Error()
	if err := c.publicador.EnviarFallido(ctx, entrega.Body, encabezados); err != nil {
		_ = entrega.Nack(false, true)
		return err
	}
	return entrega.Ack(false)
}

func eventoRechazo(comando string) string {
	switch comando {
	case events.ComandoSolicitarDebito:
		return events.EventoDebitoRechazado
	case events.ComandoSolicitarCredito:
		return events.EventoCreditoRechazado
	case events.ComandoSolicitarCompensacion:
		return events.EventoCompensacionRechazada
	default:
		return events.EventoCreacionCuentaRechazada
	}
}

func codigoError(err error) string {
	switch {
	case errors.Is(err, repositories.ErrCuentaNoEncontrada):
		return "CUENTA_NO_ENCONTRADA"
	case errors.Is(err, repositories.ErrCuentaNoActiva):
		return "CUENTA_NO_ACTIVA"
	case errors.Is(err, repositories.ErrFondosInsuficientes):
		return "FONDOS_INSUFICIENTES"
	case errors.Is(err, repositories.ErrMovimientoNoEncontrado):
		return "MOVIMIENTO_NO_ENCONTRADO"
	case errors.Is(err, services.ErrMontoInvalido):
		return "MONTO_INVALIDO"
	default:
		return "MENSAJE_INVALIDO"
	}
}

func obtenerIntentos(encabezados amqp.Table) int {
	valor, existe := encabezados["x-intentos"]
	if !existe {
		return 0
	}
	switch numero := valor.(type) {
	case int32:
		return int(numero)
	case int64:
		return int(numero)
	case int:
		return numero
	default:
		return 0
	}
}

func copiarEncabezados(origen amqp.Table) amqp.Table {
	destino := amqp.Table{}
	for clave, valor := range origen {
		destino[clave] = valor
	}
	return destino
}

package messaging

import (
	"bank-usac/service-customer/internal/telemetry"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var errSesionNoDisponible = errors.New("sesion RabbitMQ no disponible")

// sesionAMQP reconstruye conexion, publicador y consumidor como una unidad.
// No repite publicaciones ambiguas: los reintentos conservan la idempotencia
// de los mensajes persistidos por el servicio.
type sesionAMQP struct {
	mu         sync.RWMutex
	publicarMu sync.Mutex
	conexion   *amqp.Connection
	publicador *amqp.Channel
	consumidor *amqp.Channel
	cancelar   context.CancelFunc
	terminado  chan struct{}
	empezar    chan struct{}
}

func nuevaSesion(url, cola string, qos int, topologia func(*amqp.Channel) error, manejar func(context.Context, amqp.Delivery) error) (*sesionAMQP, error) {
	if _, err := amqp.ParseURI(url); err != nil {
		return nil, fmt.Errorf("URL RabbitMQ invalida")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	s := &sesionAMQP{cancelar: cancelar, terminado: make(chan struct{}), empezar: make(chan struct{})}
	go func() {
		select {
		case <-s.empezar:
			s.supervisar(ctx, url, cola, qos, topologia, manejar)
		case <-ctx.Done():
			close(s.terminado)
		}
	}()
	return s, nil
}

func (s *sesionAMQP) iniciar() { close(s.empezar) }

func (s *sesionAMQP) listo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conexion != nil && !s.conexion.IsClosed() &&
		s.publicador != nil && !s.publicador.IsClosed() &&
		s.consumidor != nil && !s.consumidor.IsClosed()
}

func (s *sesionAMQP) publicar(ctx context.Context, intercambio, clave string, mensaje amqp.Publishing) (resultErr error) {
	var finish func(error)
	ctx, mensaje.Headers, finish = telemetry.Publish(ctx, mensaje.Headers, mensaje.Body, clave)
	defer func() { finish(resultErr) }()
	ctx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	s.publicarMu.Lock()
	defer s.publicarMu.Unlock()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", errSesionNoDisponible, err)
	}
	s.mu.RLock()
	canal := s.publicador
	s.mu.RUnlock()
	if canal == nil || canal.IsClosed() {
		return errSesionNoDisponible
	}
	confirmacion, err := canal.PublishWithDeferredConfirmWithContext(ctx, intercambio, clave, false, false, mensaje)
	if err == nil && confirmacion != nil {
		var confirmado bool
		confirmado, err = confirmacion.WaitContext(ctx)
		if err == nil && !confirmado {
			err = errors.New("publicacion no confirmada")
		}
	} else if err == nil {
		err = errors.New("publicacion sin confirmacion")
	}
	if err != nil {
		// Un canal cuyo resultado es ambiguo no se reutiliza.
		s.mu.RLock()
		conexion := s.conexion
		actual := s.publicador == canal
		s.mu.RUnlock()
		if actual && conexion != nil {
			_ = conexion.CloseDeadline(time.Now().Add(time.Second))
		}
		return fmt.Errorf("%w: publicacion no confirmada", errSesionNoDisponible)
	}
	return nil
}

func (s *sesionAMQP) cerrar() {
	s.cancelar()
	s.mu.RLock()
	conexion := s.conexion
	s.mu.RUnlock()
	if conexion != nil {
		_ = conexion.CloseDeadline(time.Now().Add(time.Second))
	}
	<-s.terminado
}

func (s *sesionAMQP) supervisar(ctx context.Context, url, cola string, qos int, topologia func(*amqp.Channel) error, manejar func(context.Context, amqp.Delivery) error) {
	defer close(s.terminado)
	espera := time.Second
	for ctx.Err() == nil {
		inicio := time.Now()
		s.consumir(ctx, url, cola, qos, topologia, manejar)
		if ctx.Err() != nil {
			return
		}
		log.Printf("RabbitMQ: sesion de %s no disponible; reconectando en %s", cola, espera)
		temporizador := time.NewTimer(espera)
		select {
		case <-ctx.Done():
			temporizador.Stop()
			return
		case <-temporizador.C:
		}
		if time.Since(inicio) > 30*time.Second {
			espera = time.Second
		} else if espera < 15*time.Second {
			espera *= 2
			if espera > 15*time.Second {
				espera = 15 * time.Second
			}
		}
	}
}

func (s *sesionAMQP) consumir(ctx context.Context, url, cola string, qos int, topologia func(*amqp.Channel) error, manejar func(context.Context, amqp.Delivery) error) {
	conexion, err := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second, Dial: amqp.DefaultDial(5 * time.Second)})
	if err != nil {
		return
	}
	defer conexion.Close()
	publicador, err := conexion.Channel()
	if err != nil {
		return
	}
	defer publicador.Close()
	if err = topologia(publicador); err != nil {
		return
	}
	if err = publicador.Confirm(false); err != nil {
		return
	}
	consumidor, err := conexion.Channel()
	if err != nil {
		return
	}
	defer consumidor.Close()
	if err = consumidor.Qos(qos, 0, false); err != nil {
		return
	}
	entregas, err := consumidor.ConsumeWithContext(ctx, cola, "", false, false, false, false, nil)
	if err != nil {
		return
	}
	cierreConexion := conexion.NotifyClose(make(chan *amqp.Error, 1))
	cierrePublicador := publicador.NotifyClose(make(chan *amqp.Error, 1))
	cierreConsumidor := consumidor.NotifyClose(make(chan *amqp.Error, 1))
	s.mu.Lock()
	s.conexion, s.publicador, s.consumidor = conexion, publicador, consumidor
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.conexion, s.publicador, s.consumidor = nil, nil, nil
		s.mu.Unlock()
	}()
	log.Printf("RabbitMQ: conexion y consumidor de %s disponibles", cola)
	for {
		select {
		case <-ctx.Done():
			return
		case <-cierreConexion:
			return
		case <-cierrePublicador:
			return
		case <-cierreConsumidor:
			return
		case entrega, ok := <-entregas:
			if !ok {
				return
			}
			procesarCtx, cancelar := context.WithTimeout(ctx, 15*time.Second)
			procesarCtx, finish := telemetry.Consume(procesarCtx, entrega)
			err = manejar(procesarCtx, entrega)
			finish(err)
			cancelar()
			if err != nil {
				log.Printf("RabbitMQ: error procesando mensaje en %s: %v", cola, err)
				// Las respuestas no confirmadas se vuelven a entregar con el mismo ID.
				if entrega.Nack(false, errors.Is(err, errSesionNoDisponible)) != nil {
					return
				}
			} else if entrega.Ack(false) != nil {
				return
			}
		}
	}
}

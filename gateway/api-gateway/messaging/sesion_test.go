package messaging

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestSesionSinConexionRechazaPublicaciones(t *testing.T) {
	s, err := nuevaSesion("amqp://guest:guest@127.0.0.1:1/", "prueba", 1, func(*amqp.Channel) error { return nil }, func(context.Context, amqp.Delivery) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if s.listo() {
		t.Fatal("sesion sin conexion no debe estar lista")
	}
	if !errors.Is(s.publicar(context.Background(), "x", "k", amqp.Publishing{}), errSesionNoDisponible) {
		t.Fatal("debe rechazar publicacion desconectada")
	}
	s.iniciar()
	s.cerrar()
	select {
	case <-s.terminado:
	default:
		t.Fatal("supervisor no termino al cerrar")
	}
}

func TestSesionReconectaCanalesYConexion(t *testing.T) {
	url := os.Getenv("TEST_RABBITMQ_URL")
	if url == "" {
		t.Skip("requiere RabbitMQ de pruebas en TEST_RABBITMQ_URL")
	}
	cola := "qa.recuperacion." + uuid.NewString()
	recibidos := make(chan string, 10)
	topologia := func(ch *amqp.Channel) error {
		if err := ch.ExchangeDeclare("qa.recuperacion", "topic", false, false, false, false, nil); err != nil {
			return err
		}
		if _, err := ch.QueueDeclare(cola, true, false, false, false, nil); err != nil {
			return err
		}
		return ch.QueueBind(cola, cola, "qa.recuperacion", false, nil)
	}
	s, err := nuevaSesion(url, cola, 1, topologia, func(ctx context.Context, d amqp.Delivery) error { recibidos <- string(d.Body); return nil })
	if err != nil {
		t.Fatal(err)
	}
	s.iniciar()
	defer s.cerrar()
	esperar := func(condicion func() bool) {
		t.Helper()
		limite := time.Now().Add(35 * time.Second)
		for time.Now().Before(limite) {
			if condicion() {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatal("no se recupero la sesion RabbitMQ")
	}
	publicar := func(valor string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.publicar(ctx, "qa.recuperacion", cola, amqp.Publishing{Body: []byte(valor)}); err != nil {
			t.Fatal(err)
		}
		select {
		case recibido := <-recibidos:
			if recibido != valor {
				t.Fatalf("mensaje inesperado %s", recibido)
			}
		case <-ctx.Done():
			t.Fatal("consumidor no recuperado")
		}
	}
	esperar(s.listo)
	publicar("inicial")
	for _, recurso := range []string{"publicador", "consumidor", "conexion"} {
		s.mu.RLock()
		anterior, pub, cons := s.conexion, s.publicador, s.consumidor
		s.mu.RUnlock()
		switch recurso {
		case "publicador":
			_ = pub.Close()
		case "consumidor":
			_ = cons.Close()
		case "conexion":
			_ = anterior.Close()
		}
		esperar(func() bool {
			s.mu.RLock()
			actual := s.conexion
			s.mu.RUnlock()
			return actual != nil && actual != anterior && s.listo()
		})
		publicar("recuperado-" + recurso)
	}
	s.mu.RLock()
	conexion := s.conexion
	s.mu.RUnlock()
	limpieza, err := conexion.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer limpieza.Close()
	if _, err := limpieza.QueueDelete(cola, false, false, false); err != nil {
		t.Fatal(err)
	}
}

func TestSesionRecuperaReinicioBroker(t *testing.T) {
	url := os.Getenv("TEST_RABBITMQ_URL")
	if url == "" || os.Getenv("TEST_BROKER_RESTART") != "1" {
		t.Skip("requiere reinicio externo del broker de pruebas")
	}
	cola := "qa.reinicio." + uuid.NewString()
	recibidos := make(chan string, 2)
	topologia := func(ch *amqp.Channel) error {
		_, err := ch.QueueDeclare(cola, true, false, false, false, nil)
		return err
	}
	s, err := nuevaSesion(url, cola, 1, topologia, func(ctx context.Context, d amqp.Delivery) error { recibidos <- string(d.Body); return nil })
	if err != nil {
		t.Fatal(err)
	}
	s.iniciar()
	defer s.cerrar()
	esperar := func(condicion func() bool) {
		t.Helper()
		limite := time.Now().Add(65 * time.Second)
		for time.Now().Before(limite) {
			if condicion() {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatal("no se recupero despues del reinicio del broker")
	}
	esperar(s.listo)
	s.mu.RLock()
	anterior := s.conexion
	s.mu.RUnlock()
	t.Log("BROKER_RESTART_READY")
	esperar(func() bool { return anterior.IsClosed() })
	esperar(func() bool {
		s.mu.RLock()
		actual := s.conexion
		s.mu.RUnlock()
		return actual != nil && actual != anterior && s.listo()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.publicar(ctx, "", cola, amqp.Publishing{Body: []byte("despues-del-reinicio")}); err != nil {
		t.Fatal(err)
	}
	select {
	case valor := <-recibidos:
		if valor != "despues-del-reinicio" {
			t.Fatal(valor)
		}
	case <-ctx.Done():
		t.Fatal("mensaje no recibido despues del reinicio")
	}
	s.mu.RLock()
	conexion := s.conexion
	s.mu.RUnlock()
	ch, err := conexion.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	if _, err = ch.QueueDelete(cola, false, false, false); err != nil {
		t.Fatal(err)
	}
}

package routes

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"bank-usac/service-customer/config"
	"github.com/gofiber/fiber/v2"
)

func TestReadinessReflejaMensajeriaSinRomperLiveness(t *testing.T) {
	var disponible atomic.Bool
	app := fiber.New()
	SetupRoutes(app, nil, &config.Config{JWTSecret: "secreto-de-prueba"}, disponible.Load)
	for _, caso := range []struct {
		ruta       string
		disponible bool
		estado     int
	}{
		{"/health/ready", false, 503}, {"/health/live", false, 200}, {"/health/ready", true, 200}, {"/health/ready", false, 503},
	} {
		disponible.Store(caso.disponible)
		respuesta, err := app.Test(httptest.NewRequest("GET", caso.ruta, nil))
		if err != nil {
			t.Fatal(err)
		}
		respuesta.Body.Close()
		if respuesta.StatusCode != caso.estado {
			t.Fatalf("%s disponible=%v: estado %d, esperado %d", caso.ruta, caso.disponible, respuesta.StatusCode, caso.estado)
		}
	}
}

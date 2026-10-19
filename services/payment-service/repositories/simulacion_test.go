package repositories

import (
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
	"testing"
)

func TestSimulacionProbabilityAndLimits(t *testing.T) {
	c := models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 1000, TasaFallosPorcentaje: 37.5}
	failures := 0
	for i := 0; i < 10000; i++ {
		id := uuid.New()
		r, d := decidirSimulacion(c, id, models.ResultadoExito)
		again, _ := decidirSimulacion(c, id, models.ResultadoFallo)
		if again != r || d != 0 {
			t.Fatal("non deterministic decision")
		}
		if r == models.ResultadoFallo {
			failures++
		}
	}
	if failures < 3300 || failures > 4200 {
		t.Fatalf("unexpected 37.5 percent sample: %d", failures)
	}
	id := uuid.New()
	c.TasaFallosPorcentaje = 0
	if r, _ := decidirSimulacion(c, id, models.ResultadoFallo); r != models.ResultadoExito {
		t.Fatal("0 percent must always succeed")
	}
	c.TasaFallosPorcentaje = 100
	if r, _ := decidirSimulacion(c, id, models.ResultadoExito); r != models.ResultadoFallo {
		t.Fatal("100 percent must always fail")
	}
	c.LatenciaMs = 2000
	if r, d := decidirSimulacion(c, id, models.ResultadoExito); r != models.ResultadoTimeout || d != 1000 {
		t.Fatal("latency must hit configured deadline")
	}
	c.LatenciaMs = 0
	c.ForzarTimeout = true
	if r, d := decidirSimulacion(c, id, models.ResultadoExito); r != models.ResultadoTimeout || d != 1000 {
		t.Fatal("forced timeout")
	}
	c.Habilitada = false
	if r, d := decidirSimulacion(c, id, models.ResultadoFallo); r != models.ResultadoFallo || d != 0 {
		t.Fatal("legacy behavior must remain")
	}
}

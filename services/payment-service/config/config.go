package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Configuracion struct {
	Entorno          string
	PuertoHTTP       string
	URLBaseDatos     string
	URLRabbitMQ      string
	IntervaloOutbox  time.Duration
	MaximoReintentos int
}

func CargarConfiguracion() (Configuracion, error) {
	segundos, err := strconv.Atoi(obtener("INTERVALO_OUTBOX_SEGUNDOS", "2"))
	if err != nil || segundos <= 0 {
		return Configuracion{}, fmt.Errorf("INTERVALO_OUTBOX_SEGUNDOS invalido")
	}
	reintentos, err := strconv.Atoi(obtener("MAXIMO_REINTENTOS", "3"))
	if err != nil || reintentos < 0 {
		return Configuracion{}, fmt.Errorf("MAXIMO_REINTENTOS invalido")
	}
	c := Configuracion{Entorno: obtener("ENTORNO", "desarrollo"), PuertoHTTP: obtener("PUERTO_HTTP", "8084"), URLBaseDatos: os.Getenv("URL_BASE_DATOS"), URLRabbitMQ: os.Getenv("URL_RABBITMQ"), IntervaloOutbox: time.Duration(segundos) * time.Second, MaximoReintentos: reintentos}
	if c.URLBaseDatos == "" || c.URLRabbitMQ == "" {
		return Configuracion{}, fmt.Errorf("URL_BASE_DATOS y URL_RABBITMQ son obligatorias")
	}
	return c, nil
}
func obtener(nombre, valor string) string {
	if v := os.Getenv(nombre); v != "" {
		return v
	}
	return valor
}

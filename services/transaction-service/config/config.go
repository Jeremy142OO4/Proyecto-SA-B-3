package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Configuracion struct {
	Entorno, PuertoHTTP, URLBaseDatos, URLRabbitMQ string
	IntervaloOutbox, TiempoEsperaApagado           time.Duration
	MaximoReintentos                               int
}

func CargarConfiguracion() (Configuracion, error) {
	outbox, err := enteroPositivo("INTERVALO_OUTBOX_SEGUNDOS", 2)
	if err != nil {
		return Configuracion{}, err
	}
	espera, err := enteroPositivo("TIEMPO_ESPERA_APAGADO_SEGUNDOS", 10)
	if err != nil {
		return Configuracion{}, err
	}
	reintentos, err := strconv.Atoi(obtener("MAXIMO_REINTENTOS", "3"))
	if err != nil || reintentos < 0 {
		return Configuracion{}, fmt.Errorf("MAXIMO_REINTENTOS invalido")
	}
	c := Configuracion{Entorno: obtener("ENTORNO", "desarrollo"), PuertoHTTP: obtener("PUERTO_HTTP", "8083"), URLBaseDatos: os.Getenv("URL_BASE_DATOS"), URLRabbitMQ: os.Getenv("URL_RABBITMQ"), IntervaloOutbox: time.Duration(outbox) * time.Second, TiempoEsperaApagado: time.Duration(espera) * time.Second, MaximoReintentos: reintentos}
	if c.URLBaseDatos == "" || c.URLRabbitMQ == "" {
		return Configuracion{}, fmt.Errorf("URL_BASE_DATOS y URL_RABBITMQ son obligatorias")
	}
	return c, nil
}
func enteroPositivo(nombre string, defecto int) (int, error) {
	v, e := strconv.Atoi(obtener(nombre, strconv.Itoa(defecto)))
	if e != nil || v <= 0 {
		return 0, fmt.Errorf("%s invalido", nombre)
	}
	return v, nil
}
func obtener(n, d string) string {
	if v := os.Getenv(n); v != "" {
		return v
	}
	return d
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Configuracion struct {
	Entorno              string
	PuertoHTTP           string
	URLBaseDatos         string
	URLRabbitMQ          string
	TiempoEsperaApagado  time.Duration
	IntervaloOutbox      time.Duration
	MaximoReintentos     int
	IntervaloInactividad time.Duration
}

func CargarConfiguracion() (Configuracion, error) {
	segundosApagado, err := strconv.Atoi(obtenerVariable("TIEMPO_ESPERA_APAGADO_SEGUNDOS", "10"))
	if err != nil || segundosApagado <= 0 {
		return Configuracion{}, fmt.Errorf("TIEMPO_ESPERA_APAGADO_SEGUNDOS debe ser un entero positivo")
	}
	segundosOutbox, err := strconv.Atoi(obtenerVariable("INTERVALO_OUTBOX_SEGUNDOS", "2"))
	if err != nil || segundosOutbox <= 0 {
		return Configuracion{}, fmt.Errorf("INTERVALO_OUTBOX_SEGUNDOS debe ser un entero positivo")
	}
	maximoReintentos, err := strconv.Atoi(obtenerVariable("MAXIMO_REINTENTOS", "3"))
	if err != nil || maximoReintentos < 0 {
		return Configuracion{}, fmt.Errorf("MAXIMO_REINTENTOS debe ser un entero mayor o igual a cero")
	}
	segundosInactividad, err := strconv.Atoi(obtenerVariable("INTERVALO_INACTIVIDAD_SEGUNDOS", "86400"))
	if err != nil || segundosInactividad <= 0 {
		return Configuracion{}, fmt.Errorf("INTERVALO_INACTIVIDAD_SEGUNDOS debe ser un entero positivo")
	}

	configuracion := Configuracion{
		Entorno:              obtenerVariable("ENTORNO", "desarrollo"),
		PuertoHTTP:           obtenerVariable("PUERTO_HTTP", "8082"),
		URLBaseDatos:         os.Getenv("URL_BASE_DATOS"),
		URLRabbitMQ:          os.Getenv("URL_RABBITMQ"),
		TiempoEsperaApagado:  time.Duration(segundosApagado) * time.Second,
		IntervaloOutbox:      time.Duration(segundosOutbox) * time.Second,
		MaximoReintentos:     maximoReintentos,
		IntervaloInactividad: time.Duration(segundosInactividad) * time.Second,
	}

	if configuracion.URLBaseDatos == "" {
		return Configuracion{}, fmt.Errorf("la variable URL_BASE_DATOS es obligatoria")
	}
	if configuracion.URLRabbitMQ == "" {
		return Configuracion{}, fmt.Errorf("la variable URL_RABBITMQ es obligatoria")
	}

	return configuracion, nil
}

func obtenerVariable(nombre, valorPredeterminado string) string {
	valor := os.Getenv(nombre)
	if valor == "" {
		return valorPredeterminado
	}
	return valor
}

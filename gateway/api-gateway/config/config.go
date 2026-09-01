package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Configuracion struct {
	PuertoHTTP        string
	URLRabbitMQ       string
	SecretoJWT        string
	OrigenesCORS      string
	TiempoPublicacion time.Duration
}

func Cargar() (Configuracion, error) {
	segundos, err := strconv.Atoi(obtener("TIMEOUT_PUBLICACION_SEGUNDOS", "5"))
	if err != nil || segundos <= 0 {
		return Configuracion{}, fmt.Errorf("TIMEOUT_PUBLICACION_SEGUNDOS invalido")
	}
	c := Configuracion{
		PuertoHTTP: obtener("PUERTO_HTTP", "8080"), URLRabbitMQ: os.Getenv("URL_RABBITMQ"),
		SecretoJWT: os.Getenv("JWT_SECRET"), OrigenesCORS: obtener("CORS_ORIGINS", "http://localhost:5173"),
		TiempoPublicacion: time.Duration(segundos) * time.Second,
	}
	if c.URLRabbitMQ == "" || c.SecretoJWT == "" {
		return Configuracion{}, fmt.Errorf("URL_RABBITMQ y JWT_SECRET son obligatorias")
	}
	if len(c.SecretoJWT) < 32 {
		return Configuracion{}, fmt.Errorf("JWT_SECRET debe tener al menos 32 caracteres")
	}
	return c, nil
}

func obtener(nombre, defecto string) string {
	if valor := os.Getenv(nombre); valor != "" {
		return valor
	}
	return defecto
}

package customers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Respuesta struct {
	Estado int
	Cuerpo []byte
}

type Cliente struct {
	url  string
	http *http.Client
}

func Nuevo(url string, timeout time.Duration) *Cliente {
	return &Cliente{
		url: strings.TrimRight(url, "/"),
		http: &http.Client{Timeout: timeout},
	}
}

func (c *Cliente) Solicitar(ctx context.Context, metodo, ruta, autorizacion, correlacion string, cuerpo []byte) (Respuesta, error) {
	peticion, err := http.NewRequestWithContext(ctx, metodo, c.url+ruta, bytes.NewReader(cuerpo))
	if err != nil {
		return Respuesta{}, err
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("X-Correlation-ID", correlacion)
	if autorizacion != "" {
		peticion.Header.Set("Authorization", autorizacion)
	}
	respuesta, err := c.http.Do(peticion)
	if err != nil {
		return Respuesta{}, fmt.Errorf("customer-service no disponible: %w", err)
	}
	defer respuesta.Body.Close()
	datos, err := io.ReadAll(io.LimitReader(respuesta.Body, 2<<20))
	if err != nil {
		return Respuesta{}, err
	}
	return Respuesta{Estado: respuesta.StatusCode, Cuerpo: datos}, nil
}

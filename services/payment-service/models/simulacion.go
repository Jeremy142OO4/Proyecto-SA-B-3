package models

import "math"

type ConfiguracionSimulacion struct {
	Habilitada           bool    `json:"habilitada"`
	TasaFallosPorcentaje float64 `json:"tasaFallosPorcentaje"`
	LatenciaMs           int     `json:"latenciaMs"`
	TimeoutMs            int     `json:"timeoutMs"`
	ForzarTimeout        bool    `json:"forzarTimeout"`
	Version              int64   `json:"version"`
}

func (c ConfiguracionSimulacion) Valida() bool {
	return !math.IsNaN(c.TasaFallosPorcentaje) && !math.IsInf(c.TasaFallosPorcentaje, 0) && c.TasaFallosPorcentaje >= 0 && c.TasaFallosPorcentaje <= 100 && c.LatenciaMs >= 0 && c.LatenciaMs <= 30000 && c.TimeoutMs >= 1 && c.TimeoutMs <= 30000
}

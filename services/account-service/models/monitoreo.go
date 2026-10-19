package models

type ConfiguracionMonitoreo struct {
	MaxOperaciones int    `json:"maxOperaciones"`
	VentanaMinutos int    `json:"ventanaMinutos"`
	Version        int64  `json:"version"`
	Origen         string `json:"origen"`
}

func (c ConfiguracionMonitoreo) Valida() bool {
	return c.MaxOperaciones >= 1 && c.MaxOperaciones <= 10000 && c.VentanaMinutos >= 1 && c.VentanaMinutos <= 1440
}

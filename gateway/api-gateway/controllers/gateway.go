package controllers

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"math"
	"strconv"
	"strings"
	"time"
)

type Publicador interface {
	Publicar(context.Context, events.SobreMensaje) error
}
type Gateway struct {
	publicador  Publicador
	operaciones *operations.Store
	respuestas  *responses.Gestor
	timeout     time.Duration
	solicitante SolicitanteRPC
}

func NuevoGateway(p Publicador, o *operations.Store, r *responses.Gestor, t time.Duration, solicitantes ...SolicitanteRPC) *Gateway {
	var solicitante SolicitanteRPC
	if len(solicitantes) > 0 {
		solicitante = solicitantes[0]
	}
	return &Gateway{publicador: p, operaciones: o, respuestas: r, timeout: t, solicitante: solicitante}
}

type entradaCuenta struct {
	TipoCuenta                    string `json:"tipoCuenta"`
	Documento                     string `json:"documento"`
	SaldoMinimoQuetzales          string `json:"saldoMinimoQuetzales"`
	ComisionTransaccionQuetzales  string `json:"comisionTransaccionQuetzales"`
}
type entradaPago struct {
	IDCuentaOrigen    uuid.UUID `json:"idCuentaOrigen"`
	Beneficiario      string    `json:"beneficiario"`
	Concepto          string    `json:"concepto"`
	MontoCentavos     int64     `json:"montoCentavos"`
	TipoPago          string    `json:"tipoPago"`
	ResultadoSimulado string    `json:"resultadoSimulado"`
}
type entradaTransferencia struct {
	IDCuentaOrigen           uuid.UUID `json:"idCuentaOrigen"`
	IDCuentaDestino          uuid.UUID `json:"idCuentaDestino"`
	MontoCentavos            int64     `json:"montoCentavos"`
	Descripcion              string    `json:"descripcion"`
	ResultadoExternoSimulado string    `json:"resultadoExternoSimulado"`
}

func (g *Gateway) CrearCuenta(c *fiber.Ctx) error {
	var e entradaCuenta
	if c.BodyParser(&e) != nil {
		return fiber.NewError(400, "JSON invalido")
	}
	tipo := strings.ToUpper(strings.TrimSpace(e.TipoCuenta))
	if tipo != "MONETARIA" && tipo != "AHORRO" && tipo != "CORRIENTE" {
		return fiber.NewError(422, "tipoCuenta debe ser MONETARIA, AHORRO o CORRIENTE")
	}
	if strings.TrimSpace(e.Documento) == "" {
		return fiber.NewError(422, "documento es obligatorio")
	}
	saldoMinimoCentavos, err := quetzalesACentavos(e.SaldoMinimoQuetzales)
	if err != nil {
		return fiber.NewError(422, "saldoMinimoQuetzales debe ser un monto válido con hasta 2 decimales")
	}
	comisionTransaccionCentavos, err := quetzalesACentavos(e.ComisionTransaccionQuetzales)
	if err != nil {
		return fiber.NewError(422, "comisionTransaccionQuetzales debe ser un monto válido con hasta 2 decimales")
	}
	idCliente, err := g.buscarClientePorDPI(c, e.Documento)
	if err != nil {
		return err
	}
	if saldoMinimoCentavos < 0 || comisionTransaccionCentavos < 0 {
		return fiber.NewError(422, "las reglas de cuenta no pueden ser negativas")
	}
	id := uuid.New()
	return g.aceptar(c, events.ComandoCrearCuenta, id, events.SolicitudCrearCuenta{
		IDSolicitud: id, IDCliente: idCliente, TipoCuenta: tipo,
		SaldoMinimoCentavos: saldoMinimoCentavos,
		ComisionTransaccionCentavos: comisionTransaccionCentavos,
	})
}

func (g *Gateway) buscarClientePorDPI(c *fiber.Ctx, documento string) (uuid.UUID, error) {
	if g.solicitante == nil {
		return uuid.Nil, fiber.NewError(503, "customer-service no disponible")
	}
	corr, ok := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	if !ok || corr == uuid.Nil {
		corr = uuid.New()
	}
	respuesta, err := g.solicitante.Solicitar(c.UserContext(), events.ComandoBuscarClienteDPI, corr, events.SolicitudBuscarClienteDPI{Documento: strings.TrimSpace(documento)})
	if err != nil {
		return uuid.Nil, fiber.NewError(503, "customer-service no disponible")
	}
	if respuesta.Estado < 200 || respuesta.Estado >= 300 {
		var detalle struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respuesta.Cuerpo, &detalle)
		if detalle.Error == "" {
			detalle.Error = "cliente no encontrado"
		}
		return uuid.Nil, fiber.NewError(respuesta.Estado, detalle.Error)
	}
	var cliente struct {
		IDCliente uuid.UUID `json:"customerId"`
	}
	if err := json.Unmarshal(respuesta.Cuerpo, &cliente); err != nil || cliente.IDCliente == uuid.Nil {
		return uuid.Nil, fiber.NewError(502, "respuesta invalida de customer-service")
	}
	return cliente.IDCliente, nil
}

func quetzalesACentavos(valor string) (int64, error) {
	valor = strings.TrimSpace(strings.ReplaceAll(valor, ",", "."))
	if valor == "" {
		return 0, nil
	}
	if strings.HasPrefix(valor, "-") {
		return 0, fiber.ErrUnprocessableEntity
	}
	partes := strings.Split(valor, ".")
	if len(partes) > 2 || partes[0] == "" && len(partes) == 1 {
		return 0, fiber.ErrUnprocessableEntity
	}
	enteroTexto := partes[0]
	if enteroTexto == "" {
		enteroTexto = "0"
	}
	fraccionTexto := ""
	if len(partes) == 2 {
		fraccionTexto = partes[1]
	}
	if len(fraccionTexto) > 2 {
		return 0, fiber.ErrUnprocessableEntity
	}
	for len(fraccionTexto) < 2 {
		fraccionTexto += "0"
	}
	entero, err := strconv.ParseUint(enteroTexto, 10, 64)
	if err != nil || entero > uint64(math.MaxInt64)/100 {
		return 0, fiber.ErrUnprocessableEntity
	}
	fraccion := uint64(0)
	if fraccionTexto != "" {
		fraccion, err = strconv.ParseUint(fraccionTexto, 10, 64)
		if err != nil {
			return 0, fiber.ErrUnprocessableEntity
		}
	}
	centavos := entero*100 + fraccion
	if centavos > uint64(math.MaxInt64) {
		return 0, fiber.ErrUnprocessableEntity
	}
	return int64(centavos), nil
}
func (g *Gateway) CrearPago(c *fiber.Ctx) error {
	var e entradaPago
	if c.BodyParser(&e) != nil {
		return fiber.NewError(400, "JSON invalido")
	}
	tipo := strings.ToUpper(e.TipoPago)
	resultado := strings.ToUpper(strings.TrimSpace(e.ResultadoSimulado))
	if resultado == "" {
		resultado = "EXITO"
	}
	if e.IDCuentaOrigen == uuid.Nil || e.MontoCentavos <= 0 || strings.TrimSpace(e.Beneficiario) == "" || (tipo != "INTERNO" && tipo != "EXTERNO") {
		return fiber.NewError(422, "datos del pago invalidos")
	}
	if tipo == "EXTERNO" && resultado != "EXITO" && resultado != "FALLO" && resultado != "TIMEOUT" {
		return fiber.NewError(422, "resultadoSimulado debe ser EXITO, FALLO o TIMEOUT")
	}
	if tipo == "INTERNO" {
		resultado = "EXITO"
	}
	if err := g.validarPropiedadCuenta(c, e.IDCuentaOrigen); err != nil {
		return err
	}
	id := uuid.New()
	return g.aceptar(c, events.ComandoProcesarPago, id, events.SolicitudPago{IDPago: id, IDCliente: idCliente(c), IDCuentaOrigen: e.IDCuentaOrigen, Beneficiario: strings.TrimSpace(e.Beneficiario), Concepto: strings.TrimSpace(e.Concepto), MontoCentavos: e.MontoCentavos, TipoPago: tipo, ResultadoSimulado: resultado})
}

func (g *Gateway) Depositar(c *fiber.Ctx) error {
	idCuenta, err := uuid.Parse(c.Params("idCuenta"))
	if err != nil || idCuenta == uuid.Nil {
		return fiber.NewError(422, "idCuenta inválido")
	}
	if err := g.validarPropiedadCuenta(c, idCuenta); err != nil {
		return err
	}
	var entrada struct {
		MontoQuetzales string `json:"montoQuetzales"`
		MontoCentavos  int64  `json:"montoCentavos"`
	}
	if c.BodyParser(&entrada) != nil {
		return fiber.NewError(422, "monto inválido")
	}
	montoCentavos := entrada.MontoCentavos
	if strings.TrimSpace(entrada.MontoQuetzales) != "" {
		montoCentavos, err = quetzalesACentavos(entrada.MontoQuetzales)
		if err != nil {
			return fiber.NewError(422, "montoQuetzales debe ser un monto válido con hasta 2 decimales")
		}
	}
	if montoCentavos <= 0 {
		return fiber.NewError(422, "el monto debe ser mayor que cero")
	}
	id := uuid.New()
	return g.aceptar(c, events.ComandoDepositar, id, events.SolicitudDeposito{
		IDCuenta: idCuenta, IDOperacion: id, MontoCentavos: montoCentavos,
	})
}
func (g *Gateway) Transferir(c *fiber.Ctx) error {
	var e entradaTransferencia
	if c.BodyParser(&e) != nil {
		return fiber.NewError(400, "JSON invalido")
	}
	if e.IDCuentaOrigen == uuid.Nil || e.IDCuentaDestino == uuid.Nil || e.IDCuentaOrigen == e.IDCuentaDestino || e.MontoCentavos <= 0 {
		return fiber.NewError(422, "cuentas distintas y montoCentavos mayor que cero son obligatorios")
	}
	resultadoExterno := strings.ToUpper(strings.TrimSpace(e.ResultadoExternoSimulado))
	if resultadoExterno == "" {
		resultadoExterno = "EXITO"
	}
	if resultadoExterno != "EXITO" && resultadoExterno != "FALLO" && resultadoExterno != "TIMEOUT" {
		return fiber.NewError(422, "resultadoExternoSimulado debe ser EXITO, FALLO o TIMEOUT")
	}
	if err := g.validarPropiedadCuenta(c, e.IDCuentaOrigen); err != nil {
		return err
	}
	id := uuid.New()
	return g.aceptar(c, events.ComandoTransferir, id, events.SolicitudTransferencia{IDTransferencia: id, IDCliente: idCliente(c), IDCuentaOrigen: e.IDCuentaOrigen, IDCuentaDestino: e.IDCuentaDestino, MontoCentavos: e.MontoCentavos, Descripcion: strings.TrimSpace(e.Descripcion), ResultadoExternoSimulado: resultadoExterno})
}
func (g *Gateway) ListarCuentas(c *fiber.Ctx) error {
	return g.consultar(c, events.ComandoListarCuentas, events.EventoCuentasConsultadas, events.SolicitudHistorial{IDCliente: idCliente(c), Limite: limite(c), Desplazamiento: desplazamiento(c)}, func(b json.RawMessage) (any, error) {
		var x struct {
			IDCliente uuid.UUID       `json:"idCliente"`
			Cuentas   json.RawMessage `json:"cuentas"`
		}
		if e := json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		if x.IDCliente != idCliente(c) {
			return nil, fiber.ErrForbidden
		}
		var lista any
		if e := json.Unmarshal(x.Cuentas, &lista); e != nil {
			return nil, e
		}
		return lista, nil
	})
}
func (g *Gateway) ConsultarCuenta(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("idCuenta"))
	if e != nil {
		return fiber.NewError(400, "idCuenta invalido")
	}
	return g.consultarCuentaPropia(c, id)
}
func (g *Gateway) consultarCuentaPropia(c *fiber.Ctx, id uuid.UUID) error {
	return g.consultar(c, events.ComandoConsultarCuenta, events.EventoCuentaConsultada, events.SolicitudConsultarCuenta{IDCuenta: id}, func(b json.RawMessage) (any, error) {
		var x struct {
			IDCliente uuid.UUID `json:"idCliente"`
		}
		if e := json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		if x.IDCliente != idCliente(c) {
			return nil, fiber.ErrForbidden
		}
		var cuenta any
		return cuenta, json.Unmarshal(b, &cuenta)
	})
}
func (g *Gateway) ListarMovimientos(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("idCuenta"))
	if e != nil {
		return fiber.NewError(400, "idCuenta invalido")
	}
	if e = g.validarPropiedadCuenta(c, id); e != nil {
		return e
	}
	return g.consultar(c, events.ComandoMovimientos, events.EventoMovimientos, events.SolicitudMovimientos{IDCuenta: id, Limite: limite(c), Desplazamiento: desplazamiento(c)}, func(b json.RawMessage) (any, error) {
		var x struct {
			Movimientos any `json:"movimientos"`
		}
		e := json.Unmarshal(b, &x)
		return x.Movimientos, e
	})
}
func (g *Gateway) ListarPagos(c *fiber.Ctx) error {
	return g.consultarLista(c, events.ComandoHistorialPagos, events.EventoHistorialPagos, "pagos")
}
func (g *Gateway) ConsultarPago(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("idPago"))
	if e != nil {
		return fiber.NewError(400, "idPago invalido")
	}
	return g.consultar(c, events.ComandoConsultarPago, events.EventoPagoConsultado, events.SolicitudConsultarPago{IDPago: id}, propietario(c))
}
func (g *Gateway) ListarTransferencias(c *fiber.Ctx) error {
	solicitud := events.SolicitudHistorial{IDCliente: idCliente(c), Limite: limite(c), Desplazamiento: desplazamiento(c), Estado: strings.TrimSpace(c.Query("estado")), FechaDesde: strings.TrimSpace(c.Query("fechaDesde")), FechaHasta: strings.TrimSpace(c.Query("fechaHasta"))}
	if valor := strings.TrimSpace(c.Query("idCuenta")); valor != "" {
		id, e := uuid.Parse(valor)
		if e != nil || id == uuid.Nil {
			return fiber.NewError(400, "idCuenta invalido")
		}
		solicitud.IDCuenta = &id
	}
	return g.consultar(c, events.ComandoHistorialTransferencias, events.EventoHistorialTransferencias, solicitud, func(b json.RawMessage) (any, error) {
		var x map[string]json.RawMessage
		if e := json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		var id uuid.UUID
		if e := json.Unmarshal(x["idCliente"], &id); e != nil || id != idCliente(c) {
			return nil, fiber.ErrForbidden
		}
		var lista any
		return lista, json.Unmarshal(x["transferencias"], &lista)
	})
}
func (g *Gateway) ConsultarTransferencia(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("idTransferencia"))
	if e != nil {
		return fiber.NewError(400, "idTransferencia invalido")
	}
	return g.consultar(c, events.ComandoConsultarTransferencia, events.EventoTransferenciaConsultada, events.SolicitudConsultarTransferencia{IDTransferencia: id}, propietario(c))
}
func (g *Gateway) consultarLista(c *fiber.Ctx, cmd, evento, campo string) error {
	return g.consultar(c, cmd, evento, events.SolicitudHistorial{IDCliente: idCliente(c), Limite: limite(c), Desplazamiento: desplazamiento(c)}, func(b json.RawMessage) (any, error) {
		var x map[string]json.RawMessage
		if e := json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		var id uuid.UUID
		if e := json.Unmarshal(x["idCliente"], &id); e != nil || id != idCliente(c) {
			return nil, fiber.ErrForbidden
		}
		var lista any
		e := json.Unmarshal(x[campo], &lista)
		return lista, e
	})
}
func propietario(c *fiber.Ctx) func(json.RawMessage) (any, error) {
	return func(b json.RawMessage) (any, error) {
		var x struct {
			IDCliente uuid.UUID `json:"idCliente"`
		}
		if e := json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		if x.IDCliente != idCliente(c) {
			return nil, fiber.ErrForbidden
		}
		var v any
		return v, json.Unmarshal(b, &v)
	}
}
func (g *Gateway) validarPropiedadCuenta(c *fiber.Ctx, id uuid.UUID) error {
	_, e := g.solicitar(c, events.ComandoConsultarCuenta, events.EventoCuentaConsultada, events.SolicitudConsultarCuenta{IDCuenta: id}, func(b json.RawMessage) (any, error) {
		var x struct {
			IDCliente uuid.UUID `json:"idCliente"`
		}
		if e := json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		if x.IDCliente != idCliente(c) {
			return nil, fiber.ErrForbidden
		}
		return x, nil
	})
	return e
}
func (g *Gateway) consultar(c *fiber.Ctx, cmd, esperado string, p any, adaptar func(json.RawMessage) (any, error)) error {
	v, e := g.solicitar(c, cmd, esperado, p, adaptar)
	if e != nil {
		return e
	}
	return c.JSON(v)
}
func (g *Gateway) solicitar(c *fiber.Ctx, cmd, esperado string, p any, adaptar func(json.RawMessage) (any, error)) (any, error) {
	corr, ok := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	if !ok || corr == uuid.Nil {
		corr = uuid.New()
	}
	ch, cancel := g.respuestas.Registrar(corr)
	defer cancel()
	m, e := events.Nuevo(cmd, corr, p)
	if e != nil {
		return nil, e
	}
	ctx, fin := context.WithTimeout(c.Context(), g.timeout)
	defer fin()
	if e = g.publicador.Publicar(ctx, m); e != nil {
		return nil, fiber.NewError(503, "mensajeria no disponible")
	}
	select {
	case r := <-ch:
		if r.Tipo != esperado {
			return nil, fiber.NewError(502, "respuesta inesperada")
		}
		return adaptar(r.Contenido)
	case <-ctx.Done():
		return nil, fiber.NewError(504, "el servicio no respondio a tiempo")
	}
}
func (g *Gateway) aceptar(c *fiber.Ctx, tipo string, id uuid.UUID, p any) error {
	corr := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	m, e := events.Nuevo(tipo, corr, p)
	if e != nil {
		return e
	}
	ctx, fin := context.WithTimeout(c.Context(), g.timeout)
	defer fin()
	if e = g.publicador.Publicar(ctx, m); e != nil {
		return fiber.NewError(503, "mensajeria no disponible")
	}
	o := operations.Operacion{OperationID: id.String(), CorrelationID: corr.String(), CustomerID: idCliente(c).String(), Type: tipo, Status: "PENDIENTE", UpdatedAt: time.Now().UTC()}
	g.operaciones.Crear(o)
	return c.Status(202).JSON(fiber.Map{"operationId": o.OperationID, "correlationId": o.CorrelationID, "status": o.Status, "statusUrl": "/api/operaciones/" + o.OperationID})
}
func (g *Gateway) ConsultarOperacion(c *fiber.Ctx) error {
	o, ok := g.operaciones.Obtener(c.Params("id"))
	if !ok {
		return fiber.NewError(404, "operacion no encontrada")
	}
	if o.CustomerID != idCliente(c).String() {
		return fiber.ErrForbidden
	}
	return c.JSON(o)
}
func idCliente(c *fiber.Ctx) uuid.UUID {
	id, _ := uuid.Parse(c.Locals("customerId").(string))
	return id
}
func limite(c *fiber.Ctx) int {
	n := c.QueryInt("limite", 25)
	if n < 1 || n > 100 {
		return 25
	}
	return n
}
func desplazamiento(c *fiber.Ctx) int {
	n := c.QueryInt("desplazamiento", 0)
	if n < 0 {
		return 0
	}
	return n
}

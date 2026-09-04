package controllers

import (
	"context"
	"encoding/json"

	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/messaging"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SolicitanteRPC interface {
	Solicitar(context.Context, string, uuid.UUID, any) (messaging.RespuestaRPC, error)
}

type ControladorClientes struct{ solicitante SolicitanteRPC }

func NuevoControladorClientes(solicitante SolicitanteRPC) *ControladorClientes {
	return &ControladorClientes{solicitante: solicitante}
}

type registroCliente struct {
	Nombres          string `json:"nombres"`
	Apellidos        string `json:"apellidos"`
	Documento        string `json:"documento"`
	FotoDocumentoURL string `json:"fotoDocumentoUrl"`
	Correo           string `json:"correo"`
	FechaNacimiento  string `json:"fechaNacimiento"`
	Direccion        string `json:"direccion"`
	Password         string `json:"password"`
}

func (cc *ControladorClientes) Registrar(c *fiber.Ctx) error {
	var entrada registroCliente
	if err := c.BodyParser(&entrada); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "JSON invalido")
	}
	contenido := map[string]any{
		"firstName": entrada.Nombres, "lastName": entrada.Apellidos,
		"documentId": entrada.Documento, "documentPhotoUrl": entrada.FotoDocumentoURL,
		"email": entrada.Correo, "birthDate": entrada.FechaNacimiento,
		"address": entrada.Direccion, "password": entrada.Password,
	}
	return cc.enviar(c, events.ComandoRegistrarCliente, contenido)
}

func (cc *ControladorClientes) Activar(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token requerido")
	}
	return cc.enviar(c, events.ComandoActivarCliente, map[string]string{"token": token})
}

func (cc *ControladorClientes) Login(c *fiber.Ctx) error {
	var entrada struct {
		Usuario  string `json:"usuario"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&entrada); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "JSON invalido")
	}
	usuario := entrada.Usuario
	if usuario == "" {
		usuario = entrada.Username
	}
	if usuario == "" || entrada.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "usuario y password son requeridos")
	}
	respuesta, err := cc.solicitar(c, events.ComandoLoginCliente, map[string]string{"username": usuario, "password": entrada.Password})
	if err != nil {
		return err
	}
	if respuesta.Estado < 200 || respuesta.Estado >= 300 {
		return responderErrorCustomer(c, respuesta)
	}
	var origen struct {
		Token     string          `json:"token"`
		ExpiresAt any             `json:"expiresAt"`
		Customer  json.RawMessage `json:"customer"`
	}
	if err := json.Unmarshal(respuesta.Cuerpo, &origen); err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "respuesta invalida de customer-service")
	}
	cliente, err := adaptarCliente(origen.Customer)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "respuesta invalida de customer-service")
	}
	return c.JSON(fiber.Map{"token": origen.Token, "expiraEn": origen.ExpiresAt, "cliente": cliente})
}

func (cc *ControladorClientes) Perfil(c *fiber.Ctx) error {
	respuesta, err := cc.solicitar(c, events.ComandoPerfilCliente, map[string]any{"idCliente": idCliente(c)})
	if err != nil {
		return err
	}
	if respuesta.Estado < 200 || respuesta.Estado >= 300 {
		return responderErrorCustomer(c, respuesta)
	}
	cliente, err := adaptarCliente(respuesta.Cuerpo)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "respuesta invalida de customer-service")
	}
	return c.JSON(cliente)
}

func (cc *ControladorClientes) ActualizarPerfil(c *fiber.Ctx) error {
	var entrada struct {
		Direccion        string `json:"direccion"`
		Correo           string `json:"correo"`
		FotoDocumentoURL string `json:"fotoDocumentoUrl"`
	}
	if err := c.BodyParser(&entrada); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "JSON invalido")
	}
	contenido := map[string]any{
		"idCliente": idCliente(c),
		"address":   entrada.Direccion, "email": entrada.Correo,
		"documentPhotoUrl": entrada.FotoDocumentoURL,
	}
	respuesta, err := cc.solicitar(c, events.ComandoActualizarCliente, contenido)
	if err != nil {
		return err
	}
	if respuesta.Estado < 200 || respuesta.Estado >= 300 {
		return responderErrorCustomer(c, respuesta)
	}
	var origen struct {
		Message  string          `json:"message"`
		Customer json.RawMessage `json:"customer"`
	}
	if err := json.Unmarshal(respuesta.Cuerpo, &origen); err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "respuesta invalida de customer-service")
	}
	cliente, err := adaptarCliente(origen.Customer)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "respuesta invalida de customer-service")
	}
	return c.JSON(fiber.Map{"mensaje": origen.Message, "cliente": cliente})
}

func (cc *ControladorClientes) Listar(c *fiber.Ctx) error {
	limite := c.QueryInt("limite", 50)
	desplazamiento := c.QueryInt("desplazamiento", 0)
	if limite < 1 || limite > 100 {
		limite = 50
	}
	if desplazamiento < 0 {
		desplazamiento = 0
	}
	return cc.reenviar(c, events.ComandoListarClientes, map[string]int{"limite": limite, "desplazamiento": desplazamiento})
}

func (cc *ControladorClientes) CambiarEstado(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("idCliente"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "idCliente invalido")
	}
	var entrada struct {
		Estado string `json:"estado"`
	}
	if c.BodyParser(&entrada) != nil {
		return fiber.NewError(fiber.StatusBadRequest, "JSON invalido")
	}
	return cc.reenviar(c, events.ComandoEstadoCliente, map[string]any{"idCliente": id, "estado": entrada.Estado})
}

func (cc *ControladorClientes) reenviar(c *fiber.Ctx, tipo string, contenido any) error {
	respuesta, err := cc.solicitar(c, tipo, contenido)
	if err != nil {
		return err
	}
	c.Type("json")
	return c.Status(respuesta.Estado).Send(respuesta.Cuerpo)
}

func (cc *ControladorClientes) enviar(c *fiber.Ctx, tipo string, contenido any) error {
	respuesta, err := cc.solicitar(c, tipo, contenido)
	if err != nil {
		return err
	}
	if respuesta.Estado < 200 || respuesta.Estado >= 300 {
		return responderErrorCustomer(c, respuesta)
	}
	var datos map[string]any
	if json.Unmarshal(respuesta.Cuerpo, &datos) != nil {
		return fiber.NewError(fiber.StatusBadGateway, "respuesta invalida de customer-service")
	}
	if mensaje, ok := datos["message"]; ok {
		datos["mensaje"] = mensaje
		delete(datos, "message")
	}
	return c.Status(respuesta.Estado).JSON(datos)
}

func (cc *ControladorClientes) solicitar(c *fiber.Ctx, tipo string, contenido any) (messaging.RespuestaRPC, error) {
	correlacion, ok := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	if !ok || correlacion == uuid.Nil {
		correlacion = uuid.New()
	}
	respuesta, err := cc.solicitante.Solicitar(c.UserContext(), tipo, correlacion, contenido)
	if err != nil {
		return messaging.RespuestaRPC{}, fiber.NewError(fiber.StatusServiceUnavailable, "customer-service no disponible")
	}
	return respuesta, nil
}

func responderErrorCustomer(c *fiber.Ctx, respuesta messaging.RespuestaRPC) error {
	var origen struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(respuesta.Cuerpo, &origen)
	if origen.Error == "" {
		origen.Error = "customer-service rechazo la solicitud"
	}
	return c.Status(respuesta.Estado).JSON(fiber.Map{"mensaje": origen.Error})
}

func adaptarCliente(datos []byte) (fiber.Map, error) {
	var origen struct {
		CustomerID string `json:"customerId"`
		FullName   string `json:"fullName"`
		DocumentID string `json:"documentId"`
		Email      string `json:"email"`
		Username   string `json:"username"`
		Role       string `json:"role"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(datos, &origen); err != nil {
		return nil, err
	}
	return fiber.Map{
		"clienteId": origen.CustomerID, "nombreCompleto": origen.FullName,
		"documento": origen.DocumentID, "correo": origen.Email,
		"usuario": origen.Username, "rol": origen.Role, "estado": origen.Status,
	}, nil
}

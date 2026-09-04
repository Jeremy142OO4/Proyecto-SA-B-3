package controllers

import (
	"bank-usac/service-customer/middleware"
	"bank-usac/service-customer/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strconv"
)

type CustomerController struct {
	svc services.CustomerService
}

func (cc *CustomerController) ListCustomers(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	customers, err := cc.svc.ListCustomers(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No fue posible consultar los clientes"})
	}
	return c.JSON(customers)
}

func (cc *CustomerController) UpdateCustomerStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Identificador de cliente invalido"})
	}
	var input struct {
		Status string `json:"status"`
	}
	if c.BodyParser(&input) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cuerpo de solicitud invalido"})
	}
	customer, err := cc.svc.UpdateCustomerStatus(c.Context(), id, input.Status)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(customer)
}

func NewCustomerController(svc services.CustomerService) *CustomerController {
	return &CustomerController{svc: svc}
}

func (cc *CustomerController) Register(c *fiber.Ctx) error {
	var req services.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cuerpo de solicitud inválido"})
	}

	corrID := middleware.GetCorrelationID(c)
	customer, err := cc.svc.RegisterCustomer(c.Context(), req, corrID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Cliente registrado exitosamente. Se ha enviado un enlace de activación al correo.",
		"customer": customer,
	})
}

func (cc *CustomerController) Activate(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El parámetro token es requerido"})
	}

	corrID := middleware.GetCorrelationID(c)
	if err := cc.svc.ActivateCustomer(c.Context(), token, corrID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Cuenta activada exitosamente. Ahora puede iniciar sesión.",
	})
}

func (cc *CustomerController) Login(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Formato de credenciales inválido"})
	}

	res, err := cc.svc.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (cc *CustomerController) GetProfile(c *fiber.Ctx) error {
	customerID, ok := c.Locals("customerId").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "No autorizado"})
	}

	cust, err := cc.svc.GetProfile(c.Context(), customerID)
	if err != nil || cust == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perfil no encontrado"})
	}

	return c.Status(fiber.StatusOK).JSON(cust)
}

func (cc *CustomerController) UpdateProfile(c *fiber.Ctx) error {
	customerID, ok := c.Locals("customerId").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "No autorizado"})
	}

	var req services.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cuerpo de solicitud inválido"})
	}

	corrID := middleware.GetCorrelationID(c)
	cust, err := cc.svc.UpdateCustomer(c.Context(), customerID, req, corrID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Perfil actualizado correctamente",
		"customer": cust,
	})
}

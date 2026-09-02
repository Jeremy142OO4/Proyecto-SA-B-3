package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"bank-usac/service-customer/config"
	"bank-usac/service-customer/events"
	"bank-usac/service-customer/models"
	"bank-usac/service-customer/repositories"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CustomerService interface {
	RegisterCustomer(ctx context.Context, req RegisterRequest, correlationID uuid.UUID) (*models.Customer, error)
	ActivateCustomer(ctx context.Context, plainToken string, correlationID uuid.UUID) error
	Login(ctx context.Context, username, password string) (*LoginResponse, error)
	GetProfile(ctx context.Context, customerID uuid.UUID) (*models.Customer, error)
	UpdateCustomer(ctx context.Context, customerID uuid.UUID, req UpdateRequest, correlationID uuid.UUID) (*models.Customer, error)
	ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error)
	UpdateCustomerStatus(ctx context.Context, customerID uuid.UUID, status string) (*models.Customer, error)
}

func (s *customerService) ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *customerService) UpdateCustomerStatus(ctx context.Context, customerID uuid.UUID, status string) (*models.Customer, error) {
	estado := models.CustomerStatus(strings.ToUpper(status))
	if estado != models.StatusPendingActivation && estado != models.StatusActive && estado != models.StatusBlocked {
		return nil, errors.New("estado de cliente invalido")
	}
	cliente, err := s.repo.UpdateStatus(ctx, customerID, estado)
	if err != nil {
		return nil, err
	}
	if cliente == nil {
		return nil, errors.New("cliente no encontrado")
	}
	return cliente, nil
}

type RegisterRequest struct {
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	DocumentID       string `json:"documentId"`
	DocumentPhotoURL string `json:"documentPhotoUrl"`
	Email            string `json:"email"`
	BirthDate        string `json:"birthDate"` // YYYY-MM-DD
	Address          string `json:"address"`
	Password         string `json:"password"`
	Role             string `json:"role"`
}

type UpdateRequest struct {
	Address          string `json:"address"`
	Email            string `json:"email"`
	DocumentPhotoURL string `json:"documentPhotoUrl"`
}

type LoginResponse struct {
	Token     string           `json:"token"`
	ExpiresAt time.Time        `json:"expiresAt"`
	Customer  *models.Customer `json:"customer"`
}

type customerService struct {
	repo repositories.CustomerRepository
	cfg  *config.Config
}

func NewCustomerService(repo repositories.CustomerRepository, cfg *config.Config) CustomerService {
	return &customerService{repo: repo, cfg: cfg}
}

func (s *customerService) RegisterCustomer(ctx context.Context, req RegisterRequest, correlationID uuid.UUID) (*models.Customer, error) {
	if req.FirstName == "" || req.LastName == "" || req.DocumentID == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("todos los campos obligatorios deben ser completados")
	}

	bDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		return nil, errors.New("formato de fecha de nacimiento inválido (use YYYY-MM-DD)")
	}

	// Validar que sea mayor de 18 años
	if time.Now().AddDate(-18, 0, 0).Before(bDate) {
		return nil, errors.New("el cliente debe ser mayor de 18 años")
	}

	// Validar unicidad
	if existing, _ := s.repo.GetByDocumentID(ctx, req.DocumentID); existing != nil {
		return nil, errors.New("el documento de identificación ya se encuentra registrado")
	}
	if existing, _ := s.repo.GetByEmail(ctx, req.Email); existing != nil {
		return nil, errors.New("el correo electrónico ya se encuentra registrado")
	}

	// Generar username único
	username := s.generateUsername(req.FirstName, req.LastName)

	// Hashear contraseña
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("error al procesar la contraseña")
	}

	role := models.RoleCustomer
	if req.Role == string(models.RoleAdmin) {
		role = models.RoleAdmin
	} else if req.Role == string(models.RoleTeller) {
		role = models.RoleTeller
	}

	customerID := uuid.New()
	fullName := fmt.Sprintf("%s %s", strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName))
	now := time.Now().UTC()

	customer := &models.Customer{
		CustomerID:       customerID,
		FirstName:        strings.TrimSpace(req.FirstName),
		LastName:         strings.TrimSpace(req.LastName),
		FullName:         fullName,
		DocumentID:       strings.TrimSpace(req.DocumentID),
		DocumentPhotoURL: req.DocumentPhotoURL,
		Email:            strings.TrimSpace(strings.ToLower(req.Email)),
		BirthDate:        bDate,
		Address:          strings.TrimSpace(req.Address),
		Username:         username,
		PasswordHash:     string(hash),
		Role:             role,
		Status:           models.StatusPendingActivation,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Token de activación
	plainToken := uuid.New().String() + uuid.New().String()
	tokenHash := sha256Hex(plainToken)
	activationToken := &models.ActivationToken{
		ID:         uuid.New(),
		CustomerID: customerID,
		TokenHash:  tokenHash,
		ExpiresAt:  now.Add(24 * time.Hour),
		IsUsed:     false,
		CreatedAt:  now,
	}

	// Crear eventos para el Outbox
	custPayload := events.CustomerCreatedPayload{
		CustomerID: customer.CustomerID,
		FullName:   customer.FullName,
		Email:      customer.Email,
		Username:   customer.Username,
		DocumentID: customer.DocumentID,
		Role:       string(customer.Role),
		Status:     string(customer.Status),
	}
	envCust, _ := events.NewEnvelope(events.EventoClienteCreado, correlationID, nil, custPayload)
	envCustBytes, _ := json.Marshal(envCust)

	emailPayload := events.ActivationEmailRequestedPayload{
		CustomerID:     customer.CustomerID,
		Email:          customer.Email,
		FullName:       customer.FullName,
		ActivationLink: fmt.Sprintf("%s?token=%s", s.cfg.ActivationLinkBase, plainToken),
		ExpiresAt:      activationToken.ExpiresAt,
	}
	envEmail, _ := events.NewEnvelope(events.EventoCorreoActivacion, correlationID, nil, emailPayload)
	envEmailBytes, _ := json.Marshal(envEmail)

	outbox := []*models.OutboxMessage{
		{
			ID:            uuid.New(),
			EventType:     events.EventoClienteCreado,
			Payload:       envCustBytes,
			CorrelationID: correlationID,
			CreatedAt:     now,
		},
		{
			ID:            uuid.New(),
			EventType:     events.EventoCorreoActivacion,
			Payload:       envEmailBytes,
			CorrelationID: correlationID,
			CreatedAt:     now,
		},
	}

	if err := s.repo.CreateWithOutbox(ctx, customer, activationToken, outbox); err != nil {
		return nil, fmt.Errorf("error al registrar cliente: %w", err)
	}

	return customer, nil
}

func (s *customerService) ActivateCustomer(ctx context.Context, plainToken string, correlationID uuid.UUID) error {
	tokenHash := sha256Hex(plainToken)
	tokenRecord, err := s.repo.FindActivationToken(ctx, tokenHash)
	if err != nil || tokenRecord == nil {
		return errors.New("el token de activación no existe")
	}

	if tokenRecord.IsUsed {
		return errors.New("este enlace de activación ya fue utilizado previamente")
	}

	if time.Now().UTC().After(tokenRecord.ExpiresAt) {
		return errors.New("el enlace de activación ha expirado")
	}

	// Preparar evento de activación
	actPayload := events.CustomerActivatedPayload{
		CustomerID:  tokenRecord.CustomerID,
		ActivatedAt: time.Now().UTC(),
	}
	env, _ := events.NewEnvelope(events.EventoClienteActivado, correlationID, nil, actPayload)
	envBytes, _ := json.Marshal(env)

	outboxEvent := &models.OutboxMessage{
		ID:            uuid.New(),
		EventType:     events.EventoClienteActivado,
		Payload:       envBytes,
		CorrelationID: correlationID,
		CreatedAt:     time.Now().UTC(),
	}

	return s.repo.ActivateCustomer(ctx, tokenRecord.CustomerID, tokenRecord.ID, outboxEvent)
}

func (s *customerService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	customer, err := s.repo.GetByUsername(ctx, username)
	if err != nil || customer == nil {
		return nil, errors.New("credenciales inválidas")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("credenciales inválidas")
	}

	if customer.Status == models.StatusPendingActivation {
		return nil, errors.New("la cuenta aún no ha sido activada. Por favor revisa tu correo")
	}

	if customer.Status == models.StatusBlocked {
		return nil, errors.New("la cuenta se encuentra bloqueada. Contacte con soporte")
	}

	exp := time.Now().Add(time.Duration(s.cfg.JWTExpirationHours) * time.Hour)
	claims := jwt.MapClaims{
		"sub":      customer.CustomerID.String(),
		"username": customer.Username,
		"email":    customer.Email,
		"role":     customer.Role,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, errors.New("error al generar token de autenticación")
	}

	return &LoginResponse{
		Token:     tokenString,
		ExpiresAt: exp,
		Customer:  customer,
	}, nil
}

func (s *customerService) GetProfile(ctx context.Context, customerID uuid.UUID) (*models.Customer, error) {
	return s.repo.GetByID(ctx, customerID)
}

func (s *customerService) UpdateCustomer(ctx context.Context, customerID uuid.UUID, req UpdateRequest, correlationID uuid.UUID) (*models.Customer, error) {
	cust, err := s.repo.GetByID(ctx, customerID)
	if err != nil || cust == nil {
		return nil, errors.New("cliente no encontrado")
	}

	if req.Address != "" {
		cust.Address = req.Address
	}
	if req.DocumentPhotoURL != "" {
		cust.DocumentPhotoURL = req.DocumentPhotoURL
	}
	if req.Email != "" && req.Email != cust.Email {
		existing, _ := s.repo.GetByEmail(ctx, req.Email)
		if existing != nil && existing.CustomerID != cust.CustomerID {
			return nil, errors.New("el correo electrónico ya está en uso")
		}
		cust.Email = req.Email
	}

	if err := s.repo.Update(ctx, cust); err != nil {
		return nil, err
	}

	return cust, nil
}

func (s *customerService) generateUsername(firstName, lastName string) string {
	clean := func(str string) string {
		reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
		return strings.ToLower(reg.ReplaceAllString(str, ""))
	}
	fn := clean(firstName)
	ln := clean(lastName)
	if len(fn) > 3 {
		fn = fn[:3]
	}
	base := fmt.Sprintf("%s%s", fn, ln)
	// sufijo aleatorio para evitar colisiones
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s%03d", base, r.Intn(999))
}

func sha256Hex(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

package repositories

import (
	"bank-usac/service-customer/internal/telemetry"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type customerRepo struct {
	db *sqlx.DB
}

func NewCustomerRepository(db *sqlx.DB) CustomerRepository {
	return &customerRepo{db: db}
}

func (r *customerRepo) CreateWithOutbox(ctx context.Context, customer *models.Customer, token *models.ActivationToken, outboxEvents []*models.OutboxMessage) error {
	correlationID := uuid.New()
	if len(outboxEvents) > 0 && outboxEvents[0].CorrelationID != uuid.Nil {
		correlationID = outboxEvents[0].CorrelationID
	}
	ctx, lifecycleSpan := telemetry.StartCustomerLifecycle(ctx, "", "customer.lifecycle.created", correlationID)
	defer lifecycleSpan.End()
	customer.LifecycleCorrelationID = &correlationID
	customer.LifecycleTraceContext = telemetry.Serialize(ctx)
	customer.LifecycleObservedFromCreation = true

	ctx, finishDatabase := telemetry.Database(ctx, "CreateWithOutbox")
	defer finishDatabase()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryCustomer := `
		INSERT INTO customers (customer_id, first_name, last_name, full_name, document_id, document_photo_url, email, birth_date, address, username, password_hash, role, status, created_at, updated_at)
		VALUES (:customer_id, :first_name, :last_name, :full_name, :document_id, :document_photo_url, :email, :birth_date, :address, :username, :password_hash, :role, :status, :created_at, :updated_at)
	`
	if _, err := tx.NamedExecContext(ctx, queryCustomer, customer); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE customers SET lifecycle_correlation_id=$1,lifecycle_trace_context=$2,lifecycle_observed_from_creation=TRUE WHERE customer_id=$3`, correlationID, customer.LifecycleTraceContext, customer.CustomerID); err != nil {
		return err
	}
	if err = saveLifecycle(ctx, tx, customer, nil, events.EventoClienteCreado, "CREATION", "", string(customer.Status)); err != nil {
		return err
	}
	if token != nil {
		queryToken := `
			INSERT INTO activation_tokens (id, customer_id, token_hash, expires_at, is_used, created_at)
			VALUES (:id, :customer_id, :token_hash, :expires_at, :is_used, :created_at)
		`
		if _, err := tx.NamedExecContext(ctx, queryToken, token); err != nil {
			return err
		}
	}

	queryOutbox := `
		INSERT INTO outbox_messages (id, event_type, payload, correlation_id, causation_id, created_at,trace_context)
		VALUES ($1, $2, $3, $4, $5, $6,$7)
	`
	for _, ev := range outboxEvents {
		if _, err := tx.ExecContext(ctx, queryOutbox, ev.ID, ev.EventType, ev.Payload, ev.CorrelationID, ev.CausationID, ev.CreatedAt, telemetry.Serialize(ctx)); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	telemetry.CustomerLifecycleCommitted(ctx, customer.CustomerID, events.EventoClienteCreado, "CREATION", "", string(customer.Status), nil)
	return nil
}

func (r *customerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Customer, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "GetByID")
	defer finishDatabase()
	var c models.Customer
	err := r.db.GetContext(ctx, &c, "SELECT * FROM customers WHERE customer_id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *customerRepo) GetByUsername(ctx context.Context, username string) (*models.Customer, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "GetByUsername")
	defer finishDatabase()
	var c models.Customer
	err := r.db.GetContext(ctx, &c, "SELECT * FROM customers WHERE username = $1", username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *customerRepo) GetByEmail(ctx context.Context, email string) (*models.Customer, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "GetByEmail")
	defer finishDatabase()
	var c models.Customer
	err := r.db.GetContext(ctx, &c, "SELECT * FROM customers WHERE email = $1", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *customerRepo) GetByDocumentID(ctx context.Context, docID string) (*models.Customer, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "GetByDocumentID")
	defer finishDatabase()
	var c models.Customer
	err := r.db.GetContext(ctx, &c, "SELECT * FROM customers WHERE document_id = $1", docID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *customerRepo) FindActivationToken(ctx context.Context, tokenHash string) (*models.ActivationToken, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "FindActivationToken")
	defer finishDatabase()
	var t models.ActivationToken
	query := "SELECT * FROM activation_tokens WHERE token_hash = $1"
	err := r.db.GetContext(ctx, &t, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (r *customerRepo) IsMessageProcessed(ctx context.Context, messageID uuid.UUID) (bool, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "IsMessageProcessed")
	defer finishDatabase()
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(1) FROM processed_messages WHERE message_id = $1", messageID)
	return count > 0, err
}

func (r *customerRepo) MarkMessageProcessed(ctx context.Context, tx *sqlx.Tx, messageID uuid.UUID, consumerName, ref string) error {
	ctx, finishDatabase := telemetry.Database(ctx, "MarkMessageProcessed")
	defer finishDatabase()
	query := `INSERT INTO processed_messages (message_id, consumer_name, result_reference)
		VALUES ($1, $2, $3) ON CONFLICT (message_id) DO NOTHING`
	if tx != nil {
		_, err := tx.ExecContext(ctx, query, messageID, consumerName, ref)
		return err
	}
	_, err := r.db.ExecContext(ctx, query, messageID, consumerName, ref)
	return err
}

func (r *customerRepo) GetProcessedMessageResult(ctx context.Context, messageID uuid.UUID) (bool, string, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "GetProcessedMessageResult")
	defer finishDatabase()
	var result string
	err := r.db.GetContext(ctx, &result, "SELECT result_reference FROM processed_messages WHERE message_id = $1", messageID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	return err == nil, result, err
}

func (r *customerRepo) RecordProcessedMessage(ctx context.Context, messageID uuid.UUID, consumerName, ref string) error {
	ctx, finishDatabase := telemetry.Database(ctx, "RecordProcessedMessage")
	defer finishDatabase()
	_, err := r.db.ExecContext(ctx, `INSERT INTO processed_messages
		(message_id, consumer_name, result_reference) VALUES ($1,$2,$3)
		ON CONFLICT (message_id) DO NOTHING`, messageID, consumerName, ref)
	return err
}

func (r *customerRepo) GetPendingOutbox(ctx context.Context, limit int) ([]*models.OutboxMessage, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "GetPendingOutbox")
	defer finishDatabase()
	var msgs []*models.OutboxMessage
	query := "SELECT * FROM outbox_messages WHERE published_at IS NULL ORDER BY created_at ASC LIMIT $1"
	err := r.db.SelectContext(ctx, &msgs, query, limit)
	return msgs, err
}

func (r *customerRepo) MarkOutboxPublished(ctx context.Context, id uuid.UUID) error {
	ctx, finishDatabase := telemetry.Database(ctx, "MarkOutboxPublished")
	defer finishDatabase()
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, "UPDATE outbox_messages SET published_at = $1 WHERE id = $2", now, id)
	return err
}

func (r *customerRepo) IncrementOutboxAttempt(ctx context.Context, id uuid.UUID, errStr string) error {
	ctx, finishDatabase := telemetry.Database(ctx, "IncrementOutboxAttempt")
	defer finishDatabase()
	_, err := r.db.ExecContext(ctx, "UPDATE outbox_messages SET attempts = attempts + 1, last_error = $1 WHERE id = $2", errStr, id)
	return err
}

func (r *customerRepo) List(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	ctx, finishDatabase := telemetry.Database(ctx, "List")
	defer finishDatabase()
	customers := make([]*models.Customer, 0)
	err := r.db.SelectContext(ctx, &customers, `SELECT * FROM customers ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	return customers, err
}

func (r *customerRepo) RegistrarValidacionCliente(ctx context.Context, mensajeID, correlacionID uuid.UUID, solicitudID, clienteID uuid.UUID) error {
	ctx, finishDatabase := telemetry.Database(ctx, "RegistrarValidacionCliente")
	defer finishDatabase()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var procesado bool
	if err = tx.GetContext(ctx, &procesado, "SELECT EXISTS(SELECT 1 FROM processed_messages WHERE message_id=$1)", mensajeID); err != nil {
		return err
	}
	if procesado {
		return tx.Commit()
	}

	var estado string
	err = tx.GetContext(ctx, &estado, "SELECT status FROM customers WHERE customer_id=$1", clienteID)
	activo := err == nil && estado == string(models.StatusActive)
	motivo := ""
	if errors.Is(err, sql.ErrNoRows) {
		motivo = "cliente no encontrado"
	} else if err != nil {
		return err
	} else if !activo {
		motivo = "cliente no activo"
	}
	tipo := events.EventoClienteValidado
	if !activo {
		tipo = events.EventoClienteRechazado
	}
	resultado := events.ResultadoValidacionCliente{IDSolicitud: solicitudID, IDCliente: clienteID, Activo: activo, Motivo: motivo}
	sobre, err := events.NewEnvelope(tipo, correlacionID, &mensajeID, resultado)
	if err != nil {
		return err
	}
	contenido, err := json.Marshal(sobre)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO outbox_messages
		(id,event_type,payload,correlation_id,causation_id,created_at,trace_context)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), tipo, contenido, correlacionID, mensajeID, time.Now().UTC(), telemetry.Serialize(ctx)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO processed_messages
		(message_id,consumer_name,result_reference) VALUES($1,$2,$3)
		ON CONFLICT (message_id) DO NOTHING`, mensajeID, "customer-service.validacion-cliente", solicitudID.String()); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *customerRepo) RegistrarValidacionKYC(ctx context.Context, mensajeID, correlacionID, operacionID, clienteID uuid.UUID) error {
	ctx, finishDatabase := telemetry.Database(ctx, "RegistrarValidacionKYC")
	defer finishDatabase()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var procesado bool
	if err = tx.GetContext(ctx, &procesado, `SELECT EXISTS(SELECT 1 FROM processed_messages WHERE message_id=$1 AND consumer_name='customer-service.validacion-kyc')`, mensajeID); err != nil {
		return err
	}
	if procesado {
		return tx.Commit()
	}
	var acceso models.CustomerStatus
	var kyc models.KYCStatus
	err = tx.QueryRowContext(ctx, `SELECT status,kyc_status FROM customers WHERE customer_id=$1`, clienteID).Scan(&acceso, &kyc)
	valido := err == nil && acceso == models.StatusActive && kyc == models.KYCVerified
	motivo := ""
	if errors.Is(err, sql.ErrNoRows) {
		motivo = "cliente no encontrado"
	} else if err != nil {
		return err
	} else if acceso != models.StatusActive {
		motivo = "cliente no activo"
	} else if kyc != models.KYCVerified {
		motivo = "KYC no verificado: " + string(kyc)
	}
	tipo := events.EventoKYCVerificado
	if !valido {
		tipo = events.EventoKYCRechazado
	}
	resultado := events.ResultadoValidacionKYC{IDOperacion: operacionID, IDCliente: clienteID, EstadoKYC: string(kyc), Valido: valido, Motivo: motivo}
	sobre, err := events.NewEnvelope(tipo, correlacionID, &mensajeID, resultado)
	if err != nil {
		return err
	}
	contenido, err := json.Marshal(sobre)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO outbox_messages(id,event_type,payload,correlation_id,causation_id,created_at,trace_context) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), tipo, contenido, correlacionID, mensajeID, time.Now().UTC(), telemetry.Serialize(ctx)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO processed_messages(message_id,consumer_name,result_reference) VALUES($1,'customer-service.validacion-kyc',$2)
		ON CONFLICT (message_id) DO NOTHING`, mensajeID, operacionID.String()); err != nil {
		return err
	}
	return tx.Commit()
}

package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/internal/telemetry"
	"bank-usac/service-customer/models"
	"github.com/google/uuid"
)

func (r *customerRepo) UpdateKYCStatusWithOutbox(ctx context.Context, id uuid.UUID, status models.KYCStatus, event *models.OutboxMessage) (*models.Customer, error) {
	if status != models.KYCPending && status != models.KYCVerified && status != models.KYCRejected {
		return nil, errors.New("estado KYC invalido")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var customer models.Customer
	err = tx.GetContext(ctx, &customer, `SELECT * FROM customers WHERE customer_id=$1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	previous := customer.KYCStatus
	// Row locking makes repeated or concurrent identical decisions a no-op.
	if previous == status {
		return &customer, tx.Commit()
	}
	if err = ensureLifecycle(ctx, tx, &customer, event); err != nil {
		return nil, err
	}
	ctx, span := telemetry.StartCustomerLifecycle(ctx, customer.LifecycleTraceContext, "customer.lifecycle.kyc", *customer.LifecycleCorrelationID)
	defer span.End()
	ctx, finishDatabase := telemetry.Database(ctx, "UpdateKYCStatusWithOutbox")
	defer finishDatabase()
	firstVerification := status == models.KYCVerified && customer.KYCVerifiedAt == nil && customer.LifecycleObservedFromCreation
	err = tx.GetContext(ctx, &customer, `UPDATE customers SET kyc_status=$1::varchar,
		updated_at=clock_timestamp(),kyc_updated_at=clock_timestamp(),
		kyc_verified_at=CASE WHEN $1::varchar='VERIFIED' AND lifecycle_observed_from_creation
		THEN COALESCE(kyc_verified_at,clock_timestamp()) ELSE kyc_verified_at END
		WHERE customer_id=$2 RETURNING *`, status, id)
	if err != nil {
		return nil, err
	}
	if err = saveLifecycle(ctx, tx, &customer, event, events.EventoKYCActualizado, "KYC", string(previous), string(status)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	var duration *time.Duration
	if firstVerification && customer.KYCVerifiedAt != nil {
		elapsed := customer.KYCVerifiedAt.Sub(customer.CreatedAt)
		duration = &elapsed
	}
	telemetry.CustomerLifecycleCommitted(ctx, id, events.EventoKYCActualizado, "KYC", string(previous), string(status), duration)
	return &customer, nil
}

func (r *customerRepo) UpdateStatusWithOutbox(ctx context.Context, id uuid.UUID, status models.CustomerStatus, event *models.OutboxMessage) (*models.Customer, error) {
	if status != models.StatusPendingActivation && status != models.StatusActive && status != models.StatusBlocked {
		return nil, errors.New("estado de cliente invalido")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var customer models.Customer
	err = tx.GetContext(ctx, &customer, `SELECT * FROM customers WHERE customer_id=$1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	previous := customer.Status
	if previous == status {
		return &customer, tx.Commit()
	}
	if err = ensureLifecycle(ctx, tx, &customer, event); err != nil {
		return nil, err
	}
	ctx, span := telemetry.StartCustomerLifecycle(ctx, customer.LifecycleTraceContext, "customer.lifecycle.access", *customer.LifecycleCorrelationID)
	defer span.End()
	ctx, finishDatabase := telemetry.Database(ctx, "UpdateStatusWithOutbox")
	defer finishDatabase()
	err = tx.GetContext(ctx, &customer, `UPDATE customers SET status=$1,updated_at=clock_timestamp() WHERE customer_id=$2 RETURNING *`, status, id)
	if err != nil {
		return nil, err
	}
	if err = saveLifecycle(ctx, tx, &customer, event, events.EventoClienteEstadoActualizado, "ACCESS", string(previous), string(status)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	telemetry.CustomerLifecycleCommitted(ctx, id, events.EventoClienteEstadoActualizado, "ACCESS", string(previous), string(status), nil)
	return &customer, nil
}

func (r *customerRepo) ActivateCustomer(ctx context.Context, customerID uuid.UUID, tokenID uuid.UUID, event *models.OutboxMessage) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var customer models.Customer
	if err = tx.GetContext(ctx, &customer, `SELECT * FROM customers WHERE customer_id=$1 FOR UPDATE`, customerID); err != nil {
		return err
	}
	if customer.Status != models.StatusPendingActivation {
		return errors.New("el cliente no esta pendiente de activacion")
	}
	result, err := tx.ExecContext(ctx, `UPDATE activation_tokens SET is_used=TRUE,used_at=clock_timestamp()
		WHERE id=$1 AND customer_id=$2 AND NOT is_used AND expires_at>clock_timestamp()`, tokenID, customerID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("token de activacion usado, expirado o inexistente")
	}
	if err = ensureLifecycle(ctx, tx, &customer, event); err != nil {
		return err
	}
	ctx, span := telemetry.StartCustomerLifecycle(ctx, customer.LifecycleTraceContext, "customer.lifecycle.activated", *customer.LifecycleCorrelationID)
	defer span.End()
	ctx, finishDatabase := telemetry.Database(ctx, "ActivateCustomer")
	defer finishDatabase()
	if _, err = tx.ExecContext(ctx, `UPDATE customers SET status='ACTIVO',updated_at=clock_timestamp() WHERE customer_id=$1`, customerID); err != nil {
		return err
	}
	if err = saveLifecycle(ctx, tx, &customer, event, events.EventoClienteActivado, "ACCESS", string(customer.Status), string(models.StatusActive)); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	telemetry.CustomerLifecycleCommitted(ctx, customerID, events.EventoClienteActivado, "ACCESS", string(customer.Status), string(models.StatusActive), nil)
	return nil
}

func (r *customerRepo) UpdateWithOutbox(ctx context.Context, customer *models.Customer, event *models.OutboxMessage) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current models.Customer
	if err = tx.GetContext(ctx, &current, `SELECT * FROM customers WHERE customer_id=$1 FOR UPDATE`, customer.CustomerID); err != nil {
		return err
	}
	if err = ensureLifecycle(ctx, tx, &current, event); err != nil {
		return err
	}
	ctx, span := telemetry.StartCustomerLifecycle(ctx, current.LifecycleTraceContext, "customer.lifecycle.updated", *current.LifecycleCorrelationID)
	defer span.End()
	ctx, finishDatabase := telemetry.Database(ctx, "UpdateWithOutbox")
	defer finishDatabase()
	// Profile updates cannot overwrite a concurrently changed access/KYC state.
	var updated models.Customer
	err = tx.GetContext(ctx, &updated, `UPDATE customers SET first_name=$1,last_name=$2,full_name=$3,
		email=$4,address=$5,document_photo_url=$6,updated_at=clock_timestamp()
		WHERE customer_id=$7 RETURNING *`, customer.FirstName, customer.LastName, customer.FullName,
		customer.Email, customer.Address, customer.DocumentPhotoURL, customer.CustomerID)
	if err != nil {
		return err
	}
	if err = saveLifecycle(ctx, tx, &current, event, events.EventoClienteActualizado, "PROFILE", "", "UPDATED"); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	*customer = updated
	telemetry.CustomerLifecycleCommitted(ctx, customer.CustomerID, events.EventoClienteActualizado, "PROFILE", "", "UPDATED", nil)
	return nil
}

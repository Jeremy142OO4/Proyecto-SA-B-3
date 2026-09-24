package services

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/repositories"

	"github.com/google/uuid"
)

type AuditService interface {
	ProcessEvent(ctx context.Context, envelope *events.EventEnvelope) error
	SetRecipientResolver(resolver RecipientResolver)
	GetAuditByCorrelation(ctx context.Context, correlationID uuid.UUID) ([]*models.AuditLog, error)
	GetRecentAudits(ctx context.Context, limit int) ([]*models.AuditLog, error)
	GetNotifications(ctx context.Context, filter models.NotificationFilter) ([]*models.NotificationLog, error)
}

// RecipientResolver obtains the current email of the customer referenced by a
// domain event. Event producers only need to publish idCliente; the audit
// service keeps the notification concern independent from the business data.
type RecipientResolver interface {
	ResolveCustomer(ctx context.Context, customerID uuid.UUID) (email string, fullName string, err error)
}

type auditService struct {
	auditRepo        repositories.AuditRepository
	notificationRepo repositories.NotificationRepository
	idempotencyRepo  repositories.IdempotencyRepository
	emailSender      EmailSender
	recipientResolver RecipientResolver
}

func (s *auditService) SetRecipientResolver(resolver RecipientResolver) {
	s.recipientResolver = resolver
}

func NewAuditService(
	auditRepo repositories.AuditRepository,
	notificationRepo repositories.NotificationRepository,
	idempotencyRepo repositories.IdempotencyRepository,
	emailSender EmailSender,
) AuditService {
	return &auditService{
		auditRepo:        auditRepo,
		notificationRepo: notificationRepo,
		idempotencyRepo:  idempotencyRepo,
		emailSender:      emailSender,
	}
}

func (s *auditService) ProcessEvent(ctx context.Context, envelope *events.EventEnvelope) error {
	if envelope == nil || envelope.MessageID == uuid.Nil || envelope.CorrelationID == uuid.Nil || strings.TrimSpace(envelope.Type) == "" {
		return fmt.Errorf("sobre de evento inválido: messageId, correlationId y tipo son obligatorios")
	}

	processed, err := s.idempotencyRepo.IsMessageProcessed(ctx, envelope.MessageID)
	if err != nil {
		return fmt.Errorf("error verificando idempotencia: %w", err)
	}

	if processed {
		log.Printf(
			"[notification-audit-service] mensaje duplicado omitido: messageId=%s correlationId=%s",
			envelope.MessageID,
			envelope.CorrelationID,
		)
		return nil
	}

	auditLog := &models.AuditLog{
		ID:            idempotentID("audit", envelope.MessageID),
		EventID:       envelope.MessageID,
		CorrelationID: envelope.CorrelationID,
		CausationID:   envelope.CausationID,
		EventType:     envelope.Type,
		Severity:      ClassifyEventWithPayload(envelope.Type, envelope.Payload),
		Producer:      envelope.Producer,
		Version:       envelope.Version,
		Payload:       envelope.Payload,
		OccurredAt:    envelope.OccurredAt,
		RecordedAt:    time.Now().UTC(),
	}

	if err := s.auditRepo.SaveAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("error guardando audit log: %w", err)
	}

	if err := s.handleNotificationDispatch(ctx, envelope); err != nil {
		return fmt.Errorf("error procesando notificación: %w", err)
	}

	if err := s.idempotencyRepo.MarkMessageProcessed(
		ctx,
		envelope.MessageID,
		"notification-audit-consumer",
		envelope.Type,
	); err != nil {
		return fmt.Errorf("error marcando mensaje como procesado: %w", err)
	}

	return nil
}

func (s *auditService) handleNotificationDispatch(
	ctx context.Context,
	envelope *events.EventEnvelope,
) error {
	switch envelope.Type {
	case "notificacion.correo-activacion.solicitado":
		return s.sendActivationEmail(ctx, envelope)
	default:
		rule, ok := notificationRuleFor(envelope.Type)
		if !ok {
			return nil
		}

		recipient := extractRecipient(envelope.Payload, rule.defaultRecipient)
		fullName := extractFullName(envelope.Payload)
		errorDetail := ""
		if !isEmail(recipient) {
			customerID := extractCustomerID(envelope.Payload)
			if s.recipientResolver == nil || customerID == uuid.Nil {
				errorDetail = "destinatario de correo no disponible en el evento"
			} else {
				var err error
				resolverCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				recipient, fullName, err = s.recipientResolver.ResolveCustomer(resolverCtx, customerID)
				cancel()
				if err != nil {
					errorDetail = fmt.Sprintf("no fue posible consultar el correo del cliente: %v", err)
				}
			}
		}
		return s.sendEventEmail(ctx, envelope, rule, recipient, fullName, errorDetail)
	}
}

type notificationRule struct {
	notificationType string
	subject          string
	bodySummary      string
	defaultRecipient  string
}

func notificationRuleFor(eventType string) (notificationRule, bool) {
	rules := map[string]notificationRule{
		"cliente.creado":                    {"CLIENT_CREATED", "Registro de cliente recibido", "El cliente fue registrado correctamente.", "customer"},
		"cliente.activado":                  {"CLIENT_ACTIVATED", "Cliente activado", "El cliente fue activado correctamente.", "customer"},
		"cliente.actualizado":               {"CLIENT_UPDATED", "Perfil actualizado", "El perfil del cliente fue actualizado.", "customer"},
		"cliente.estado.actualizado":        {"CLIENT_STATUS_UPDATED", "Estado de cliente actualizado", "El estado del cliente fue actualizado.", "customer"},
		"cliente.rechazado":                 {"CLIENT_VALIDATION_REJECTED", "Validación de cliente rechazada", "El cliente no pudo validarse para la operación solicitada.", "customer"},
		"cliente.kyc.verificado":            {"KYC_VERIFIED", "Validación KYC aprobada", "La validación KYC fue aprobada.", "customer"},
		"cliente.kyc.rechazado":             {"KYC_REJECTED", "Validación KYC rechazada", "La validación KYC fue rechazada.", "customer"},
		"cliente.kyc.estado.actualizado":    {"KYC_UPDATED", "Estado KYC actualizado", "El estado KYC fue actualizado.", "customer"},
		"cuenta.creada":                     {"ACCOUNT_CREATED", "Cuenta creada", "La cuenta bancaria fue creada correctamente.", "account-owner"},
		"cuenta.creacion.rechazada":         {"ACCOUNT_REJECTED", "Creación de cuenta rechazada", "La solicitud de cuenta no pudo completarse.", "account-owner"},
		"cuenta.debitada":                   {"ACCOUNT_DEBITED", "Débito aplicado", "El débito fue aplicado correctamente.", "account-owner"},
		"cuenta.debito.rechazado":           {"ACCOUNT_DEBIT_REJECTED", "Débito rechazado", "El débito no pudo aplicarse.", "account-owner"},
		"cuenta.acreditada":                 {"ACCOUNT_CREDITED", "Crédito aplicado", "El crédito fue aplicado correctamente.", "account-owner"},
		"cuenta.credito.rechazado":          {"ACCOUNT_CREDIT_REJECTED", "Crédito rechazado", "El crédito no pudo aplicarse.", "account-owner"},
		"cuenta.compensada":                 {"ACCOUNT_COMPENSATED", "Cuenta compensada", "La operación fue compensada correctamente.", "account-owner"},
		"cuenta.compensacion.rechazada":     {"ACCOUNT_COMPENSATION_REJECTED", "Compensación rechazada", "La compensación no pudo aplicarse.", "account-owner"},
		"cuenta.desactivada":                {"ACCOUNT_DEACTIVATED", "Cuenta desactivada", "La cuenta fue desactivada por inactividad.", "account-owner"},
		"cuenta.transferencia.rechazada":    {"ACCOUNT_TRANSFER_REJECTED", "Validación de transferencia rechazada", "La transferencia no cumplió las reglas de las cuentas.", "accounts-involved"},
		"transferencia.completada":          {"TRANSFER_COMPLETED", "Transferencia completada", "La transferencia fue completada satisfactoriamente.", "accounts-involved"},
		"transferencia.rechazada":           {"TRANSFER_REJECTED", "Transferencia rechazada", "La transferencia fue rechazada sin aplicar cambios definitivos.", "accounts-involved"},
		"transferencia.compensando":         {"TRANSFER_COMPENSATING", "Transferencia en compensación", "La transferencia está siendo compensada.", "accounts-involved"},
		"transferencia.compensada":          {"TRANSFER_COMPENSATED", "Transferencia compensada", "La transferencia fue compensada y los fondos regresaron a la cuenta origen.", "accounts-involved"},
		"transferencia.compensacion.fallida": {"TRANSFER_COMPENSATION_FAILED", "Error al compensar transferencia", "La compensación requiere revisión administrativa.", "accounts-involved"},
		"pago.completado":                   {"PAYMENT_COMPLETED", "Pago completado", "El pago fue procesado correctamente.", "customer"},
		"pago.rechazado":                    {"PAYMENT_REJECTED", "Pago rechazado", "El pago no pudo completarse.", "customer"},
	}
	rule, ok := rules[eventType]
	return rule, ok
}

func extractRecipient(payload json.RawMessage, fallback string) string {
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err == nil {
		for _, key := range []string{"correo", "email", "recipient", "destinatario", "idCliente", "customerId", "idCuenta", "accountId"} {
			if value, ok := fields[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return fallback
}

func extractFullName(payload json.RawMessage) string {
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err == nil {
		for _, key := range []string{"nombreCompleto", "fullName", "nombre"} {
			if value, ok := fields[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return "cliente"
}

func extractCustomerID(payload json.RawMessage) uuid.UUID {
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		return uuid.Nil
	}
	for _, key := range []string{"idCliente", "customerId", "customerID"} {
		if value, ok := fields[key].(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(value)); err == nil {
				return id
			}
		}
	}
	return uuid.Nil
}

func isEmail(value string) bool {
	value = strings.TrimSpace(value)
	return strings.Contains(value, "@") && !strings.ContainsAny(value, " \t\r\n")
}

func (s *auditService) sendEventEmail(
	ctx context.Context,
	envelope *events.EventEnvelope,
	rule notificationRule,
	recipient string,
	fullName string,
	errorDetail string,
) error {
	severity := ClassifyEventWithPayload(envelope.Type, envelope.Payload)
	icon, greeting, subject := notificationPresentation(severity)
	if severity != models.EventError {
		subject = asuntoCorreo(envelope.Type, subject)
	}
	rule.subject = subject
	status := models.NotificationSent
	if strings.TrimSpace(errorDetail) != "" {
		status = models.NotificationFailed
	} else if s.emailSender == nil {
		status = models.NotificationFailed
		errorDetail = "SMTP no configurado"
	} else {
		body := construirCorreoOperacion(envelope, rule, fullName, icon, subject, greeting)
		if err := s.emailSender.Send(recipient, subject, body); err != nil {
			status = models.NotificationFailed
			errorDetail = err.Error()
			log.Printf("[notification-audit-service] fallo al enviar correo: event=%s correlationId=%s recipient=%s error=%v", envelope.Type, envelope.CorrelationID, recipient, err)
		}
	}
	if err := s.saveGeneratedNotification(ctx, envelope, rule, recipient, status, errorDetail); err != nil {
		log.Printf("[notification-audit-service] error registrando notificacion: correlationId=%s error=%v", envelope.CorrelationID, err)
		return err
	}
	return nil
}

func notificationPresentation(severity models.EventSeverity) (icon, greeting, subject string) {
	switch severity {
	case models.EventError:
		return "❌", "Ha ocurrido un problema con una operación de tu cuenta.", "Problema con una operación de Bank USAC"
	case models.EventWarning:
		return "⚠️", "Hay una situación de tu cuenta que requiere tu atención.", "Aviso importante de Bank USAC"
	default:
		return "ℹ️", "Te informamos que una operación de tu cuenta fue procesada.", "Información de tu cuenta en Bank USAC"
	}
}

func construirCorreoOperacion(envelope *events.EventEnvelope, rule notificationRule, fullName, icon, subject, greeting string) string {
	campos := map[string]any{}
	_ = json.Unmarshal(envelope.Payload, &campos)
	estado := valorCorreo(campos, "estado")
	if estado == "" {
		estado = estadoCorreo(envelope.Type, ClassifyEventWithPayload(envelope.Type, envelope.Payload))
	}
	severity := ClassifyEventWithPayload(envelope.Type, envelope.Payload)
	colorPrincipal := "#2f8f68"
	colorSuave := "#e8f5ee"
	titulo := "OPERACIÓN EXITOSA"
	switch severity {
	case models.EventWarning:
		colorPrincipal = "#b27612"
		colorSuave = "#fff4d8"
		titulo = "AVISO DE OPERACIÓN"
	case models.EventError:
		colorPrincipal = "#b33a3a"
		colorSuave = "#fdeaea"
		titulo = "PROBLEMA CON LA OPERACIÓN"
	}

	var cuerpo strings.Builder
	fmt.Fprintf(&cuerpo, `<!doctype html>
<html lang="es"><head><meta charset="UTF-8"></head>
<body style="margin:0;background:#eef4f8;font-family:Arial,Helvetica,sans-serif;color:#173f61;">
  <div style="max-width:620px;margin:24px auto;background:#fff;border:1px solid #dbe5ec;border-radius:10px;overflow:hidden;">
    <div style="padding:25px 28px;background:#0b1d35;color:#fff;text-align:center;">
      <div style="font-size:26px;font-weight:800;letter-spacing:.3px;">%s %s</div>
      <div style="margin-top:8px;color:#d9e7f1;font-size:14px;">Bank USAC · Comprobante informativo</div>
    </div>
    <div style="padding:30px 30px 12px;">
      <p style="margin:0 0 16px;font-size:18px;">Hola <strong>%s</strong>,</p>
      <p style="margin:0 0 24px;color:#526c80;line-height:1.55;">%s</p>
      <div style="border:1px solid #dbe5ec;border-radius:10px;overflow:hidden;">
        <div style="padding:16px 18px;background:%s;color:%s;font-size:18px;font-weight:800;">%s</div>
        <div style="padding:18px;">
          <table role="presentation" style="width:100%%;border-collapse:collapse;font-size:14px;">`, html.EscapeString(icon), html.EscapeString(titulo), html.EscapeString(nombreCorreo(fullName)), html.EscapeString(greeting), colorSuave, colorPrincipal, html.EscapeString(strings.ToUpper(estado)))

	agregarFilaHTML(&cuerpo, "Operación", tipoOperacionCorreo(envelope.Type))
	agregarFilaHTML(&cuerpo, "Estado", estado)
	agregarFilaHTML(&cuerpo, "Realizado por", nombreCorreo(fullName))

	switch {
	case strings.HasPrefix(envelope.Type, "transferencia."):
		agregarFilaHTML(&cuerpo, "Cuenta origen", valorCorreo(campos, "idCuentaOrigen"))
		agregarFilaHTML(&cuerpo, "Cuenta destino", valorCorreo(campos, "idCuentaDestino"))
		agregarMontoHTML(&cuerpo, "Monto enviado", campos, "montoCentavos", "montoTotalCentavos")
		agregarFilaHTML(&cuerpo, "Descripción", valorCorreo(campos, "descripcion"))
	case strings.HasPrefix(envelope.Type, "pago."):
		agregarFilaHTML(&cuerpo, "Cuenta origen", valorCorreo(campos, "idCuentaOrigen"))
		agregarFilaHTML(&cuerpo, "Beneficiario", valorCorreo(campos, "beneficiario"))
		agregarFilaHTML(&cuerpo, "Tipo de pago", valorCorreo(campos, "tipoPago"))
		agregarFilaHTML(&cuerpo, "Concepto", valorCorreo(campos, "concepto"))
		agregarMontoHTML(&cuerpo, "Monto", campos, "montoCentavos")
		agregarFilaHTML(&cuerpo, "Referencia externa", valorCorreo(campos, "referenciaExterna"))
	default:
		agregarFilaHTML(&cuerpo, "Cuenta", valorCorreo(campos, "idCuenta"))
		agregarMontoHTML(&cuerpo, "Monto", campos, "montoCentavos", "saldoCentavos")
	}

	agregarFilaHTML(&cuerpo, "Código", valorCorreo(campos, "codigo", "codigoError"))
	agregarFilaHTML(&cuerpo, "Motivo", valorCorreo(campos, "motivo", "motivoRechazo"))
	agregarFilaHTML(&cuerpo, "Fecha", envelope.OccurredAt.Local().Format("02/01/2006 15:04"))
	agregarFilaHTML(&cuerpo, "CorrelationId", envelope.CorrelationID.String())
	fmt.Fprintf(&cuerpo, `</table>
        </div>
      </div>
      <div style="margin:22px 0;padding:14px 16px;border-left:4px solid %s;background:#f4f7fa;color:#526c80;font-size:13px;line-height:1.5;"><strong>Resumen:</strong> %s</div>
      <p style="color:#526c80;font-size:13px;line-height:1.5;">Gracias por utilizar Bank USAC. Conserva este comprobante y el CorrelationId para cualquier consulta.</p>
    </div>
    <div style="padding:16px 24px;background:#0b1d35;color:#d9e7f1;text-align:center;font-size:12px;">Este correo fue generado automáticamente. Por favor, no respondas a este mensaje.</div>
  </div>
</body></html>`, colorPrincipal, html.EscapeString(rule.bodySummary))
	return cuerpo.String()
}

func agregarFilaHTML(cuerpo *strings.Builder, etiqueta, valor string) {
	if strings.TrimSpace(valor) == "" {
		return
	}
	fmt.Fprintf(cuerpo, `<tr><td style="padding:9px 8px;border-bottom:1px solid #edf1f4;color:#6d8190;width:38%%;vertical-align:top;">%s</td><td style="padding:9px 8px;border-bottom:1px solid #edf1f4;color:#173f61;font-weight:700;overflow-wrap:anywhere;">%s</td></tr>`, html.EscapeString(etiqueta), html.EscapeString(valor))
}

func agregarMontoHTML(cuerpo *strings.Builder, etiqueta string, campos map[string]any, claves ...string) {
	for _, clave := range claves {
		if monto, ok := numeroCorreo(campos[clave]); ok {
			agregarFilaHTML(cuerpo, etiqueta, fmt.Sprintf("Q %.2f", monto/100))
			return
		}
	}
}

func valorCorreo(campos map[string]any, claves ...string) string {
	for _, clave := range claves {
		if valor, ok := campos[clave]; ok && valor != nil {
			texto := strings.TrimSpace(fmt.Sprint(valor))
			if texto != "" && texto != "<nil>" {
				return texto
			}
		}
	}
	return ""
}

func numeroCorreo(valor any) (float64, bool) {
	switch numero := valor.(type) {
	case float64:
		return numero, true
	case float32:
		return float64(numero), true
	case int:
		return float64(numero), true
	case int64:
		return float64(numero), true
	case json.Number:
		resultado, err := numero.Float64()
		return resultado, err == nil
	default:
		return 0, false
	}
}

func nombreCorreo(nombre string) string {
	if strings.TrimSpace(nombre) == "" {
		return "cliente"
	}
	return strings.TrimSpace(nombre)
}

func tipoOperacionCorreo(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "transferencia."):
		return "Transferencia bancaria"
	case strings.HasPrefix(eventType, "pago."):
		return "Pago"
	case strings.HasPrefix(eventType, "cuenta."):
		return "Operación de cuenta"
	default:
		return "Operación bancaria"
	}
}

func asuntoCorreo(eventType, fallback string) string {
	switch {
	case strings.HasPrefix(eventType, "transferencia."):
		if strings.Contains(eventType, "rechaz") || strings.Contains(eventType, "fallida") {
			return "Transferencia rechazada - Bank USAC"
		}
		if strings.Contains(eventType, "complet") {
			return "Transferencia completada - Bank USAC"
		}
		return "Actualización de transferencia - Bank USAC"
	case strings.HasPrefix(eventType, "pago."):
		if strings.Contains(eventType, "rechaz") {
			return "Pago rechazado - Bank USAC"
		}
		if strings.Contains(eventType, "complet") {
			return "Pago completado - Bank USAC"
		}
		return "Actualización de pago - Bank USAC"
	default:
		return fallback
	}
}

func estadoCorreo(eventType string, severity models.EventSeverity) string {
	if severity == models.EventError {
		return "ERROR"
	}
	if strings.Contains(eventType, "rechaz") || strings.Contains(eventType, "fallida") {
		return "RECHAZADA"
	}
	if strings.Contains(eventType, "complet") || strings.Contains(eventType, "cread") || strings.Contains(eventType, "acredit") || strings.Contains(eventType, "debit") || strings.Contains(eventType, "compensad") {
		return "COMPLETADA"
	}
	return "INFORMACIÓN"
}

func (s *auditService) saveGeneratedNotification(
	ctx context.Context,
	envelope *events.EventEnvelope,
	rule notificationRule,
	recipient string,
	status models.NotificationStatus,
	errorDetail string,
) error {
	return s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
		ID:               idempotentID("notification", envelope.MessageID),
		CorrelationID:    envelope.CorrelationID,
		Recipient:        recipient,
		NotificationType: rule.notificationType,
		Subject:          rule.subject,
		BodySummary:      rule.bodySummary,
		Status:           status,
		ErrorDetail:      optionalString(errorDetail),
		SentAt:           time.Now().UTC(),
	})
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func idempotentID(prefix string, messageID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(uuid.Nil, []byte(prefix+":"+messageID.String()))
}

// ClassifyEvent keeps the event-name-only classifier used by existing callers.
// Events received from RabbitMQ should use ClassifyEventWithPayload so that an
// event such as pago.rechazado can still be ERROR when its payload says
// TIMEOUT_PROVEEDOR or contains a 5xx status.
func ClassifyEvent(eventType string) models.EventSeverity {
	return classifyEvent(eventType, nil)
}

// ClassifyEventWithPayload applies the transversal policy:
// INFO for successful/in-progress events, WARNING for recoverable business or
// validation errors, and ERROR for technical/infrastructure failures.
func ClassifyEventWithPayload(eventType string, payload json.RawMessage) models.EventSeverity {
	var fields map[string]any
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &fields)
	}
	return classifyEvent(eventType, fields)
}

func classifyEvent(eventType string, fields map[string]any) models.EventSeverity {
	tipo := strings.ToLower(strings.TrimSpace(eventType))
	if status, ok := numericField(fields, "statusCode", "httpStatus", "codigoHttp", "status"); ok {
		if status >= 500 {
			return models.EventError
		}
		if status >= 400 {
			return models.EventWarning
		}
	}

	for _, value := range stringFields(fields, "codigo", "codigoError", "errorCode", "motivo", "motivoRechazo", "error", "detalleError", "reason", "estado", "status") {
		if technicalFailure(value) {
			return models.EventError
		}
		if recoverableFailure(value) {
			return models.EventWarning
		}
	}

	if technicalFailure(tipo) {
		return models.EventError
	}
	if recoverableFailure(tipo) {
		return models.EventWarning
	}
	return models.EventInfo
}

func numericField(fields map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok {
			continue
		}
		switch number := value.(type) {
		case float64:
			return int(number), true
		case int:
			return number, true
		case int64:
			return int(number), true
		case json.Number:
			parsed, err := number.Int64()
			if err == nil {
				return int(parsed), true
			}
		case string:
			var parsed int
			if _, err := fmt.Sscanf(strings.TrimSpace(number), "%d", &parsed); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func stringFields(fields map[string]any, keys ...string) []string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := fields[key].(string); ok && strings.TrimSpace(value) != "" {
			values = append(values, value)
		}
	}
	return values
}

func technicalFailure(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, token := range []string{
		"5xx", "timeout", "tiempo de espera", "time out", "dlq", "dead-letter",
		"base de datos", "database", "postgres", "rabbit", "infraestructura",
		"infrastructure", "proveedor externo", "fallo externo", "fallo tecnico",
		"falla tecnica", "error interno", "internal server", "connection refused",
		"conexión rechazada", "panic", "no disponible", "compensacion.fallida",
		"compensación fallida",
	} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return strings.Contains(value, "fallid") && !recoverableFailure(value)
}

func recoverableFailure(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, token := range []string{
		"4xx", "rechaz", "invalid", "inval", "no encontrado", "not found",
		"no autorizado", "unauthorized", "forbidden", "prohibid", "conflict",
		"rate limit", "limite", "saldo", "kyc", "cuenta_no", "cuentas_no",
		"cliente_no", "pendiente", "compens", "datos incorrectos", "datos invalidos",
	} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func (s *auditService) sendActivationEmail(
	ctx context.Context,
	envelope *events.EventEnvelope,
) error {
	var payload events.ActivationEmailPayload

	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		log.Printf(
			"[notification-audit-service] payload inválido para correo de activación: messageId=%s correlationId=%s error=%v",
			envelope.MessageID,
			envelope.CorrelationID,
			err,
		)
		detail := "payload inválido para correo de activación"
		if saveErr := s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
			ID:               idempotentID("notification", envelope.MessageID),
			CorrelationID:    envelope.CorrelationID,
			Recipient:        "unknown",
			NotificationType: "ACTIVATION_EMAIL",
			Subject:          "Activa tu cuenta en Bank USAC",
			BodySummary:      "No se pudo generar el correo de activación.",
			Status:           models.NotificationFailed,
			ErrorDetail:      &detail,
			SentAt:           time.Now().UTC(),
		}); saveErr != nil {
			log.Printf("[notification-audit-service] no se pudo registrar payload inválido: correlationId=%s error=%v", envelope.CorrelationID, saveErr)
			return saveErr
		}
		return nil
	}

	subject := "Activa tu cuenta en Bank USAC"

	body := fmt.Sprintf(
		`<!doctype html><html lang="es"><head><meta charset="UTF-8"></head>
<body style="margin:0;background:#eef4f8;font-family:Arial,Helvetica,sans-serif;color:#173f61;">
  <div style="max-width:620px;margin:24px auto;background:#fff;border:1px solid #dbe5ec;border-radius:10px;overflow:hidden;">
    <div style="padding:25px 28px;background:#0b1d35;color:#fff;text-align:center;font-size:25px;font-weight:800;">¡REGISTRO EXITOSO!</div>
    <div style="padding:30px;">
      <p style="font-size:18px;">Hola <strong>%s</strong>,</p>
      <p style="color:#526c80;line-height:1.55;">Tu cuenta de Bank USAC ha sido registrada correctamente.</p>
      <div style="margin:24px 0;padding:18px;background:#f4f7fa;border-left:4px solid #f4773c;">
        <strong>Activa tu cuenta</strong><br><br>
        <a href="%s" style="display:inline-block;padding:12px 18px;background:#f4773c;color:#fff;text-decoration:none;border-radius:6px;font-weight:700;">Activar cuenta</a>
      </div>
      <p style="color:#526c80;font-size:13px;">El enlace vence el: %s</p>
      <p style="color:#526c80;font-size:13px;">Si no solicitaste este registro, ignora este correo.</p>
    </div>
    <div style="padding:16px 24px;background:#0b1d35;color:#d9e7f1;text-align:center;font-size:12px;">Bank USAC · Correo automático</div>
  </div>
</body></html>`,
		html.EscapeString(payload.FullName),
		html.EscapeString(payload.ActivationLink),
		html.EscapeString(payload.ExpiresAt.Format(time.RFC1123)),
	)

	status := models.NotificationSent
	var errorDetail string
	if s.emailSender == nil {
		status = models.NotificationFailed
		errorDetail = "SMTP no configurado"
	} else if err := s.emailSender.Send(payload.Email, subject, body); err != nil {
		status = models.NotificationFailed
		errorDetail = err.Error()
		log.Printf(
			"[notification-audit-service] fallo al enviar correo de activación: messageId=%s correlationId=%s recipient=%s error=%v",
			envelope.MessageID,
			envelope.CorrelationID,
			payload.Email,
			err,
		)
	}

	if err := s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
		ID:               idempotentID("notification", envelope.MessageID),
		CorrelationID:    envelope.CorrelationID,
		Recipient:        payload.Email,
		NotificationType: "ACTIVATION_EMAIL",
		Subject:          subject,
		BodySummary:      fmt.Sprintf("Correo de activación para %s", payload.FullName),
		Status:           status,
		ErrorDetail:      optionalString(errorDetail),
		SentAt:           time.Now().UTC(),
	}); err != nil {
		log.Printf(
			"[notification-audit-service] no se pudo registrar el resultado de notificación: correlationId=%s error=%v",
			envelope.CorrelationID,
			err,
		)
		return err
	}

	if status == models.NotificationSent {
		log.Printf("[notification-audit-service] correo de activación enviado: messageId=%s correlationId=%s recipient=%s", envelope.MessageID, envelope.CorrelationID, payload.Email)
	}
	return nil
}

func (s *auditService) GetAuditByCorrelation(
	ctx context.Context,
	correlationID uuid.UUID,
) ([]*models.AuditLog, error) {
	return s.auditRepo.GetAuditLogsByCorrelationID(ctx, correlationID)
}

func (s *auditService) GetRecentAudits(
	ctx context.Context,
	limit int,
) ([]*models.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.auditRepo.GetRecentAuditLogs(ctx, limit)
}

func (s *auditService) GetNotifications(
	ctx context.Context,
	filter models.NotificationFilter,
) ([]*models.NotificationLog, error) {
	return s.notificationRepo.GetNotificationLogs(ctx, filter)
}

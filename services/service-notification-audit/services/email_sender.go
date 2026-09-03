package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

type EmailSender interface {
	Send(to string, subject string, body string) error
}

type SMTPEmailSender struct {
	host        string
	port        int
	username    string
	appPassword string
	from        string
}

func NewSMTPEmailSender(
	host string,
	port int,
	username string,
	appPassword string,
	from string,
) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:        host,
		port:        port,
		username:    username,
		appPassword: appPassword,
		from:        from,
	}
}

func (s *SMTPEmailSender) Send(to string, subject string, body string) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("el destinatario del correo es obligatorio")
	}

	if strings.TrimSpace(s.host) == "" ||
		s.port <= 0 ||
		strings.TrimSpace(s.username) == "" ||
		strings.TrimSpace(s.appPassword) == "" ||
		strings.TrimSpace(s.from) == "" {
		return fmt.Errorf("la configuración SMTP está incompleta")
	}

	address := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	client, err := smtp.Dial(address)
	if err != nil {
		return fmt.Errorf("no fue posible conectar al servidor SMTP: %w", err)
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}); err != nil {
		return fmt.Errorf("no fue posible iniciar TLS con el servidor SMTP: %w", err)
	}

	auth := smtp.PlainAuth(
		"",
		s.username,
		s.appPassword,
		s.host,
	)

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("autenticación SMTP rechazada: %w", err)
	}

	// SMTP MAIL FROM debe ser una dirección limpia, sin el nombre visible.
	if err := client.Mail(s.username); err != nil {
		return fmt.Errorf("no fue posible establecer el remitente: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("no fue posible establecer el destinatario: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("no fue posible crear el contenido del correo: %w", err)
	}

	message := strings.Join([]string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("no fue posible escribir el correo: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("no fue posible enviar el correo: %w", err)
	}

	return nil
}

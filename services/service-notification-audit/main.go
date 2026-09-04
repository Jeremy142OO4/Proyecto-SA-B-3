package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bank-usac/service-notification-audit/config"
	"bank-usac/service-notification-audit/controllers"
	"bank-usac/service-notification-audit/database"
	"bank-usac/service-notification-audit/messaging"
	"bank-usac/service-notification-audit/repositories"
	"bank-usac/service-notification-audit/routes"
	"bank-usac/service-notification-audit/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Conexión a Base de Datos
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fallo al conectar a notification_audit_db: %v", err)
	}
	defer db.Close()

	// 2. Repositorios y Servicios
	auditRepo := repositories.NewAuditRepository(db)
	notifRepo := repositories.NewNotificationRepository(db)
	idempotencyRepo := repositories.NewIdempotencyRepository(db)

	emailSender := services.NewSMTPEmailSender(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPAppPassword,
		cfg.SMTPFrom,
	)

	auditSvc := services.NewAuditService(auditRepo, notifRepo, idempotencyRepo, emailSender)
	controller := controllers.NewAuditController(auditSvc)

	// 3. Conexión y Consumidor de RabbitMQ
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rabbitConsumer, err := messaging.NewRabbitMQConsumer(cfg.RabbitMQURL, auditSvc)
	if err != nil {
		log.Fatalf("Fallo al inicializar RabbitMQ Consumer: %v", err)
	}
	defer rabbitConsumer.Close()

	if err := rabbitConsumer.StartConsuming(ctx); err != nil {
		log.Fatalf("Fallo al iniciar consumo de eventos: %v", err)
	}

	// 4. Servidor HTTP Fiber
	app := fiber.New(fiber.Config{
		AppName: "Bank USAC - Notification & Audit Service",
	})
	app.Use(recover.New())
	app.Use(logger.New())

	routes.SetupRoutes(app, controller, cfg)

	// 5. Cierre controlado
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Printf("Error en servidor HTTP: %v", err)
		}
	}()
	log.Printf("[service-notification-audit] Ejecutándose en el puerto %s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[service-notification-audit] Apagando servicio de forma controlada...")
	cancel()
	_ = app.ShutdownWithTimeout(5 * time.Second)
	log.Println("[service-notification-audit] Servicio detenido")
}

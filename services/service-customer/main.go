package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bank-usac/service-customer/config"
	"bank-usac/service-customer/controllers"
	"bank-usac/service-customer/database"
	"bank-usac/service-customer/messaging"
	"bank-usac/service-customer/repositories"
	"bank-usac/service-customer/routes"
	"bank-usac/service-customer/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Conexión a Base de Datos
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Fallo al conectar a customer_db: %v", err)
	}
	defer db.Close()

	// 2. Repositorios y Servicios
	repo := repositories.NewCustomerRepository(db)
	svc := services.NewCustomerService(repo, cfg)
	controller := controllers.NewCustomerController(svc)

	// 3. Conexión RabbitMQ y Outbox Worker
	rabbit, err := messaging.NewRabbitMQClient(cfg.RabbitMQURL, repo)
	if err != nil {
		log.Fatalf("Fallo al conectar a RabbitMQ: %v", err)
	}
	defer rabbit.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rabbit.StartOutboxWorker(ctx, 1*time.Second)
	if err := rabbit.StartValidationConsumer(ctx); err != nil {
		log.Fatalf("Fallo al iniciar consumidor de validacion de clientes: %v", err)
	}

	// 4. Servidor HTTP Fiber
	app := fiber.New(fiber.Config{
		AppName: "Bank USAC - Customer Service",
	})
	app.Use(recover.New())
	app.Use(logger.New())

	routes.SetupRoutes(app, controller, cfg)

	// 5. Cierre controlado (Graceful Shutdown)
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Printf("Error en servidor HTTP: %v", err)
		}
	}()
	log.Printf("[customer-service] Ejecutándose en el puerto %s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[customer-service] Apagando servicio de forma controlada...")
	cancel()
	_ = app.ShutdownWithTimeout(5 * time.Second)
	log.Println("[customer-service] Servicio detenido")
}

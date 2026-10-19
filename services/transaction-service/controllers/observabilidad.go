package controllers

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/Proyecto-SA-B-3/transaction-service/repositories"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"time"
)

type TransferLookup interface {
	Consultar(context.Context, uuid.UUID) (models.Transferencia, error)
}

func TrazaTransferencia(repo TransferLookup) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("idTransferencia"))
		if err != nil || id == uuid.Nil {
			return fiber.NewError(400, "idTransferencia invalido")
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		t, err := repo.Consultar(ctx, id)
		if errors.Is(err, repositories.ErrNoEncontrada) {
			return fiber.ErrNotFound
		}
		if err != nil {
			return fiber.NewError(503, "consulta de trazabilidad no disponible")
		}
		return c.JSON(fiber.Map{"id_transaccion": t.IDTransferencia, "traceId": t.TraceID, "correlationId": t.IDCorrelacion, "estado": t.Estado, "estadoFase2": telemetry.Phase2State(t.Estado), "trazaDisponible": t.TraceID != ""})
	}
}

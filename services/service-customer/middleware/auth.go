package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthMiddleware(secret string, allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token de autenticación faltante o inválido",
			})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token expirado o inválido",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Claims inválidos"})
		}

		userRole, _ := claims["role"].(string)
		sub, _ := claims["sub"].(string)
		customerID, _ := uuid.Parse(sub)

		c.Locals("customerId", customerID)
		c.Locals("userRole", userRole)

		if len(allowedRoles) > 0 {
			roleAllowed := false
			for _, r := range allowedRoles {
				if r == userRole {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "No posee los permisos necesarios para realizar esta acción",
				})
			}
		}

		return c.Next()
	}
}

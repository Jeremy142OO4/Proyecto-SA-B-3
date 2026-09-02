package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Claims struct {
	Subject string `json:"sub"`
	Role    string `json:"role"`
	Expires int64  `json:"exp"`
}

func Autenticacion(secreto string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		partes := strings.SplitN(c.Get("Authorization"), " ", 2)
		if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "token requerido")
		}
		claims, ok := validarJWT(partes[1], []byte(secreto))
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "token invalido o expirado")
		}
		c.Locals("customerId", claims.Subject)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}

func validarJWT(token string, secreto []byte) (Claims, bool) {
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return Claims{}, false
	}
	entrada := partes[0] + "." + partes[1]
	firma, err := base64.RawURLEncoding.DecodeString(partes[2])
	if err != nil {
		return Claims{}, false
	}
	mac := hmac.New(sha256.New, secreto)
	mac.Write([]byte(entrada))
	if !hmac.Equal(firma, mac.Sum(nil)) {
		return Claims{}, false
	}
	encabezado, err := base64.RawURLEncoding.DecodeString(partes[0])
	if err != nil {
		return Claims{}, false
	}
	var h struct {
		Algorithm string `json:"alg"`
	}
	if json.Unmarshal(encabezado, &h) != nil || h.Algorithm != "HS256" {
		return Claims{}, false
	}
	cuerpo, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil {
		return Claims{}, false
	}
	var claims Claims
	if json.Unmarshal(cuerpo, &claims) != nil || claims.Subject == "" || !rolValido(claims.Role) || claims.Expires <= time.Now().Unix() {
		return Claims{}, false
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return Claims{}, false
	}
	return claims, true
}

func AutorizarRoles(roles ...string) fiber.Handler {
	permitidos := make(map[string]struct{}, len(roles))
	for _, rol := range roles {
		permitidos[rol] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		rol, ok := c.Locals("role").(string)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "usuario no autenticado")
		}
		if _, permitido := permitidos[rol]; !permitido {
			return fiber.NewError(fiber.StatusForbidden, "rol sin permiso para esta operacion")
		}
		return c.Next()
	}
}

func rolValido(rol string) bool {
	switch rol {
	case "ADMIN", "TELLER", "CLIENTE":
		return true
	default:
		return false
	}
}

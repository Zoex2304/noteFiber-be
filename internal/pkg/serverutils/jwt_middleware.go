// FILE: internal/pkg/serverutils/jwt_middleware.go
package serverutils

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JwtMiddleware(ctx *fiber.Ctx) error {
	authHeader := ctx.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing token"})
	}
	tokenStr := authHeader[7:]

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid claims"})
	}

	ctx.Locals("user_id", claims["user_id"])
	return ctx.Next()
}
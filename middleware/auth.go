package middleware

import (
	"github.com/labstack/echo/v4"
	"redapplications.com/redreader/repository"
)

type AuthMiddleware struct {
	userRepo *repository.UserRepository
}

func NewAuthMiddleware(userRepo *repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{userRepo: userRepo}
}

func (m *AuthMiddleware) IsAuthenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("auth_token")
		if err != nil {
			return c.Redirect(302, "/auth/google/login")
		}

		user, err := m.userRepo.GetUserByToken(cookie.Value)
		if err != nil {
			return c.Redirect(302, "/auth/google/login")
		}

		c.Set("user", user)
		return next(c)
	}
}

func (m *AuthMiddleware) AttachUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("auth_token")
		if err == nil {
			user, err := m.userRepo.GetUserByToken(cookie.Value)
			if err == nil {
				c.Set("user", user)
			}
		}
		return next(c)
	}
}

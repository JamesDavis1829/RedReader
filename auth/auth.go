package auth

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"redapplications.com/redreader/models"
	"redapplications.com/redreader/repository"
)

const (
	cookieName = "auth_token"
)

func SetAuthCookie(c echo.Context, user *models.User, userRepo *repository.UserRepository) error {
	token := uuid.New().String()

	user.Tokens = append(user.Tokens, token)
	if err := userRepo.UpdateUser(user); err != nil {
		return err
	}

	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = token
	cookie.Expires = time.Now().Add(365 * 24 * time.Hour)
	cookie.MaxAge = 365 * 24 * 60 * 60
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.SameSite = http.SameSiteLaxMode

	if domain := c.Request().Host; domain != "" {
		cookie.Domain = domain
	}

	c.SetCookie(cookie)
	return nil
}

func ClearAuthCookie(c echo.Context, user *models.User, userRepo *repository.UserRepository) {
	// Remove the specific token from user's tokens
	if cookie, err := c.Cookie(cookieName); err == nil {
		token := cookie.Value
		newTokens := make([]string, 0)
		for _, t := range user.Tokens {
			if t != token {
				newTokens = append(newTokens, t)
			}
		}
		user.Tokens = newTokens
		userRepo.UpdateUser(user)
	}

	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour)
	cookie.Path = "/"
	cookie.HttpOnly = true

	c.SetCookie(cookie)
}

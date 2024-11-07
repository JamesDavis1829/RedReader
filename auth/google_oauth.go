package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"redapplications.com/redreader/models"
	"redapplications.com/redreader/repository"
)

var googleOauthConfig *oauth2.Config

func GoogleOauthInit() {
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectUrl := os.Getenv("GOOGLE_REDIRECT_URI")
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  redirectUrl,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes: []string{
			"email",
		},
		Endpoint: google.Endpoint,
	}
}

func HandleGoogleLogin(c echo.Context) error {
	url := googleOauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleGoogleCallback(c echo.Context, userRepo *repository.UserRepository) error {
	code := c.QueryParam("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return err
	}

	// New code to get user info
	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return err
	}

	var user *models.User
	user, err = userRepo.GetUserByEmail(userInfo.Email)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			user = models.NewUser(userInfo.Email, userInfo.Name)
			if err := userRepo.CreateUser(user); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := SetAuthCookie(c, user, userRepo); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

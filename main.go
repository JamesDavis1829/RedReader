package main

import (
	"html/template"
	"io"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"redapplications.com/redreader/auth"
	"redapplications.com/redreader/db"
	"redapplications.com/redreader/middleware"
	"redapplications.com/redreader/models"
	"redapplications.com/redreader/repository"
	"redapplications.com/redreader/worker"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type PageData struct {
	Title    string
	Header   string
	Subtitle string
	User     *models.User
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	auth.GoogleOauthInit()

	e := echo.New()

	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = t

	// Initialize repositories
	mongoClient := db.NewMongoClient()
	userRepo := repository.NewUserRepository(mongoClient)
	feedRepo := repository.NewFeedRepository(mongoClient)
	articleRepo := repository.NewArticleRepository(mongoClient)

	// Initialize and start background worker
	backgroundWorker := worker.NewBackgroundWorker(feedRepo, articleRepo)
	backgroundWorker.Start()
	defer backgroundWorker.Stop()

	userRepo.CreateIndex()

	e.Static("/assets", "assets")

	authMiddleware := middleware.NewAuthMiddleware(userRepo)
	e.Pre(authMiddleware.AttachUser)

	e.GET("/", func(c echo.Context) error {
		data := PageData{
			Title:    "Red Reader",
			Header:   "Hello World",
			Subtitle: "My first website with Bulma!",
		}

		if user := c.Get("user"); user != nil {
			data.User = user.(*models.User)
		}

		return c.Render(200, "index.html", data)
	})

	e.GET("/index", func(c echo.Context) error {
		return c.Redirect(301, "/")
	})

	e.GET("/login", auth.HandleGoogleLogin)
	e.GET("/callback/google", func(c echo.Context) error {
		return auth.HandleGoogleCallback(c, userRepo)
	})

	e.GET("/logout", func(c echo.Context) error {
		if user := c.Get("user"); user != nil {
			auth.ClearAuthCookie(c, user.(*models.User), userRepo)
		}
		return c.Redirect(302, "/")
	})

	e.Logger.Fatal(e.Start(":1323"))
}

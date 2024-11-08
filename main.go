package main

import (
	"html/template"
	"io"
	"math"
	"strconv"

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
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"subtract": func(a, b int64) int64 { return a - b },
			"add":      func(a, b int64) int64 { return a + b },
			"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		}).ParseGlob("templates/*.html")),
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
			Header:   "Your Feeds",
			Subtitle: "Stay updated with your favorite content",
		}

		if user := c.Get("user"); user != nil {
			data.User = user.(*models.User)
		}

		return c.Render(200, "index.html", data)
	})

	// Update feeds handler with pagination
	e.GET("/feeds", func(c echo.Context) error {
		page, _ := strconv.ParseInt(c.QueryParam("page"), 10, 64)
		if page < 1 {
			page = 1
		}
		perPage := int64(9) // Show 9 feeds per page

		feeds, total, err := feedRepo.GetPaginatedFeeds(page, perPage)
		if err != nil {
			return err
		}

		totalPages := int64(math.Ceil(float64(total) / float64(perPage)))
		pages := make([]int64, totalPages)
		for i := int64(0); i < totalPages; i++ {
			pages[i] = i + 1
		}

		return c.Render(200, "feed_list.html", map[string]interface{}{
			"Feeds":       feeds,
			"CurrentPage": page,
			"TotalPages":  totalPages,
			"Pages":       pages,
		})
	})

	// Add this new handler
	e.GET("/feeds/:id/articles", func(c echo.Context) error {
		feedId := c.Param("id")
		page, _ := strconv.ParseInt(c.QueryParam("page"), 10, 64)
		feedPage, _ := strconv.ParseInt(c.QueryParam("feedPage"), 10, 64)
		if page < 1 {
			page = 1
		}
		if feedPage < 1 {
			feedPage = 1
		}
		perPage := int64(10) // Show 10 articles per page

		feed, err := feedRepo.GetFeed(feedId)
		if err != nil {
			return err
		}

		articles, total, err := articleRepo.GetPaginatedArticlesByFeed(feedId, page, perPage)
		if err != nil {
			return err
		}

		totalPages := int64(math.Ceil(float64(total) / float64(perPage)))
		pages := make([]int64, totalPages)
		for i := int64(0); i < totalPages; i++ {
			pages[i] = i + 1
		}

		return c.Render(200, "article_list.html", map[string]interface{}{
			"Feed":        feed,
			"Articles":    articles,
			"CurrentPage": page,
			"TotalPages":  totalPages,
			"Pages":       pages,
			"FeedPage":    feedPage,
		})
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

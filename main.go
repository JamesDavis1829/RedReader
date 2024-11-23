package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
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

//go:embed templates/*
var templateFs embed.FS

//go:embed assets/*
var assetFs embed.FS

const (
	perPage = int64(15)
)

type Template struct {
	templateFuncs template.FuncMap
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	var err error
	renderTemplates := template.New("").Funcs(t.templateFuncs)
	isHtmx := c.Request().Header.Get("HX-Request") == "true"
	addUserToData(c, data)

	renderTemplates, err = renderTemplates.ParseFS(templateFs, "templates/components/*")
	if err != nil {
		return fmt.Errorf("failed to parse components: %w", err)
	}

	if isHtmx {
		renderTemplates, err = renderTemplates.ParseFS(templateFs, "templates/base_htmx.html")
	} else {
		renderTemplates, err = renderTemplates.ParseFS(templateFs, "templates/base.html")
	}
	if err != nil {
		return fmt.Errorf("failed to parse base template: %w", err)
	}

	renderTemplates, err = renderTemplates.ParseFS(templateFs, "templates/"+name)
	if err != nil {
		return fmt.Errorf("failed to parse content template: %w", err)
	}

	if isHtmx {
		return renderTemplates.ExecuteTemplate(w, "base_htmx.html", data)
	}

	return renderTemplates.ExecuteTemplate(w, "base.html", data)
}

func addUserToData(c echo.Context, data interface{}) {
	if user := c.Get("user"); user != nil {
		if mapData, ok := data.(map[string]interface{}); ok {
			mapData["User"] = user.(*models.User)
			return
		}
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	auth.GoogleOauthInit()

	e := echo.New()

	t := &Template{
		templateFuncs: template.FuncMap{
			"subtract": func(a, b int64) int64 { return a - b },
			"add":      func(a, b int64) int64 { return a + b },
			"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		},
	}
	e.Renderer = t

	mongoClient := db.NewMongoClient()
	userRepo := repository.NewUserRepository(mongoClient)
	feedRepo := repository.NewFeedRepository(mongoClient)
	articleRepo := repository.NewArticleRepository(mongoClient)

	backgroundWorker := worker.NewBackgroundWorker(feedRepo, articleRepo)
	backgroundWorker.Start()
	defer backgroundWorker.Stop()

	userRepo.CreateIndex()

	assets, err := fs.Sub(assetFs, "assets")
	if err != nil {
		panic(err)
	}

	e.StaticFS("/assets", assets)

	authMiddleware := middleware.NewAuthMiddleware(userRepo)
	e.Pre(authMiddleware.AttachUser)

	e.GET("/", func(c echo.Context) error {
		data := map[string]interface{}{
			"Title":    "Red Reader",
			"Header":   "Your Feeds",
			"Subtitle": "Stay updated with your favorite content",
		}
		return c.Render(200, "index.html", data)
	})

	e.GET("/feeds", func(c echo.Context) error {
		page, _ := strconv.ParseInt(c.QueryParam("page"), 10, 64)
		if page < 1 {
			page = 1
		}

		feeds, total, err := feedRepo.GetPaginatedFeeds(page, perPage)
		if err != nil {
			return err
		}

		// Add subscription status if user is logged in
		if user := c.Get("user"); user != nil {
			feedRepo.AddSubscriptionStatus(feeds, user.(*models.User).SubscribedTo)
		}

		pages, totalPages := calculatePages(total, perPage, page)
		return c.Render(200, "feed_list.html", map[string]interface{}{
			"Feeds":       feeds,
			"CurrentPage": page,
			"TotalPages":  totalPages,
			"Pages":       pages,
		})
	})

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

		feed, err := feedRepo.GetFeed(feedId)
		if err != nil {
			return err
		}

		articles, total, err := articleRepo.GetPaginatedArticlesByFeed(feedId, page, perPage)
		if err != nil {
			return err
		}

		pages, totalPages := calculatePages(total, perPage, page)

		return c.Render(200, "article_list.html", map[string]interface{}{
			"Feed":        feed,
			"Articles":    articles,
			"CurrentPage": page,
			"TotalPages":  totalPages,
			"Pages":       pages,
			"FeedPage":    feedPage,
		})
	})

	e.GET("/articles/:id/content", func(c echo.Context) error {
		articleId := c.Param("id")
		article, err := articleRepo.GetArticleContent(articleId)
		if err != nil {
			return err
		}
		return c.Render(200, "article_modal.html", article)
	})

	e.GET("/articles", func(c echo.Context) error {
		page, _ := strconv.ParseInt(c.QueryParam("page"), 10, 64)
		if page < 1 {
			page = 1
		}

		var articles []*repository.ArticleWithFeed
		var total int64
		var err error

		user := c.Get("user")
		if user != nil {
			articles, total, err = articleRepo.GetPaginatedArticlesForUser(user.(*models.User), page, perPage)
		} else {
			articles, total, err = articleRepo.GetPaginatedArticles(page, perPage)
		}

		if err != nil {
			return err
		}

		pages, totalPages := calculatePages(total, perPage, page)

		return c.Render(200, "articles.html", map[string]interface{}{
			"Articles":    articles,
			"CurrentPage": page,
			"TotalPages":  totalPages,
			"Pages":       pages,
			"User":        user,
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

	e.GET("/article/:id", func(c echo.Context) error {
		articleId := c.Param("id")
		article, err := articleRepo.GetArticleContent(articleId)
		if err != nil {
			return err
		}
		return c.Render(200, "article_view.html", map[string]interface{}{
			"Title":       article.Title,
			"Author":      article.Author,
			"PublishedAt": article.PublishedAt,
			"FeedTitle":   article.FeedTitle,
			"URL":         article.URL,
			"ViewContent": article.ViewContent(),
		})
	})

	// Add subscribe/unsubscribe routes
	e.POST("/feeds/:id/subscribe", func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		feedId := c.Param("id")

		if err := userRepo.SubscribeToFeed(user.ID, feedId); err != nil {
			return err
		}

		feed, err := feedRepo.GetFeed(feedId)
		if err != nil {
			return err
		}
		feed.IsSubscribed = true

		return c.Render(200, "feed_card.html", feed)
	})

	e.DELETE("/feeds/:id/subscribe", func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		feedId := c.Param("id")

		if err := userRepo.UnsubscribeFromFeed(user.ID, feedId); err != nil {
			return err
		}

		feed, err := feedRepo.GetFeed(feedId)
		if err != nil {
			return err
		}
		feed.IsSubscribed = false

		return c.Render(200, "feed_card.html", feed)
	})

	e.POST("/feeds", func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		url := c.FormValue("url")

		feed, err := feedRepo.AddFeed(url)
		if err != nil {
			c.Response().Header().Set("HX-Reswap", "innerHTML")
			c.Response().Header().Set("HX-Retarget", "#modal-error-message")
			return c.String(200, "<p>Failed to add feed</p>")
		}

		if err := userRepo.AddPersonalFeed(user.ID, feed.ID.String()); err != nil {
			c.Response().Header().Set("HX-Reswap", "innerHTML")
			c.Response().Header().Set("HX-Retarget", "#modal-error-message")
			return c.String(200, "<p>Failed to add personal feed</p>")
		}

		feeds, total, err := feedRepo.GetPaginatedFeeds(1, 18)
		if err != nil {
			return c.String(400, err.Error())
		}

		pages, totalPages := calculatePages(total, perPage, 1)

		return c.Render(200, "feed_list.html", map[string]interface{}{
			"Feeds":       feeds,
			"CurrentPage": int64(1),
			"TotalPages":  totalPages,
			"Pages":       pages,
		})
	})

	e.Logger.Fatal(e.Start(":1323"))
}

func calculatePages(total int64, perPage int64, page int64) ([]int64, int64) {
	totalPages := int64(math.Ceil(float64(total) / float64(perPage)))

	windowStart := page - 5
	if windowStart < 1 {
		windowStart = 1
	}
	windowEnd := windowStart + 10
	if windowEnd > totalPages {
		windowEnd = totalPages
	}

	var pages []int64
	for i := windowStart; i <= windowEnd; i++ {
		pages = append(pages, i)
	}

	if totalPages > windowEnd {
		pages = append(pages, totalPages)
	}

	return pages, totalPages
}

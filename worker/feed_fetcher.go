package worker

import (
	"time"

	"github.com/mmcdole/gofeed"
	"redapplications.com/redreader/models"
	"redapplications.com/redreader/repository"
)

type FeedFetcher struct {
	feedRepo    *repository.FeedRepository
	articleRepo *repository.ArticleRepository
	parser      *gofeed.Parser
}

func NewFeedFetcher(feedRepo *repository.FeedRepository, articleRepo *repository.ArticleRepository) *FeedFetcher {
	return &FeedFetcher{
		feedRepo:    feedRepo,
		articleRepo: articleRepo,
		parser:      gofeed.NewParser(),
	}
}

func (f *FeedFetcher) FetchAll() error {
	feeds, err := f.feedRepo.GetAllFeeds()
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		if err := f.FetchOne(feed); err != nil {
			// Log error but continue with other feeds
			println("Error fetching feed:", feed.Title, feed.URL, err)
			continue
		}
	}
	return nil
}

func (f *FeedFetcher) FetchOne(feed *models.Feed) error {
	parsedFeed, err := f.parser.ParseURL(feed.URL)
	if err != nil {
		return err
	}

	// Update feed metadata
	feed.Title = parsedFeed.Title
	feed.Description = parsedFeed.Description
	feed.LastFetched = time.Now()

	if err := f.feedRepo.UpdateLastFetched(feed.ID, feed.LastFetched); err != nil {
		return err
	}

	// Process items
	for _, item := range parsedFeed.Items {
		if item == nil {
			continue
		}

		// Skip if article already exists
		exists, err := f.articleRepo.ArticleExists(item.Link)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		article := models.NewArticle(feed.ID)
		article.Title = item.Title
		article.Description = item.Description
		article.Content = item.Content
		article.URL = item.Link
		if len(item.Authors) > 0 {
			article.Author = item.Authors[0].Name
		} else {
			article.Author = ""
		}

		if item.PublishedParsed != nil {
			article.PublishedAt = *item.PublishedParsed
		} else {
			article.PublishedAt = time.Now()
		}

		if err := f.articleRepo.CreateArticle(article); err != nil {
			return err
		}
	}

	return nil
}

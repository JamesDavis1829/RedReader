package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"redapplications.com/redreader/models"
	"redapplications.com/redreader/repository"
)

type HackerNewsFetcher struct {
	feedRepo    *repository.FeedRepository
	articleRepo *repository.ArticleRepository
}

func NewHackerNewsFetcher(feedRepo *repository.FeedRepository, articleRepo *repository.ArticleRepository) *HackerNewsFetcher {
	return &HackerNewsFetcher{
		feedRepo:    feedRepo,
		articleRepo: articleRepo,
	}
}

func (h *HackerNewsFetcher) FetchAndSave() error {
	stories, err := FetchTopStories()
	if err != nil {
		return fmt.Errorf("failed to fetch HN stories: %v", err)
	}

	feed, err := h.feedRepo.GetFeedByTitle(context.Background(), "Hacker News")
	if err != nil {
		return fmt.Errorf("failed to get HN feed: %v", err)
	}

	for _, story := range stories {
		// Skip stories without URLs
		if story.URL == "" {
			continue
		}

		// Check if article already exists
		exists, err := h.articleRepo.ArticleExists(story.URL)
		if err != nil || exists {
			continue
		}

		article := models.NewArticle(feed.ID.String())
		article.Title = story.Title
		article.URL = story.URL
		article.Author = story.By
		article.Description = fmt.Sprintf("Points: %d | Comments: %d", story.Score, story.Descendants)

		//Hacker News stories get to the front page by votes, well after their published date
		//so we set the published date to the time the article was fetched to artificially boost it
		article.PublishedAt = time.Now()

		if err := h.articleRepo.CreateArticle(article); err != nil {
			println("Error saving HN article:", err.Error())
			continue
		}
	}

	return nil
}

// Story represents a Hacker News story
type Story struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Score       int       `json:"score"`
	Time        time.Time `json:"-"`
	RawTime     int64     `json:"time"`
	By          string    `json:"by"`
	Type        string    `json:"type"`
	Kids        []int64   `json:"kids"`
	Descendants int       `json:"descendants"`
	CreatedAt   time.Time
}

const (
	url = "https://hacker-news.firebaseio.com/v0/"
)

func FetchTopStories() ([]Story, error) {
	resp, err := http.Get(url + "beststories.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top stories: %v", err)
	}
	defer resp.Body.Close()

	var storyIDs []int64
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("failed to decode story IDs: %v", err)
	}

	stories := make([]Story, 0, len(storyIDs))
	for _, id := range storyIDs {
		story, err := fetchStory(id)
		if err != nil {
			continue
		}
		stories = append(stories, story)
	}

	return stories, nil
}

func fetchStory(id int64) (Story, error) {
	url := fmt.Sprintf(url+"item/%d.json", id)
	resp, err := http.Get(url)
	if err != nil {
		return Story{}, err
	}
	defer resp.Body.Close()

	var story Story
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return Story{}, err
	}

	story.Time = time.Unix(story.RawTime, 0)
	story.CreatedAt = time.Now()

	return story, nil
}

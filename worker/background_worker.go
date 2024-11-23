package worker

import (
	"time"

	"redapplications.com/redreader/repository"
)

type BackgroundWorker struct {
	fetcher   *FeedFetcher
	hnFetcher *HackerNewsFetcher
	ticker    *time.Ticker
	done      chan bool
}

func NewBackgroundWorker(feedRepo *repository.FeedRepository, articleRepo *repository.ArticleRepository) *BackgroundWorker {
	return &BackgroundWorker{
		fetcher:   NewFeedFetcher(feedRepo, articleRepo),
		hnFetcher: NewHackerNewsFetcher(feedRepo, articleRepo),
		ticker:    time.NewTicker(15 * time.Minute),
		done:      make(chan bool),
	}
}

func (w *BackgroundWorker) Start() {
	println("Starting background worker...")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				println("Recovered from panic in background worker:", r)
			}
		}()

		//w.safeFetch()

		for {
			select {
			case <-w.done:
				println("Background worker stopped")
				return
			case <-w.ticker.C:
				w.safeFetch()
			}
		}
	}()
}

func (w *BackgroundWorker) safeFetch() {
	defer func() {
		if r := recover(); r != nil {
			println("Recovered from panic in fetch:", r)
		}
	}()

	// Fetch RSS feeds
	if err := w.fetcher.FetchAll(); err != nil {
		println("Error in feed fetching:", err)
	}

	// Fetch Hacker News stories
	if err := w.hnFetcher.FetchAndSave(); err != nil {
		println("Error in Hacker News fetching:", err)
	}
}

func (w *BackgroundWorker) Stop() {
	w.ticker.Stop()
	w.done <- true
}

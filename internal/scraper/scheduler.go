package scraper

import (
	"context"
	"database/sql"
	"log"
	"math/rand"
	"time"

	"github.com/LealKevin/keiko/internal/store"
)

const (
	baseInterval    = 6 * time.Hour
	maxRandomOffset = 30 * time.Minute
)

type Scheduler struct {
	scraper *Scraper
	store   *store.Store
}

func NewScheduler(scraper *Scraper, store *store.Store) *Scheduler {
	return &Scheduler{scraper: scraper, store: store}
}

func (s *Scheduler) Start(ctx context.Context) {
	log.Println("Scheduler started")

	lastRun, err := s.store.GetLastRun(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No previous run found, running now")
		} else {
			log.Printf("Could not get last run time: %v, running now", err)
		}
		s.run(ctx)
	} else {
		elapsed := time.Since(lastRun)
		if elapsed >= baseInterval {
			log.Printf("Last run was %v ago (overdue), running now", elapsed.Round(time.Minute))
			s.run(ctx)
		} else {
			remaining := baseInterval - elapsed
			log.Printf("Last run was %v ago, waiting %v", elapsed.Round(time.Minute), remaining.Round(time.Minute))
			select {
			case <-time.After(remaining):
				s.run(ctx)
			case <-ctx.Done():
				log.Println("Scheduler stopped")
				return
			}
		}
	}

	for {
		offset := time.Duration(rand.Int63n(int64(maxRandomOffset)))
		nextRun := baseInterval + offset

		log.Printf("Next fetch scheduled in %v", nextRun.Round(time.Minute))

		select {
		case <-time.After(nextRun):
			s.run(ctx)
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) run(ctx context.Context) {
	log.Println("Running scheduled fetch...")
	if err := s.scraper.FetchAndProcess(ctx); err != nil {
		log.Printf("Fetch error: %v", err)
		return
	}
	if err := s.store.SetLastRun(ctx, time.Now()); err != nil {
		log.Printf("Failed to persist last run time: %v", err)
	}
}

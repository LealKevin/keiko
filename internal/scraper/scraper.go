package scraper

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/LealKevin/keiko/internal/ai"
	"github.com/LealKevin/keiko/internal/store"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type Scraper struct {
	store     *store.Store
	tokenizer *ai.Tokenizer
}

func New(store *store.Store, tokenizer *ai.Tokenizer) *Scraper {
	return &Scraper{
		store:     store,
		tokenizer: tokenizer,
	}
}

func (s *Scraper) FetchAndProcess(ctx context.Context) error {
	log.Println("Starting news fetch...")

	newsIDs, err := s.fetchNewsIDs()
	if err != nil {
		return fmt.Errorf("failed to fetch news IDs: %w", err)
	}

	log.Printf("Found %d news articles", len(newsIDs))

	browser := rod.New().ControlURL(newLauncher().MustLaunch()).MustConnect()
	defer func() {
		if err := browser.Close(); err != nil {
			log.Printf("Error closing browser: %v", err)
		}
	}()

	newCount := 0
	for _, nhkID := range newsIDs {
		exists, err := s.store.NewsExists(ctx, nhkID)
		if err != nil {
			log.Printf("Error checking if news exists: %v", err)
			continue
		}

		if exists {
			log.Printf("Skipping %s (already exists)", nhkID)
			continue
		}

		if err := s.processArticle(ctx, browser, nhkID); err != nil {
			log.Printf("Error processing %s: %v", nhkID, err)
			if isRateLimitError(err) {
				log.Println("Rate limit reached, stopping for now. Will retry next run.")
				break
			}
			continue
		}

		newCount++
		log.Printf("Processed %s", nhkID)

		time.Sleep(2 * time.Second)
	}

	log.Printf("Finished: processed %d new articles", newCount)
	return nil
}

func (s *Scraper) fetchNewsIDs() ([]string, error) {
	browser := rod.New().ControlURL(newLauncher().MustLaunch()).MustConnect()
	defer func() {
		if err := browser.Close(); err != nil {
			log.Printf("Error closing browser: %v", err)
		}
	}()

	page := browser.MustPage("https://www3.nhk.or.jp/news/easy/")
	defer page.MustClose()
	page.MustWaitLoad()
	page.MustWaitStable()

	for _, b := range page.MustElements("button") {
		if regexp.MustCompile(`understand|確認しました`).MatchString(b.MustText()) {
			b.MustClick()
			break
		}
	}

	loadMoreButton, err := page.Element(".button-more")
	if err == nil && loadMoreButton != nil {
		loadMoreButton.MustClick()
		page.MustWaitStable()
	}

	html := page.MustHTML()

	re := regexp.MustCompile(`/news/easy/(ne[0-9]+)/`)
	matches := re.FindAllStringSubmatch(html, -1)

	var newsIDs []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			newsIDs = append(newsIDs, m[1])
			seen[m[1]] = true
		}
	}

	return newsIDs, nil
}

func (s *Scraper) processArticle(ctx context.Context, browser *rod.Browser, nhkID string) error {
	url := fmt.Sprintf("https://www3.nhk.or.jp/news/easy/%s/%s.html", nhkID, nhkID)
	page := browser.MustPage(url)
	defer page.MustClose()
	page.MustWaitLoad()
	page.MustWaitStable()

	for _, b := range page.MustElements("button") {
		if regexp.MustCompile(`understand|確認しました`).MatchString(b.MustText()) {
			b.MustClick()
			break
		}
	}

	title := page.MustElement(".article-title").MustText()
	dateStr := page.MustElement(".article-date").MustText()
	publishedAt := parseJapaneseDate(dateStr)

	article := page.MustElement(".article-body")
	ps := article.MustElements("p")

	var paragraphs []store.Paragraph
	for i, p := range ps {
		rawText := p.MustEval(`() => {
			let clone = this.cloneNode(true);
			clone.querySelectorAll('rt').forEach(rt => rt.remove());
			return clone.textContent;
		}`).String()

		if rawText == "" {
			continue
		}

		tokens, err := s.tokenizer.Tokenize(ctx, rawText)
		if err != nil {
			return fmt.Errorf("tokenization failed for paragraph %d: %w", i, err)
		}

		paragraphs = append(paragraphs, store.Paragraph{
			Position: i,
			RawText:  rawText,
			Tokens:   tokens,
		})

		time.Sleep(500 * time.Millisecond)
	}

	news := &store.News{
		NHKID:       nhkID,
		Title:       title,
		URL:         url,
		PublishedAt: publishedAt,
	}

	return s.store.InsertNews(ctx, news, paragraphs)
}

func parseJapaneseDate(s string) *time.Time {
	re := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 4 {
		return nil
	}

	var year, month, day int
	fmt.Sscanf(matches[1], "%d", &year)
	fmt.Sscanf(matches[2], "%d", &month)
	fmt.Sscanf(matches[3], "%d", &day)

	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &t
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED")
}

func newLauncher() *launcher.Launcher {
	l := launcher.New()

	if path := os.Getenv("CHROME_PATH"); path != "" {
		l = l.Bin(path)
	} else {
		paths := []string{
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				l = l.Bin(p)
				break
			}
		}
	}

	l = l.Headless(true).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("disable-setuid-sandbox")

	return l
}

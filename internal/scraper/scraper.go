package scraper

import (
	"context"
	"errors"
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

type ProcessingError struct {
	NhkID  string
	Reason string
	Err    error
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

	var (
		successCount     int
		skippedExisting  int
		skippedMaxRetry  int
		errors           []ProcessingError
		rateLimited      bool
	)

	for _, nhkID := range newsIDs {
		exists, err := s.store.NewsExists(ctx, nhkID)
		if err != nil {
			errors = append(errors, ProcessingError{
				NhkID:  nhkID,
				Reason: "db_check",
				Err:    err,
			})
			continue
		}

		if exists {
			skippedExisting++
			continue
		}

		shouldSkip, failure, err := s.store.ShouldSkipArticle(ctx, nhkID)
		if err != nil {
			log.Printf("Error checking failure status for %s: %v", nhkID, err)
		}
		if shouldSkip {
			skippedMaxRetry++
			log.Printf("Skipping %s (failed %d times, last error: %s)", nhkID, failure.Attempts, failure.ErrorType)
			continue
		}

		if err := s.processArticle(ctx, browser, nhkID); err != nil {
			reason := classifyError(err)
			errors = append(errors, ProcessingError{
				NhkID:  nhkID,
				Reason: reason,
				Err:    err,
			})
			log.Printf("Failed %s [%s]: %v", nhkID, reason, err)

			if recErr := s.store.RecordFailure(ctx, nhkID, reason, err.Error()); recErr != nil {
				log.Printf("Error recording failure: %v", recErr)
			}

			if isRateLimitError(err) {
				rateLimited = true
				log.Println("Rate limit reached, stopping processing")
				break
			}
			continue
		}

		s.store.ClearFailure(ctx, nhkID)
		successCount++
		log.Printf("Processed %s", nhkID)
		time.Sleep(2 * time.Second)
	}

	s.printSummary(len(newsIDs), successCount, skippedExisting, skippedMaxRetry, errors, rateLimited)
	return nil
}

func (s *Scraper) printSummary(total, success, skippedExisting, skippedMaxRetry int, errors []ProcessingError, rateLimited bool) {
	log.Println("")
	log.Println("=== Processing Summary ===")
	log.Printf("Total found:     %d", total)
	log.Printf("Already in DB:   %d", skippedExisting)
	log.Printf("Max retries:     %d (skipped)", skippedMaxRetry)
	log.Printf("Succeeded:       %d", success)
	log.Printf("Failed:          %d", len(errors))

	if rateLimited {
		log.Printf("Status:          Stopped (rate limit reached)")
	} else {
		log.Printf("Status:          Completed")
	}

	if len(errors) > 0 {
		log.Println("")
		log.Println("=== Failures ===")

		byReason := make(map[string][]ProcessingError)
		for _, e := range errors {
			byReason[e.Reason] = append(byReason[e.Reason], e)
		}

		for reason, errs := range byReason {
			log.Printf("[%s] %d failures:", reason, len(errs))
			for _, e := range errs {
				log.Printf("  - %s: %v", e.NhkID, e.Err)
			}
		}
	}
	log.Println("==========================")
}

func classifyError(err error) string {
	if err == nil {
		return "unknown"
	}
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "429") || strings.Contains(errStr, "RESOURCE_EXHAUSTED"):
		return "rate_limit"
	case strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "JSON parse") || strings.Contains(errStr, "unexpected end of JSON"):
		return "json_parse"
	case strings.Contains(errStr, "empty response"):
		return "empty_response"
	case strings.Contains(errStr, "paragraph count mismatch") || strings.Contains(errStr, "missing tokens"):
		return "tokenization_mismatch"
	case strings.Contains(errStr, "API error"):
		return "api_error"
	case strings.Contains(errStr, "scrape") || strings.Contains(errStr, "element"):
		return "scrape_error"
	default:
		return "other"
	}
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
	log.Printf("Processing %s...", nhkID)

	url := fmt.Sprintf("https://www3.nhk.or.jp/news/easy/%s/%s.html", nhkID, nhkID)

	var (
		title     string
		dateStr   string
		rawTexts  []string
		positions []int
	)

	scrapeErr := rod.Try(func() {
		page := browser.MustPage(url).Timeout(60 * time.Second)
		defer func() {
			page.CancelTimeout().Close()
		}()
		page.MustWaitLoad()
		page.MustWaitStable()

		for _, b := range page.MustElements("button") {
			if regexp.MustCompile(`understand|確認しました`).MatchString(b.MustText()) {
				b.MustClick()
				break
			}
		}

		title = page.MustElement(".article-title").MustText()
		dateStr = page.MustElement(".article-date").MustText()

		article := page.MustElement(".article-body")
		ps := article.MustElements("p")

		for i, p := range ps {
			rawText := p.MustEval(`() => {
				let clone = this.cloneNode(true);
				clone.querySelectorAll('rt').forEach(rt => rt.remove());
				return clone.textContent;
			}`).String()

			if rawText == "" {
				continue
			}
			rawTexts = append(rawTexts, rawText)
			positions = append(positions, i)
		}
	})
	if scrapeErr != nil {
		return fmt.Errorf("failed to scrape article: %w", errors.Unwrap(scrapeErr))
	}

	if len(rawTexts) == 0 {
		return fmt.Errorf("no paragraphs found in article")
	}

	publishedAt := parseJapaneseDate(dateStr)

	log.Printf("  Tokenizing %d paragraphs in batch...", len(rawTexts))
	allTokens, err := s.tokenizer.TokenizeBatch(ctx, rawTexts)
	if err != nil {
		return fmt.Errorf("batch tokenization failed: %w", err)
	}

	var paragraphs []store.Paragraph
	for i, tokens := range allTokens {
		paragraphs = append(paragraphs, store.Paragraph{
			Position: positions[i],
			RawText:  rawTexts[i],
			Tokens:   tokens,
		})
	}

	log.Printf("  Got %d tokens total", countTokens(allTokens))

	news := &store.News{
		NHKID:       nhkID,
		Title:       title,
		URL:         url,
		PublishedAt: publishedAt,
	}

	return s.store.InsertNews(ctx, news, paragraphs)
}

func countTokens(allTokens [][]store.Token) int {
	total := 0
	for _, tokens := range allTokens {
		total += len(tokens)
	}
	return total
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

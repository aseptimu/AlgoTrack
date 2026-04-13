package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
)

var ErrProblemNotFound = errors.New("leetcode problem not found")

// ErrLeetCodeUnavailable indicates that LeetCode API is temporarily unavailable.
var ErrLeetCodeUnavailable = errors.New("leetcode API unavailable")

// ErrResponseTooLarge indicates that the response body exceeded the allowed limit.
var ErrResponseTooLarge = errors.New("leetcode response too large")

const (
	// maxResponseSize limits the response body to 1 MB.
	maxResponseSize = 1 << 20 // 1 MB

	// defaultRateLimit is the maximum number of requests per second.
	defaultRateLimit = 2

	// retryMaxAttempts is the number of total attempts (1 initial + retries).
	retryMaxAttempts = 3

	// retryBaseDelay is the base delay for exponential backoff.
	retryBaseDelay = 500 * time.Millisecond

	userAgent = "AlgoTrack-Bot/1.0 (Go; +https://github.com/aseptimu/AlgoTrack)"
)

// rateLimiter is a simple token-bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	maxRate  float64
	lastTime time.Time
}

func newRateLimiter(rps float64) *rateLimiter {
	return &rateLimiter{
		tokens:   rps,
		maxRate:  rps,
		lastTime: time.Now(),
	}
}

// wait blocks until a token is available or ctx is cancelled.
func (rl *rateLimiter) wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(rl.lastTime).Seconds()
		rl.tokens += elapsed * rl.maxRate
		if rl.tokens > rl.maxRate {
			rl.tokens = rl.maxRate
		}
		rl.lastTime = now

		if rl.tokens >= 1 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate how long to wait for the next token.
		waitDuration := time.Duration((1.0 - rl.tokens) / rl.maxRate * float64(time.Second))
		rl.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}

type HTTPLeetCodeClient struct {
	baseURL     string
	httpClient  *http.Client
	rateLimiter *rateLimiter
	logger      *slog.Logger
}

func NewHTTPLeetCodeClient(logger *slog.Logger) *HTTPLeetCodeClient {
	if logger == nil {
		logger = slog.Default()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:  5 * time.Second,
		TLSNextProto:         make(map[string]func(string, *tls.Conn) http.RoundTripper),
		ResponseHeaderTimeout: 10 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
	}

	return &HTTPLeetCodeClient{
		baseURL: "https://leetcode.com/graphql",
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		rateLimiter: newRateLimiter(defaultRateLimit),
		logger:      logger,
	}
}

type leetCodeGraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type problemsetQuestionListResponse struct {
	Data struct {
		ProblemsetQuestionList struct {
			Questions []struct {
				FrontendQuestionID string `json:"questionFrontendId"`
				Title              string `json:"title"`
				TitleSlug          string `json:"titleSlug"`
				Difficulty         string `json:"difficulty"`
			} `json:"questions"`
		} `json:"problemsetQuestionList"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type recentSubmissionsResponse struct {
	Data struct {
		RecentAcSubmissionList []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			TitleSlug string `json:"titleSlug"`
			Timestamp string `json:"timestamp"`
		} `json:"recentAcSubmissionList"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// doGraphQL executes a GraphQL request with rate limiting, retry, and response validation.
func (c *HTTPLeetCodeClient) doGraphQL(ctx context.Context, gqlReq leetCodeGraphQLRequest, result any) error {
	body, err := json.Marshal(gqlReq)
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < retryMaxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			c.logger.Info("retrying leetcode request", "attempt", attempt+1, "delay", delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := c.rateLimiter.wait(ctx); err != nil {
			return fmt.Errorf("rate limiter: %w", err)
		}

		lastErr = c.doSingleRequest(ctx, body, result)
		if lastErr == nil {
			return nil
		}

		// Retry on transient errors (5xx, network errors, rate limit).
		if isRetryable(lastErr) {
			c.logger.Warn("leetcode request failed, will retry", "attempt", attempt+1, "err", lastErr)
			continue
		}

		return lastErr
	}

	return fmt.Errorf("%w: %v", ErrLeetCodeUnavailable, lastErr)
}

func (c *HTTPLeetCodeClient) doSingleRequest(ctx context.Context, body []byte, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request to leetcode: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return &retryableError{fmt.Errorf("leetcode returned status 429")}
	}
	if resp.StatusCode >= 500 {
		return &retryableError{fmt.Errorf("leetcode returned status %d", resp.StatusCode)}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("leetcode returned status %d", resp.StatusCode)
	}

	// Limit response size to prevent memory exhaustion.
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("read leetcode response: %w", err)
	}
	if len(respBody) > maxResponseSize {
		return ErrResponseTooLarge
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("decode leetcode response: %w", err)
	}

	return nil
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	var re *retryableError
	if errors.As(err, &re) {
		return true
	}
	// Network errors are retryable.
	var netErr net.Error
	return errors.As(err, &netErr)
}

func (c *HTTPLeetCodeClient) GetProblemByNumber(ctx context.Context, number int64) (*model.ProblemInfo, error) {
	if number <= 0 {
		return nil, fmt.Errorf("invalid problem number: %d", number)
	}

	query := fmt.Sprintf(`query {
  problemsetQuestionList: questionList(
    categorySlug: ""
    limit: 1
    skip: %d
    filters: {}
  ) {
    total: totalNum
    questions: data {
      questionFrontendId
      title
      titleSlug
      difficulty
    }
  }
}`, number-1)

	gqlReq := leetCodeGraphQLRequest{
		Query: query,
	}

	var gqlResp problemsetQuestionListResponse
	if err := c.doGraphQL(ctx, gqlReq, &gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("leetcode graphql error: %s", gqlResp.Errors[0].Message)
	}

	if len(gqlResp.Data.ProblemsetQuestionList.Questions) == 0 {
		return nil, ErrProblemNotFound
	}

	q := gqlResp.Data.ProblemsetQuestionList.Questions[0]

	if q.FrontendQuestionID != fmt.Sprintf("%d", number) {
		return nil, ErrProblemNotFound
	}

	return &model.ProblemInfo{
		Number:     int(number),
		Title:      sanitize(q.Title),
		TitleSlug:  sanitize(q.TitleSlug),
		Difficulty: sanitize(q.Difficulty),
		Link:       fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug),
		Platform:   "leetcode",
	}, nil
}

// GetRecentAcceptedSubmissions returns recent accepted submissions for a LeetCode user.
// This uses the public recentAcSubmissionList query which does not require authentication.
func (c *HTTPLeetCodeClient) GetRecentAcceptedSubmissions(ctx context.Context, username string, limit int) ([]model.LeetCodeSubmission, error) {
	if username == "" {
		return nil, fmt.Errorf("empty leetcode username")
	}
	if limit <= 0 || limit > 20 {
		limit = 20
	}

	gqlReq := leetCodeGraphQLRequest{
		Query: `query recentAcSubmissions($username: String!, $limit: Int!) {
  recentAcSubmissionList(username: $username, limit: $limit) {
    id
    title
    titleSlug
    timestamp
  }
}`,
		Variables: map[string]any{
			"username": username,
			"limit":    limit,
		},
	}

	var gqlResp recentSubmissionsResponse
	if err := c.doGraphQL(ctx, gqlReq, &gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("leetcode graphql error: %s", gqlResp.Errors[0].Message)
	}

	submissions := make([]model.LeetCodeSubmission, 0, len(gqlResp.Data.RecentAcSubmissionList))
	for _, s := range gqlResp.Data.RecentAcSubmissionList {
		submissions = append(submissions, model.LeetCodeSubmission{
			ID:        s.ID,
			Title:     sanitize(s.Title),
			TitleSlug: sanitize(s.TitleSlug),
			Timestamp: s.Timestamp,
		})
	}

	return submissions, nil
}

// sanitize removes HTML tags and trims whitespace from untrusted external data.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	// Strip basic HTML tags. This is a simple approach for the data we receive.
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return result.String()
}

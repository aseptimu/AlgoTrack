package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
)

var ErrProblemNotFound = errors.New("leetcode problem not found")

type HTTPLeetCodeClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPLeetCodeClient() *HTTPLeetCodeClient {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:   false,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSNextProto:        make(map[string]func(string, *tls.Conn) http.RoundTripper),
	}

	return &HTTPLeetCodeClient{
		baseURL: "https://leetcode.com/graphql",
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
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

	reqBody := leetCodeGraphQLRequest{
		Query: query,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request to leetcode: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leetcode returned status %d", resp.StatusCode)
	}

	var gqlResp problemsetQuestionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("decode leetcode response: %w", err)
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
		Title:      q.Title,
		TitleSlug:  q.TitleSlug,
		Difficulty: q.Difficulty,
		Link:       fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug),
		Platform:   "leetcode",
	}, nil
}

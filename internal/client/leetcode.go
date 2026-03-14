package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	return &HTTPLeetCodeClient{
		baseURL: "https://leetcode.com/graphql",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
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
				FrontendQuestionID string `json:"frontendQuestionId"`
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

	query := `
query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
  problemsetQuestionList: questionList(
    categorySlug: $categorySlug
    limit: $limit
    skip: $skip
    filters: $filters
  ) {
    questions: data {
      frontendQuestionId
      title
      titleSlug
      difficulty
    }
  }
}`

	reqBody := leetCodeGraphQLRequest{
		Query: query,
		Variables: map[string]any{
			"categorySlug": "all-code-essentials",
			"skip":         0,
			"limit":        1,
			"filters": map[string]any{
				"searchKeywords": fmt.Sprintf("%d", number),
			},
		},
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
	req.Header.Set("Referer", "https://leetcode.com/problemset/")
	req.Header.Set("Origin", "https://leetcode.com")
	req.Header.Set("User-Agent", "AlgoTrack/1.0")

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

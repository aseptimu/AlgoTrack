package client

import (
	"context"
	"fmt"
	"testing"
)

func TestHTTPLeetCodeClient_GetProblemByNumber(t *testing.T) {
	client := NewHTTPLeetCodeClient()
	problem, err := client.GetProblemByNumber(context.Background(), 2)
	fmt.Println(err)
	fmt.Println(problem)
}

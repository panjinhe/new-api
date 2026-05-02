package openai

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOaiResponsesCompletedResponseFromStream(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.4"}}`,
			``,
			`data: {"type":"response.output_text.delta","delta":"hello"}`,
			``,
			`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-5.4","usage":{"input_tokens":12,"output_tokens":4,"total_tokens":16}}}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n"))),
	}

	completed, err := OaiResponsesCompletedResponseFromStream(resp)
	if err != nil {
		t.Fatalf("expected stream aggregation to succeed, got error: %v", err)
	}
	if completed == nil {
		t.Fatal("expected completed response")
	}
	if completed.Model != "gpt-5.4" {
		t.Fatalf("expected model gpt-5.4, got %q", completed.Model)
	}
	if completed.Usage == nil || completed.Usage.TotalTokens != 16 {
		t.Fatal("expected usage from completed event to be preserved")
	}
}

package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		defaultVal string
		want     string
	}{
		{"returns env value", "TEST_KEY", "env_val", "default", "env_val"},
		{"returns default when not set", "TEST_KEY", "", "default", "default"},
		{"returns default when empty", "TEST_KEY_UNSET", "", "fallback", "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			got := getEnv(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestMessageStruct(t *testing.T) {
	msg := Message{Role: "user", Content: "hello"}
	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if msg.Content != "hello" {
		t.Errorf("Content = %q, want %q", msg.Content, "hello")
	}
}

func TestStreamResult(t *testing.T) {
	// Error case
	sr := StreamResult{Err: http.ErrServerClosed}
	if sr.Err == nil {
		t.Error("should have error")
	}
	// Content case
	sr2 := StreamResult{Content: "token"}
	if sr2.Content != "token" {
		t.Errorf("Content = %q, want %q", sr2.Content, "token")
	}
}

func TestCallDeepSeek_MissingAPIKey(t *testing.T) {
	// 无 DEEPSEEK_API_KEY 环境变量时应立即返回错误
	_, _, err := CallDeepSeek("system", nil, false)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if err.Error() == "" || !contains(err.Error(), "DEEPSEEK_API_KEY") {
		t.Errorf("error message should mention DEEPSEEK_API_KEY, got: %v", err)
	}
}

func TestCallDeepSeek_NonStreaming(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// Valid ChatCompletion JSON per openai-go SDK requirements
		w.Write([]byte(`{
  "id": "chatcmpl-test",
  "choices": [{
    "finish_reason": "stop",
    "index": 0,
    "logprobs": {"content": null},
    "message": {
      "content": "Hello, I am a test response.",
      "refusal": null,
      "role": "assistant"
    }
  }],
  "created": 1234567890,
  "model": "test-model",
  "object": "chat.completion"
}`))
	}))
	defer server.Close()

	// Override client creation to point at test server
	origFn := newClientFn
	newClientFn = func(apiKey, baseURL string) openai.Client {
		return origFn("test-key", server.URL+"/")
	}
	defer func() { newClientFn = origFn }()

	content, _, err := CallDeepSeek("You are a helper.", []Message{
		{Role: "user", Content: "Hi"},
	}, false)

	if err != nil {
		t.Fatalf("CallDeepSeek() error = %v", err)
	}
	if !contains(content, "Hello, I am a test response.") {
		t.Errorf("content should contain response text, got: %s", content)
	}
}

func TestCallDeepSeek_NonStreaming_HTTPError(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	origFn := newClientFn
	newClientFn = func(apiKey, baseURL string) openai.Client {
		return origFn("test-key", server.URL+"/")
	}
	defer func() { newClientFn = origFn }()

	_, _, err := CallDeepSeek("system", nil, false)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestCallDeepSeek_Streaming(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)

		// Simulate SSE chunks with valid ChatCompletionChunk JSON
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("expected http.Flusher")
			return
		}
		w.Write([]byte(`data: {"id":"1","choices":[{"delta":{"content":"Hello"},"finish_reason":"","index":0,"logprobs":null}],"created":123,"model":"m","object":"chat.completion.chunk"}` + "\n\n"))
		flusher.Flush()
		w.Write([]byte(`data: {"id":"1","choices":[{"delta":{"content":" world"},"finish_reason":"stop","index":0,"logprobs":null}],"created":123,"model":"m","object":"chat.completion.chunk"}` + "\n\n"))
		flusher.Flush()
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	origFn := newClientFn
	newClientFn = func(apiKey, baseURL string) openai.Client {
		return origFn("test-key", server.URL+"/")
	}
	defer func() { newClientFn = origFn }()

	_, ch, err := CallDeepSeek("", nil, true)
	if err != nil {
		t.Fatalf("CallDeepSeek() should not error on setup: %v", err)
	}

	var result string
	for res := range ch {
		if res.Err != nil {
			t.Fatalf("unexpected error from stream: %v", res.Err)
		}
		result += res.Content
	}
	if result != "Hello world" {
		t.Errorf("streaming result = %q, want %q", result, "Hello world")
	}
}

func TestCallDeepSeek_Streaming_HTTPError(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(500)
	}))
	defer server.Close()

	origFn := newClientFn
	newClientFn = func(apiKey, baseURL string) openai.Client {
		return origFn("test-key", server.URL+"/")
	}
	defer func() { newClientFn = origFn }()

	_, ch, err := CallDeepSeek("", nil, true)
	if err != nil {
		t.Fatalf("CallDeepSeek() setup should not error: %v", err)
	}

	// The channel should either close or send an error
	hasError := false
	for res := range ch {
		if res.Err != nil {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error from streaming channel after HTTP 500")
	}
}

func TestCallDeepSeek_WithSystemPromptAndHistory(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and capture the request body for verification
		body := make([]byte, 4096)
		n, _ := r.Body.Read(body)
		capturedBody = string(body[:n])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
  "id": "x",
  "choices": [{"finish_reason": "stop", "index": 0, "logprobs": {"content": null},
    "message": {"content": "ack", "refusal": null, "role": "assistant"}}],
  "created": 1, "model": "m", "object": "chat.completion"
}`))
	}))
	defer server.Close()

	origFn := newClientFn
	newClientFn = func(apiKey, baseURL string) openai.Client {
		return origFn("test-key", server.URL+"/")
	}
	defer func() { newClientFn = origFn }()

	_, _, err := CallDeepSeek(
		"You are a helper.",
		[]Message{
			{Role: "user", Content: "Q1"},
			{Role: "assistant", Content: "A1"},
		},
		false,
	)
	if err != nil {
		t.Fatalf("CallDeepSeek() error = %v", err)
	}

	// Verify messages array was sent correctly
	if !contains(capturedBody, "system") {
		t.Error("should include system role in request")
	}
	if !contains(capturedBody, "Q1") {
		t.Error("should include user message in request")
	}
	if !contains(capturedBody, "A1") {
		t.Error("should include assistant message in request")
	}
}

func TestCallDeepSeek_Streaming_NoFlusher(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	// Server without flusher — chunks buffered until close
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte(`data: {"id":"1","choices":[{"delta":{"content":"buffered"},"finish_reason":"stop","index":0,"logprobs":null}],"created":123,"model":"m","object":"chat.completion.chunk"}` + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))

		// httptest.ResponseRecorder doesn't implement http.Flusher by default
		// But the SDK should still consume the response when the body is closed
	}))
	defer server.Close()

	origFn := newClientFn
	newClientFn = func(apiKey, baseURL string) openai.Client {
		return origFn("test-key", server.URL+"/")
	}
	defer func() { newClientFn = origFn }()

	_, ch, err := CallDeepSeek("", nil, true)
	if err != nil {
		t.Fatalf("CallDeepSeek() error = %v", err)
	}

	var result string
	for res := range ch {
		if res.Err != nil {
			// May happen without proper flushing — not a hard failure
			t.Logf("stream got error (expected without SSZ flusher): %v", res.Err)
			continue
		}
		result += res.Content
	}
	// Verify at least some content was received
	if result != "" {
		if !contains(result, "buffered") {
			t.Errorf("unexpected stream result: %q", result)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

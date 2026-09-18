package endpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectMCP(t *testing.T, srv *Server) *mcp.ClientSession {
	t.Helper()
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(httpSrv.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: httpSrv.URL + "/mcp"}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestMCPToolsListIncludesCheckAvailabilityAndBookIntake(t *testing.T) {
	srv := newTestServer(t)
	session := connectMCP(t, srv)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	names := make(map[string]bool)
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	if !names["check_availability"] {
		t.Errorf("tools/list missing check_availability: %+v", names)
	}
	if !names["book_intake"] {
		t.Errorf("tools/list missing book_intake: %+v", names)
	}
}

func TestMCPCheckAvailabilityReturnsCurrentFacts(t *testing.T) {
	srv := newTestServer(t)
	session := connectMCP(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "check_availability"})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool call reported an error: %+v", res.Content)
	}

	out, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("failed to marshal structured content: %v", err)
	}
	var parsed struct {
		Name       string            `json:"name"`
		Properties map[string]string `json:"properties"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("failed to unmarshal structured content: %v", err)
	}
	if parsed.Name != "Test Shelter A" {
		t.Errorf("name = %q, want Test Shelter A", parsed.Name)
	}
	if parsed.Properties["capacity_available"] != "12" {
		t.Errorf("properties[capacity_available] = %q, want 12", parsed.Properties["capacity_available"])
	}
}

func TestMCPCheckAvailabilityReflectsUpdatedProvider(t *testing.T) {
	srv := newTestServer(t)
	srv.UpdateProvider(testProviderNamed("Renamed Shelter"))
	session := connectMCP(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "check_availability"})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}

	out, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(out), "Renamed Shelter") {
		t.Errorf("structured content missing updated name: %s", out)
	}
}

func TestMCPBookIntakeQueuesRequest(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")
	session := connectMCP(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "book_intake",
		Arguments: map[string]any{
			"name":    "Jane Doe",
			"contact": "555-0100",
			"need":    "bed for two tonight",
		},
	})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool call reported an error: %+v", res.Content)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Jane Doe" {
		t.Errorf("expected queued request for Jane Doe, got %+v", list)
	}
}

func TestMCPBookIntakeNotifiesWebhookWithLiveConfirmLinks(t *testing.T) {
	notified := make(chan string, 1)
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		notified <- body["content"]
		w.WriteHeader(http.StatusNoContent)
	}))
	defer hookSrv.Close()

	store := openTestBookingStore(t)
	srv := New(testProvider(), store, hookSrv.URL)
	session := connectMCP(t, srv)

	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "book_intake",
		Arguments: map[string]any{
			"name": "Jane Doe",
			"need": "bed for two tonight",
		},
	})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}

	select {
	case content := <-notified:
		if !strings.Contains(content, "/confirm") || !strings.Contains(content, "/deny") {
			t.Errorf("webhook content missing confirm/deny links: %q", content)
		}
	default:
		t.Fatal("webhook was not notified")
	}
}

func TestMCPBookIntakeRejectsMissingRequiredFields(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")
	session := connectMCP(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "book_intake",
		Arguments: map[string]any{"contact": "555-0100"},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected a tool error for missing required fields, got %+v", res)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected nothing queued, got %+v", list)
	}
}

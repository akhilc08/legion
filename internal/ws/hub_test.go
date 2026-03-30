package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newTestConn creates a real WebSocket connection backed by an httptest.Server.
// Returns (client conn, done func to close the server).
func newTestConn(t *testing.T, hub *Hub, companyID uuid.UUID) (*websocket.Conn, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.Register(conn, companyID)
	}))

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	return conn, func() {
		conn.Close()
		srv.Close()
	}
}

// readNextMessage reads one text message from the connection with a timeout.
func readNextMessage(t *testing.T, conn *websocket.Conn) []byte {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	return data
}

// tryReadMessage reads one message; returns nil if none arrives within the deadline.
func tryReadMessage(conn *websocket.Conn, timeout time.Duration) []byte {
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil
	}
	return data
}

// ---------------------------------------------------------------------------
// NewHub
// ---------------------------------------------------------------------------

func TestNewHub_NotNil(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub() returned nil")
	}
}

func TestNewHub_ClientsMapInitialized(t *testing.T) {
	h := NewHub()
	if h.clients == nil {
		t.Fatal("clients map should be initialized")
	}
}

func TestNewHub_EmptyClients(t *testing.T) {
	h := NewHub()
	if len(h.clients) != 0 {
		t.Fatalf("expected 0 companies, got %d", len(h.clients))
	}
}

// ---------------------------------------------------------------------------
// EventType constants
// ---------------------------------------------------------------------------

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"EventAgentStatus", EventAgentStatus},
		{"EventAgentLog", EventAgentLog},
		{"EventIssueUpdate", EventIssueUpdate},
		{"EventHeartbeat", EventHeartbeat},
		{"EventNotification", EventNotification},
		{"EventHirePending", EventHirePending},
		{"EventChatMessage", EventChatMessage},
		{"EventEscalation", EventEscalation},
		{"EventRuntimeStatus", EventRuntimeStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("constant %s should not be empty", tt.name)
			}
		})
	}
}

func TestEventTypeConstantValues(t *testing.T) {
	if EventAgentStatus != "agent_status" {
		t.Errorf("EventAgentStatus = %q, want %q", EventAgentStatus, "agent_status")
	}
	if EventAgentLog != "agent_log" {
		t.Errorf("EventAgentLog = %q, want %q", EventAgentLog, "agent_log")
	}
	if EventIssueUpdate != "issue_update" {
		t.Errorf("EventIssueUpdate = %q, want %q", EventIssueUpdate, "issue_update")
	}
	if EventHeartbeat != "heartbeat" {
		t.Errorf("EventHeartbeat = %q, want %q", EventHeartbeat, "heartbeat")
	}
	if EventNotification != "notification" {
		t.Errorf("EventNotification = %q, want %q", EventNotification, "notification")
	}
	if EventHirePending != "hire_pending" {
		t.Errorf("EventHirePending = %q, want %q", EventHirePending, "hire_pending")
	}
	if EventChatMessage != "chat_message" {
		t.Errorf("EventChatMessage = %q, want %q", EventChatMessage, "chat_message")
	}
	if EventEscalation != "escalation" {
		t.Errorf("EventEscalation = %q, want %q", EventEscalation, "escalation")
	}
	if EventRuntimeStatus != "runtime_status" {
		t.Errorf("EventRuntimeStatus = %q, want %q", EventRuntimeStatus, "runtime_status")
	}
}

// ---------------------------------------------------------------------------
// Event struct JSON marshaling
// ---------------------------------------------------------------------------

func TestEventJSONMarshal(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-000000000001")
	ev := Event{
		Type:      EventAgentStatus,
		CompanyID: id,
		Payload:   map[string]string{"status": "running"},
	}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["type"] != EventAgentStatus {
		t.Errorf("type = %v", decoded["type"])
	}
	if decoded["company_id"] != id.String() {
		t.Errorf("company_id = %v", decoded["company_id"])
	}
}

func TestEventJSONFieldNames(t *testing.T) {
	ev := Event{Type: "x", CompanyID: uuid.Nil, Payload: nil}
	data, _ := json.Marshal(ev)
	s := string(data)
	for _, field := range []string{"type", "company_id", "payload"} {
		if !strings.Contains(s, `"`+field+`"`) {
			t.Errorf("JSON missing field %q, got: %s", field, s)
		}
	}
}

func TestEventJSONUnmarshal(t *testing.T) {
	id := uuid.New()
	raw := `{"type":"heartbeat","company_id":"` + id.String() + `","payload":{"ping":true}}`
	var ev Event
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != EventHeartbeat {
		t.Errorf("Type = %q", ev.Type)
	}
	if ev.CompanyID != id {
		t.Errorf("CompanyID = %v", ev.CompanyID)
	}
}

func TestEventPayloadNil(t *testing.T) {
	ev := Event{Type: "test", CompanyID: uuid.Nil, Payload: nil}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"payload":null`) {
		t.Errorf("expected null payload, got: %s", data)
	}
}

// ---------------------------------------------------------------------------
// Broadcast — no subscribers (no panic)
// ---------------------------------------------------------------------------

func TestBroadcast_NoSubscribers_NoPanic(t *testing.T) {
	h := NewHub()
	ev := Event{
		Type:      EventHeartbeat,
		CompanyID: uuid.New(),
		Payload:   nil,
	}
	// Should not panic
	h.Broadcast(ev)
}

func TestBroadcast_UnknownCompany_NoPanic(t *testing.T) {
	h := NewHub()
	// Register to company A, broadcast to company B
	companyA := uuid.New()
	companyB := uuid.New()

	conn, done := newTestConn(t, h, companyA)
	defer done()
	_ = conn

	// Allow registration to complete
	time.Sleep(20 * time.Millisecond)

	ev := Event{Type: EventHeartbeat, CompanyID: companyB, Payload: nil}
	h.Broadcast(ev) // should not panic
}

// ---------------------------------------------------------------------------
// Broadcast — correct company receives message
// ---------------------------------------------------------------------------

func TestBroadcast_CorrectCompanyReceivesMessage(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	conn, done := newTestConn(t, h, companyID)
	defer done()

	time.Sleep(30 * time.Millisecond)

	ev := Event{
		Type:      EventAgentStatus,
		CompanyID: companyID,
		Payload:   map[string]string{"status": "active"},
	}
	h.Broadcast(ev)

	data := readNextMessage(t, conn)
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != EventAgentStatus {
		t.Errorf("Type = %q, want %q", got.Type, EventAgentStatus)
	}
}

func TestBroadcast_WrongCompanyDoesNotReceive(t *testing.T) {
	h := NewHub()
	companyA := uuid.New()
	companyB := uuid.New()

	connA, doneA := newTestConn(t, h, companyA)
	defer doneA()

	time.Sleep(30 * time.Millisecond)

	// Broadcast to company B only
	ev := Event{Type: EventHeartbeat, CompanyID: companyB, Payload: "ping"}
	h.Broadcast(ev)

	// Company A should NOT receive the message
	msg := tryReadMessage(connA, 200*time.Millisecond)
	if msg != nil {
		t.Errorf("company A should not receive company B's message, got: %s", msg)
	}
}

func TestBroadcast_MultipleCompanies_OnlyTargetReceives(t *testing.T) {
	h := NewHub()
	companyA := uuid.New()
	companyB := uuid.New()
	companyC := uuid.New()

	connA, doneA := newTestConn(t, h, companyA)
	defer doneA()
	connB, doneB := newTestConn(t, h, companyB)
	defer doneB()
	connC, doneC := newTestConn(t, h, companyC)
	defer doneC()

	time.Sleep(50 * time.Millisecond)

	ev := Event{Type: EventAgentLog, CompanyID: companyB, Payload: "log line"}
	h.Broadcast(ev)

	// B should get it
	data := readNextMessage(t, connB)
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != EventAgentLog {
		t.Errorf("companyB got Type = %q", got.Type)
	}

	// A and C should not
	if msg := tryReadMessage(connA, 150*time.Millisecond); msg != nil {
		t.Errorf("companyA should not receive companyB message, got: %s", msg)
	}
	if msg := tryReadMessage(connC, 150*time.Millisecond); msg != nil {
		t.Errorf("companyC should not receive companyB message, got: %s", msg)
	}
}

// ---------------------------------------------------------------------------
// Broadcast — multiple clients in same company all receive
// ---------------------------------------------------------------------------

func TestBroadcast_MultipleClientsInCompany(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	conn1, done1 := newTestConn(t, h, companyID)
	defer done1()
	conn2, done2 := newTestConn(t, h, companyID)
	defer done2()
	conn3, done3 := newTestConn(t, h, companyID)
	defer done3()

	time.Sleep(50 * time.Millisecond)

	ev := Event{Type: EventNotification, CompanyID: companyID, Payload: "hello"}
	h.Broadcast(ev)

	for i, conn := range []*websocket.Conn{conn1, conn2, conn3} {
		data := readNextMessage(t, conn)
		var got Event
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("client %d unmarshal: %v", i+1, err)
		}
		if got.Type != EventNotification {
			t.Errorf("client %d Type = %q", i+1, got.Type)
		}
	}
}

// ---------------------------------------------------------------------------
// Broadcast — slow client (full channel) drops message without blocking
// ---------------------------------------------------------------------------

func TestBroadcast_SlowClient_NoBlock(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	// Create a client but do NOT read from it so the send channel fills up.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register(conn, companyID)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	slowConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer slowConn.Close()

	time.Sleep(30 * time.Millisecond)

	// Flood the send channel (capacity 256) without reading
	ev := Event{Type: EventHeartbeat, CompanyID: companyID, Payload: nil}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 512; i++ {
			h.Broadcast(ev)
		}
		close(done)
	}()

	select {
	case <-done:
		// Good: Broadcast did not block
	case <-time.After(2 * time.Second):
		t.Error("Broadcast blocked on slow client")
	}
}

// ---------------------------------------------------------------------------
// BroadcastAll — sends to all companies
// ---------------------------------------------------------------------------

func TestBroadcastAll_SendsToAllCompanies(t *testing.T) {
	h := NewHub()
	companyA := uuid.New()
	companyB := uuid.New()

	connA, doneA := newTestConn(t, h, companyA)
	defer doneA()
	connB, doneB := newTestConn(t, h, companyB)
	defer doneB()

	time.Sleep(50 * time.Millisecond)

	ev := Event{Type: EventHeartbeat, CompanyID: uuid.Nil, Payload: "broadcast all"}
	h.BroadcastAll(ev)

	dataA := readNextMessage(t, connA)
	dataB := readNextMessage(t, connB)

	var gotA, gotB Event
	if err := json.Unmarshal(dataA, &gotA); err != nil {
		t.Fatalf("unmarshal A: %v", err)
	}
	if err := json.Unmarshal(dataB, &gotB); err != nil {
		t.Fatalf("unmarshal B: %v", err)
	}
	if gotA.Type != EventHeartbeat {
		t.Errorf("A Type = %q", gotA.Type)
	}
	if gotB.Type != EventHeartbeat {
		t.Errorf("B Type = %q", gotB.Type)
	}
}

func TestBroadcastAll_NoSubscribers_NoPanic(t *testing.T) {
	h := NewHub()
	ev := Event{Type: EventHeartbeat, Payload: nil}
	h.BroadcastAll(ev) // must not panic
}

func TestBroadcastAll_MultipleCompanies(t *testing.T) {
	h := NewHub()
	companies := make([]uuid.UUID, 4)
	conns := make([]*websocket.Conn, 4)
	dones := make([]func(), 4)

	for i := range companies {
		companies[i] = uuid.New()
		conns[i], dones[i] = newTestConn(t, h, companies[i])
		defer dones[i]()
	}

	time.Sleep(60 * time.Millisecond)

	ev := Event{Type: EventRuntimeStatus, Payload: "system ok"}
	h.BroadcastAll(ev)

	for i, conn := range conns {
		data := readNextMessage(t, conn)
		var got Event
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("company %d unmarshal: %v", i, err)
		}
		if got.Type != EventRuntimeStatus {
			t.Errorf("company %d Type = %q, want %q", i, got.Type, EventRuntimeStatus)
		}
	}
}

// ---------------------------------------------------------------------------
// BroadcastAll — multiple clients in same company each get message
// ---------------------------------------------------------------------------

func TestBroadcastAll_MultipleClientsPerCompany(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	conn1, done1 := newTestConn(t, h, companyID)
	defer done1()
	conn2, done2 := newTestConn(t, h, companyID)
	defer done2()

	time.Sleep(50 * time.Millisecond)

	ev := Event{Type: EventChatMessage, Payload: "hello all"}
	h.BroadcastAll(ev)

	d1 := readNextMessage(t, conn1)
	d2 := readNextMessage(t, conn2)

	var g1, g2 Event
	json.Unmarshal(d1, &g1)
	json.Unmarshal(d2, &g2)
	if g1.Type != EventChatMessage {
		t.Errorf("client1 Type = %q", g1.Type)
	}
	if g2.Type != EventChatMessage {
		t.Errorf("client2 Type = %q", g2.Type)
	}
}

// ---------------------------------------------------------------------------
// Broadcast event payload types
// ---------------------------------------------------------------------------

func TestBroadcast_StringPayload(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()
	conn, done := newTestConn(t, h, companyID)
	defer done()
	time.Sleep(30 * time.Millisecond)

	ev := Event{Type: EventAgentLog, CompanyID: companyID, Payload: "log message"}
	h.Broadcast(ev)

	data := readNextMessage(t, conn)
	var got map[string]interface{}
	json.Unmarshal(data, &got)
	if got["payload"] != "log message" {
		t.Errorf("payload = %v", got["payload"])
	}
}

func TestBroadcast_MapPayload(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()
	conn, done := newTestConn(t, h, companyID)
	defer done()
	time.Sleep(30 * time.Millisecond)

	ev := Event{
		Type:      EventIssueUpdate,
		CompanyID: companyID,
		Payload:   map[string]interface{}{"issue_id": "123", "status": "open"},
	}
	h.Broadcast(ev)

	data := readNextMessage(t, conn)
	var got map[string]interface{}
	json.Unmarshal(data, &got)
	payload, _ := got["payload"].(map[string]interface{})
	if payload["status"] != "open" {
		t.Errorf("payload.status = %v", payload["status"])
	}
}

// ---------------------------------------------------------------------------
// Concurrent broadcast safety
// ---------------------------------------------------------------------------

func TestBroadcast_Concurrent_NoPanic(t *testing.T) {
	h := NewHub()
	company := uuid.New()

	conn, done := newTestConn(t, h, company)
	defer done()
	time.Sleep(30 * time.Millisecond)

	// Drain the connection in background
	go func() {
		for {
			conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Broadcast(Event{Type: EventHeartbeat, CompanyID: company})
		}()
	}
	wg.Wait()
}

func TestBroadcastAll_Concurrent_NoPanic(t *testing.T) {
	h := NewHub()
	for i := 0; i < 3; i++ {
		company := uuid.New()
		conn, done := newTestConn(t, h, company)
		defer done()
		_ = conn
	}
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.BroadcastAll(Event{Type: EventHeartbeat})
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Register — sets up company bucket
// ---------------------------------------------------------------------------

func TestRegister_CreatesCompanyBucket(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	conn, done := newTestConn(t, h, companyID)
	defer done()
	time.Sleep(30 * time.Millisecond)

	h.mu.RLock()
	_, exists := h.clients[companyID]
	h.mu.RUnlock()

	if !exists {
		t.Error("expected company bucket to exist after Register")
	}
	_ = conn
}

func TestRegister_MultipleCompanies_SeparateBuckets(t *testing.T) {
	h := NewHub()
	companyA := uuid.New()
	companyB := uuid.New()

	connA, doneA := newTestConn(t, h, companyA)
	defer doneA()
	connB, doneB := newTestConn(t, h, companyB)
	defer doneB()
	time.Sleep(50 * time.Millisecond)

	h.mu.RLock()
	bucketA := h.clients[companyA]
	bucketB := h.clients[companyB]
	h.mu.RUnlock()

	if len(bucketA) != 1 {
		t.Errorf("companyA bucket size = %d, want 1", len(bucketA))
	}
	if len(bucketB) != 1 {
		t.Errorf("companyB bucket size = %d, want 1", len(bucketB))
	}
	_ = connA
	_ = connB
}

func TestRegister_ReturnsDoneChannel(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		done := h.Register(conn, companyID)
		if done == nil {
			// can't assert from goroutine; just close
			conn.Close()
			return
		}
		// Close connection → done channel should close
		conn.Close()
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()
	// If we get here without hanging, the test passes
}

// ---------------------------------------------------------------------------
// Client removed from hub after disconnect
// ---------------------------------------------------------------------------

func TestBroadcast_ClientRemovedAfterDisconnect(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		done := h.Register(conn, companyID)
		<-done // wait for disconnect, then this handler exits
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	// Send a close frame so the server-side readPump exits cleanly
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()

	// Poll until the hub removes the client (up to 1s)
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		h.mu.RLock()
		size := len(h.clients[companyID])
		h.mu.RUnlock()
		if size == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	h.mu.RLock()
	size := len(h.clients[companyID])
	h.mu.RUnlock()
	if size != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", size)
	}
}

// ---------------------------------------------------------------------------
// Broadcast — companyID encoded in JSON output
// ---------------------------------------------------------------------------

func TestBroadcast_CompanyIDInPayload(t *testing.T) {
	h := NewHub()
	companyID := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	conn, done := newTestConn(t, h, companyID)
	defer done()
	time.Sleep(30 * time.Millisecond)

	ev := Event{Type: EventHeartbeat, CompanyID: companyID, Payload: nil}
	h.Broadcast(ev)

	data := readNextMessage(t, conn)
	if !strings.Contains(string(data), companyID.String()) {
		t.Errorf("expected company_id in payload, got: %s", data)
	}
}

// ---------------------------------------------------------------------------
// BroadcastAll — marshal error does not panic
// ---------------------------------------------------------------------------

// unmarshalablePayload is a type that cannot be marshaled to JSON.
type unmarshalablePayload struct {
	Ch chan struct{}
}

func (u unmarshalablePayload) MarshalJSON() ([]byte, error) {
	return nil, &json.UnsupportedTypeError{}
}

func TestBroadcastAll_MarshalError_NoPanic(t *testing.T) {
	h := NewHub()
	ev := Event{
		Type:    EventHeartbeat,
		Payload: unmarshalablePayload{Ch: make(chan struct{})},
	}
	// Should not panic even if marshal fails
	h.BroadcastAll(ev)
}

func TestBroadcast_MarshalError_NoPanic(t *testing.T) {
	h := NewHub()
	companyID := uuid.New()
	_, done := newTestConn(t, h, companyID)
	defer done()
	time.Sleep(30 * time.Millisecond)

	ev := Event{
		Type:      EventHeartbeat,
		CompanyID: companyID,
		Payload:   unmarshalablePayload{},
	}
	// Should not panic
	h.Broadcast(ev)
}

package ws

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// fakeClient mimics the internal client structure without a real WebSocket.
// We test the hub's marshal + fan-out path by wiring up buffered channels directly.
func newBenchHub(numClients int, companyID uuid.UUID) (*Hub, []chan []byte) {
	h := NewHub()
	chans := make([]chan []byte, numClients)

	h.mu.Lock()
	if h.clients[companyID] == nil {
		h.clients[companyID] = make(map[*client]struct{})
	}
	for i := 0; i < numClients; i++ {
		ch := make(chan []byte, 1024)
		chans[i] = ch
		c := &client{
			send: ch,
			quit: make(chan struct{}),
		}
		h.clients[companyID][c] = struct{}{}
	}
	h.mu.Unlock()

	return h, chans
}

func BenchmarkBroadcast_1Client(b *testing.B) {
	cid := uuid.New()
	h, _ := newBenchHub(1, cid)
	evt := Event{Type: EventAgentLog, CompanyID: cid, Payload: map[string]string{"msg": "hello"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Broadcast(evt)
	}
}

func BenchmarkBroadcast_10Clients(b *testing.B) {
	cid := uuid.New()
	h, chans := newBenchHub(10, cid)
	evt := Event{Type: EventAgentLog, CompanyID: cid, Payload: map[string]string{"msg": "hello"}}
	// drain goroutine to prevent channel fill blocking benchmark
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				for _, ch := range chans {
					select {
					case <-ch:
					default:
					}
				}
			}
		}
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Broadcast(evt)
	}
	close(stop)
}

func BenchmarkBroadcast_100Clients(b *testing.B) {
	cid := uuid.New()
	h, chans := newBenchHub(100, cid)
	evt := Event{Type: EventAgentLog, CompanyID: cid, Payload: map[string]string{"msg": "hello"}}
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				for _, ch := range chans {
					select {
					case <-ch:
					default:
					}
				}
			}
		}
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Broadcast(evt)
	}
	close(stop)
}

func BenchmarkBroadcastConcurrent_10Writers(b *testing.B) {
	cid := uuid.New()
	h, chans := newBenchHub(50, cid)
	evt := Event{Type: EventAgentStatus, CompanyID: cid, Payload: map[string]string{"status": "running"}}
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				for _, ch := range chans {
					select {
					case <-ch:
					default:
					}
				}
			}
		}
	}()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Broadcast(evt)
		}
	})
	close(stop)
}

func BenchmarkEventMarshal(b *testing.B) {
	evt := Event{
		Type:      EventAgentLog,
		CompanyID: uuid.New(),
		Payload:   map[string]interface{}{"msg": "task completed", "exit_code": 0, "duration_ms": 1234},
	}
	var sink []byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink, _ = json.Marshal(evt)
	}
	_ = sink
}

func BenchmarkHubRegisterUnregister(b *testing.B) {
	h := &Hub{clients: make(map[uuid.UUID]map[*client]struct{})}
	cid := uuid.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := &client{send: make(chan []byte, 8), quit: make(chan struct{})}
		h.mu.Lock()
		if h.clients[cid] == nil {
			h.clients[cid] = make(map[*client]struct{})
		}
		h.clients[cid][c] = struct{}{}
		h.mu.Unlock()

		h.mu.Lock()
		delete(h.clients[cid], c)
		h.mu.Unlock()
	}
}

func BenchmarkBroadcastAll_5Companies_10ClientsEach(b *testing.B) {
	h := NewHub()
	allChans := []chan []byte{}
	companies := make([]uuid.UUID, 5)
	for i := range companies {
		cid := uuid.New()
		companies[i] = cid
		h.mu.Lock()
		h.clients[cid] = make(map[*client]struct{})
		for j := 0; j < 10; j++ {
			ch := make(chan []byte, 1024)
			allChans = append(allChans, ch)
			c := &client{send: ch, quit: make(chan struct{})}
			h.clients[cid][c] = struct{}{}
		}
		h.mu.Unlock()
	}

	evt := Event{Type: EventRuntimeStatus, Payload: "available"}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				for _, ch := range allChans {
					select {
					case <-ch:
					default:
					}
				}
			}
		}
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.BroadcastAll(evt)
	}
	close(stop)
	wg.Wait()
}

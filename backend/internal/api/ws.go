package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) run() {
	// broadcast recent run summary every 3s
	for range time.Tick(3 * time.Second) {
		h.broadcast(map[string]string{"type": "ping", "ts": time.Now().Format(time.RFC3339)})
	}
}

func (h *Hub) broadcast(msg interface{}) {
	data, _ := json.Marshal(msg)
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		_ = wsjson.Write(context.Background(), conn, data)
	}
}

func (h *Handler) handleWS(c *gin.Context) {
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	h.wsHub.mu.Lock()
	h.wsHub.clients[conn] = struct{}{}
	h.wsHub.mu.Unlock()

	defer func() {
		h.wsHub.mu.Lock()
		delete(h.wsHub.clients, conn)
		h.wsHub.mu.Unlock()
	}()

	// block until client disconnects
	_, _, _ = conn.Read(c.Request.Context())
}

package websocket

import (
	"encoding/json"
	"time"

	gorillaWS "github.com/gorilla/websocket"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 20 * 1024 * 1024
	chunkSize      = 1024 * 1024
)

type Client struct {
	hub      *Hub
	conn     *gorillaWS.Conn
	userID   string
	username string
	send     chan []byte
	log      *logger.Logger
}

func NewClient(hub *Hub, conn *gorillaWS.Conn, userID, username string, log *logger.Logger) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		userID:   userID,
		username: username,
		send:     make(chan []byte, 256),
		log:      log,
	}
}

func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if gorillaWS.IsUnexpectedCloseError(err, gorillaWS.CloseGoingAway, gorillaWS.CloseAbnormalClosure) {
				c.log.Warnf("websocket read error user_id=%s username=%s: %v", c.userID, c.username, err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			c.log.Warnf("websocket invalid message user_id=%s username=%s: %v", c.userID, c.username, err)
			continue
		}

		c.hub.HandleMessage(c, &msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(gorillaWS.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(gorillaWS.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(gorillaWS.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

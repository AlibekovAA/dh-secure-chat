package websocket

import (
	"encoding/json"
	"time"

	gorillaWS "github.com/gorilla/websocket"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
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
	hub        *Hub
	conn       *gorillaWS.Conn
	userID     string
	username   string
	send       chan []byte
	log        *logger.Logger
	authenticated bool
	jwtSecret  []byte
}

func NewClient(hub *Hub, conn *gorillaWS.Conn, userID, username string, log *logger.Logger) *Client {
	return &Client{
		hub:          hub,
		conn:         conn,
		userID:       userID,
		username:     username,
		send:         make(chan []byte, 256),
		log:          log,
		authenticated: true,
	}
}

func NewUnauthenticatedClient(hub *Hub, conn *gorillaWS.Conn, jwtSecret string, log *logger.Logger) *Client {
	return &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256),
		log:          log,
		authenticated: false,
		jwtSecret:    []byte(jwtSecret),
	}
}

func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		if c.authenticated {
			c.hub.Unregister(c)
		}
		c.conn.Close()
	}()

	authTimeout := 10 * time.Second
	if !c.authenticated {
		c.conn.SetReadDeadline(time.Now().Add(authTimeout))
	} else {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
	}
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if gorillaWS.IsUnexpectedCloseError(err, gorillaWS.CloseGoingAway, gorillaWS.CloseAbnormalClosure) {
				if c.authenticated {
					c.log.Warnf("websocket read error user_id=%s username=%s: %v", c.userID, c.username, err)
				} else {
					c.log.Warnf("websocket read error (unauthenticated): %v", err)
				}
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			if c.authenticated {
				c.log.Warnf("websocket invalid message user_id=%s username=%s: %v", c.userID, c.username, err)
			} else {
				c.log.Warnf("websocket invalid message (unauthenticated): %v", err)
			}
			continue
		}

		if !c.authenticated {
			if msg.Type != TypeAuth {
				c.log.Warnf("websocket unauthenticated client sent non-auth message type=%s", msg.Type)
				c.conn.WriteMessage(gorillaWS.CloseMessage, gorillaWS.FormatCloseMessage(gorillaWS.ClosePolicyViolation, "authentication required"))
				break
			}

			var authPayload AuthPayload
			if err := json.Unmarshal(msg.Payload, &authPayload); err != nil {
				c.log.Warnf("websocket invalid auth payload: %v", err)
				c.conn.WriteMessage(gorillaWS.CloseMessage, gorillaWS.FormatCloseMessage(gorillaWS.CloseInvalidFramePayloadData, "invalid auth payload"))
				break
			}

			claims, err := jwtverify.ParseToken(authPayload.Token, c.jwtSecret)
			if err != nil {
				c.log.Warnf("websocket authentication failed: %v", err)
				c.conn.WriteMessage(gorillaWS.CloseMessage, gorillaWS.FormatCloseMessage(gorillaWS.ClosePolicyViolation, "invalid token"))
				break
			}

			c.userID = claims.UserID
			c.username = claims.Username
			c.authenticated = true
			c.conn.SetReadDeadline(time.Now().Add(pongWait))

			authResponse := WSMessage{
				Type: TypeAuth,
				Payload: json.RawMessage(`{"authenticated":true}`),
			}
			authResponseBytes, err := json.Marshal(authResponse)
			if err == nil {
				select {
				case c.send <- authResponseBytes:
				default:
					c.log.Warnf("websocket auth response send buffer full user_id=%s", c.userID)
				}
			}

			c.hub.Register(c)
			c.log.Infof("websocket client authenticated user_id=%s username=%s", c.userID, c.username)
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

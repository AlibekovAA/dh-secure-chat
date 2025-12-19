package websocket

import (
	"encoding/json"
	"time"

	gorillaWS "github.com/gorilla/websocket"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/jwtverify"
	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
)

type Client struct {
	hub           *Hub
	conn          *gorillaWS.Conn
	userID        string
	username      string
	send          chan []byte
	log           *logger.Logger
	authenticated bool
	jwtSecret     []byte
	writeWait     time.Duration
	pongWait      time.Duration
	pingPeriod    time.Duration
	maxMsgSize    int64
	authTimeout   time.Duration
}

func NewUnauthenticatedClient(hub *Hub, conn *gorillaWS.Conn, jwtSecret string, log *logger.Logger, writeWait, pongWait, pingPeriod time.Duration, maxMsgSize int64, authTimeout time.Duration, sendBufSize int) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, sendBufSize),
		log:           log,
		authenticated: false,
		jwtSecret:     []byte(jwtSecret),
		writeWait:     writeWait,
		pongWait:      pongWait,
		pingPeriod:    pingPeriod,
		maxMsgSize:    maxMsgSize,
		authTimeout:   authTimeout,
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

	if !c.authenticated {
		c.conn.SetReadDeadline(time.Now().Add(c.authTimeout))
	} else {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
	}
	c.conn.SetReadLimit(c.maxMsgSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
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
			c.conn.SetReadDeadline(time.Now().Add(c.pongWait))

			authResponse := WSMessage{
				Type:    TypeAuth,
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
	ticker := time.NewTicker(c.pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
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
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err := c.conn.WriteMessage(gorillaWS.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

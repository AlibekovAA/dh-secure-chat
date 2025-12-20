package websocket

import "encoding/json"

func marshalMessage(msgType MessageType, payload interface{}) (*WSMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &WSMessage{
		Type:    msgType,
		Payload: payloadBytes,
	}, nil
}

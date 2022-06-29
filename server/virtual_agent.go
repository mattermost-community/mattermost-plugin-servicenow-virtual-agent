package main

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type VirtualAgentRequestBody struct {
	Message   *MessageBody `json:"message"`
	RequestID string       `json:"requestId"`
	UserID    string       `json:"userId"`
}

type MessageBody struct {
	Text  string `json:"text"`
	Typed bool   `json:"typed"`
}

func (c *client) SendMessageToVirtualAgentAPI(userID, messageText string) error {
	requestBody := &VirtualAgentRequestBody{
		Message: &MessageBody{
			Text:  messageText,
			Typed: true, // TODO: Make this dynamic after adding support for Default Picker List
		},
		RequestID: uuid.New().String(),
		UserID:    userID,
	}

	_, err := c.CallJSON(http.MethodPost, PathVirtualAgentBotIntegration, requestBody, nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to call virtual agent bot integration API")
	}

	return nil
}

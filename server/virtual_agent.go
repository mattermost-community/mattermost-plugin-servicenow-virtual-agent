package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

type VirtualAgentRequestBody struct {
	Action    string       `json:"action"`
	Message   *MessageBody `json:"message"`
	RequestID string       `json:"requestId"`
	UserID    string       `json:"userId"`
}

type MessageBody struct {
	Text  string `json:"text"`
	Typed bool   `json:"typed"`
}

type VirtualAgentResponse struct {
	VirtualAgentRequestBody
	Body []MessageResponseBody `json:"body"`
}

type MessageResponseBody struct {
	Value interface{}
}

type OutputText struct {
	UIType        string `json:"uiType"`
	Group         string `json:"group"`
	Value         string `json:"value"`
	ItemType      string `json:"itemType"`
	MaskType      string `json:"maskType"`
	Label         string `json:"label"`
}

type OutputLinkValue struct {
	Action string `json:"action"`
}

type OutputLink struct {
	UIType string `json:"uiType"`
	Group  string `json:"group"`
	Label  string `json:"label"`
	Header string `json:"header"`
	Type   string `json:"type"`
	Value  OutputLinkValue
}

type TopicPickerControl struct {
	UIType         string   `json:"uiType"`
	Group          string   `json:"group"`
	NLUTextEnabled bool     `json:"nluTextEnabled"`
	PromptMessage  string   `json:"promptMsg"`
	Label          string   `json:"label"`
	Options        []Option `json:"options"`
}

type Picker struct {
	UIType         string   `json:"uiType"`
	Group          string   `json:"group"`
	Required       bool     `json:"required"`
	NLUTextEnabled bool     `json:"nluTextEnabled"`
	Label          string   `json:"label"`
	ItemType       string   `json:"itemType"`
	Options        []Option `json:"options"`
	Style          string   `json:"style"`
	MultiSelect    bool     `json:"multiSelect"`
}

type Option struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

func (m *MessageResponseBody) UnmarshalJSON(data []byte) error {
	var uiType struct {
		UIType string `json:"uiType"`
	}

	if err := json.Unmarshal(data, &uiType); err != nil {
		return err
	}

	switch uiType.UIType {
	case OutputTextUIType:
		m.Value = new(OutputText)
	case InputTextUIType:
		m.Value = new(OutputText)
	case TopicPickerControlUIType:
		m.Value = new(TopicPickerControl)
	case PickerUIType:
		m.Value = new(Picker)
	case BooleanUIType:
		m.Value = new(Picker)
	case OutputLinkUIType:
		m.Value = new(OutputLink)
	}

	if m.Value != nil {
		return json.Unmarshal(data, m.Value)
	}

	return nil
}

func (c *client) SendMessageToVirtualAgentAPI(userID, messageText string, typed bool) error {
	requestBody := &VirtualAgentRequestBody{
		Message: &MessageBody{
			Text:  messageText,
			Typed: typed,
		},
		RequestID: c.plugin.generateUUID(),
		UserID:    userID,
	}

	if _, err := c.CallJSON(http.MethodPost, PathVirtualAgentBotIntegration, requestBody, nil, nil); err != nil {
		return errors.Wrap(err, "failed to call virtual agent bot integration API")
	}

	return nil
}

func (c *client) StartConverstaionWithVirtualAgent(userID string) error {
	requestBody := &VirtualAgentRequestBody{
		Action:    StartConversationAction,
		RequestID: c.plugin.generateUUID(),
		UserID:    userID,
	}

	if _, err := c.CallJSON(http.MethodPost, PathVirtualAgentBotIntegration, requestBody, nil, nil); err != nil {
		return errors.Wrap(err, "failed to start conversation with virtual agent bot")
	}

	return nil
}

func (p *Plugin) ProcessResponse(data []byte) error {
	vaResponse := &VirtualAgentResponse{}
	if err := json.Unmarshal(data, &vaResponse); err != nil {
		return err
	}

	user, err := p.store.LoadUserWithSysID(vaResponse.UserID)
	if err != nil {
		return err
	}

	userID := user.MattermostUserID
	for _, messageResponse := range vaResponse.Body {
		switch res := messageResponse.Value.(type) {
		case *OutputText:
			message := res.Value
			if res.Label != "" {
				message = res.Label
			}
			if _, err = p.DM(userID, message); err != nil {
				return err
			}
		case *TopicPickerControl:
			if _, err = p.DMWithAttachments(userID, p.CreateTopicPickerControlAttachment(res)); err != nil {
				return err
			}
		case *Picker:
			if _, err = p.DMWithAttachments(userID, p.CreatePickerAttachment(res)); err != nil {
				return err
			}
		case *OutputLink:
			if _, err = p.DMWithAttachments(userID, p.CreateOutputLinkAttachment(res)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Plugin) CreateOutputLinkAttachment(body *OutputLink) *model.SlackAttachment {
	return &model.SlackAttachment{
		Pretext: body.Header,
		Text:    fmt.Sprintf("[%s](%s)", body.Label, body.Value.Action),
	}
}

func (p *Plugin) CreateTopicPickerControlAttachment(body *TopicPickerControl) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: body.PromptMessage,
		Actions: []*model.PostAction{
			{
				Name: "Select an option...",
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathActionOptions),
				},
				Type:    "select",
				Options: p.getPostActionOptions(body.Options),
			},
		},
	}
}

func (p *Plugin) CreatePickerAttachment(body *Picker) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: body.Label,
		Actions: []*model.PostAction{
			{
				Name: "Select an option...",
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathActionOptions),
				},
				Type:    "select",
				Options: p.getPostActionOptions(body.Options),
			},
		},
	}
}

func (p *Plugin) getPostActionOptions(options []Option) []*model.PostActionOptions {
	var postOptions []*model.PostActionOptions
	for _, option := range options {
		postOptions = append(postOptions, &model.PostActionOptions{
			Text:  option.Label,
			Value: option.Value,
		})
	}

	return postOptions
}

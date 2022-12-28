package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
)

type MessageResponseBody struct {
	Value interface{}
}

type VirtualAgentResponse struct {
	serializer.VirtualAgentRequestBody
	Body []MessageResponseBody `json:"body"`
}

func (m *MessageResponseBody) UnmarshalJSON(data []byte) error {
	var uiType struct {
		UIType string `json:"uiType"`
	}

	if err := json.Unmarshal(data, &uiType); err != nil {
		return err
	}

	switch uiType.UIType {
	case constants.OutputTextUIType, constants.InputTextUIType, constants.FileUploadUIType:
		m.Value = new(serializer.OutputText)
	case constants.TopicPickerControlUIType:
		m.Value = new(serializer.TopicPickerControl)
	case constants.PickerUIType, constants.BooleanUIType:
		m.Value = new(serializer.Picker)
	case constants.OutputLinkUIType:
		m.Value = new(serializer.OutputLink)
	case constants.GroupedPartsOutputControlUIType:
		m.Value = new(serializer.GroupedPartsOutputControl)
	case constants.OutputCardUIType:
		m.Value = new(serializer.OutputCard)
	case constants.OutputImageUIType:
		m.Value = new(serializer.OutputImage)
	case constants.DateTimeUIType, constants.DateUIType, constants.TimeUIType:
		m.Value = new(serializer.DefaultDate)
	}

	if m.Value != nil {
		return json.Unmarshal(data, m.Value)
	}

	return nil
}

func (c *client) SendMessageToVirtualAgentAPI(serviceNowUserID, messageText string, typed bool, attachment *serializer.MessageAttachment) error {
	requestBody := &serializer.VirtualAgentRequestBody{
		Message: &serializer.MessageBody{
			Attachment: attachment,
			Text:       messageText,
			Typed:      typed,
		},
		RequestID: c.plugin.generateUUID(),
		UserID:    serviceNowUserID,
	}

	if _, err := c.CallJSON(http.MethodPost, constants.PathVirtualAgentBotIntegration, requestBody, nil, nil); err != nil {
		return errors.Wrap(err, "failed to call virtual agent bot integration API")
	}

	return nil
}

func (c *client) StartConverstaionWithVirtualAgent(userID string) error {
	requestBody := &serializer.VirtualAgentRequestBody{
		Action:    constants.StartConversationAction,
		RequestID: c.plugin.generateUUID(),
		UserID:    userID,
	}

	if _, err := c.CallJSON(http.MethodPost, constants.PathVirtualAgentBotIntegration, requestBody, nil, nil); err != nil {
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
	p.handlePreviousCarouselPosts(userID)
	for _, messageResponse := range vaResponse.Body {
		switch res := messageResponse.Value.(type) {
		case *serializer.OutputText:
			message := res.Value
			if res.Label != "" {
				message = res.Label
				if res.ItemType == constants.ItemTypeImage {
					message += constants.UploadImageMessage
				} else if res.ItemType == constants.ItemTypeFile {
					message += constants.UploadFileMessage
				}
			}

			if _, err = p.DM(userID, message); err != nil {
				return err
			}
		case *serializer.TopicPickerControl:
			if len(res.Options) == 0 {
				p.API.LogInfo("TopicPickerControl dropdown has no options to display.")
				return nil
			}

			if _, err = p.DMWithAttachments(userID, p.CreateTopicPickerControlAttachment(res)); err != nil {
				return err
			}
		case *serializer.Picker:
			if _, err = p.DM(userID, res.Label); err != nil {
				return err
			}
			if len(res.Options) == 0 {
				p.API.LogInfo("Picker dropdown has no options to display.")
				return nil
			}

			if res.ItemType == constants.ItemTypePicture && res.Style == constants.StyleCarousel {
				if err = p.HandleCarouselInput(userID, res); err != nil {
					return err
				}
			} else {
				if _, err = p.DMWithAttachments(userID, p.CreatePickerAttachment(res)); err != nil {
					return err
				}
			}
		case *serializer.OutputLink:
			if _, err = p.DMWithAttachments(userID, p.CreateOutputLinkAttachment(res)); err != nil {
				return err
			}
		// TODO: Modify the UI for this later.
		case *serializer.GroupedPartsOutputControl:
			if _, err = p.DM(userID, res.Header); err != nil {
				return err
			}

			for _, value := range res.Values {
				if _, err = p.DMWithAttachments(userID, p.CreateGroupedPartsOutputControlAttachment(value)); err != nil {
					return err
				}
			}
		case *serializer.OutputCard:
			switch res.TemplateName {
			case constants.OutputCardSmallImageType, constants.OutputCardLargeImageType:
				var data serializer.OutputCardImageData
				if err = json.Unmarshal([]byte(res.Data), &data); err != nil {
					return err
				}

				if _, err = p.DMWithAttachments(userID, p.CreateOutputCardImageAttachment(&data)); err != nil {
					return err
				}
			case constants.OutputCardVideoType:
				var data serializer.OutputCardVideoData
				if err = json.Unmarshal([]byte(res.Data), &data); err != nil {
					return err
				}

				if _, err = p.DMWithAttachments(userID, p.CreateOutputCardVideoAttachment(&data)); err != nil {
					return err
				}

				if _, err = p.dm(userID, &model.Post{
					Message: fmt.Sprintf(constants.YoutubeURL, data.ID),
				}); err != nil {
					return err
				}
			case constants.OutputCardRecordType:
				var data serializer.OutputCardRecordData
				if err = json.Unmarshal([]byte(res.Data), &data); err != nil {
					return err
				}

				if _, err = p.DMWithAttachments(userID, p.CreateOutputCardRecordAttachment(&data)); err != nil {
					return err
				}
			}
		case *serializer.OutputImage:
			linkContents := strings.Split(res.Value, "/")
			if len(linkContents) < 1 {
				if _, err = p.DM(userID, fmt.Sprintf("Image: %s", res.AltText)); err != nil {
					return err
				}

				p.API.LogError(constants.InvalidImageLinkError, "Link", res.Value)
				return errors.New(constants.InvalidImageLinkError)
			}

			completeFileName := linkContents[len(linkContents)-1]
			if _, err = p.DM(userID, fmt.Sprintf("![%s](%s)", completeFileName, res.Value)); err != nil {
				return err
			}
		case *serializer.DefaultDate:
			if _, err = p.DMWithAttachments(userID, p.CreateDefaultDateAttachment(res)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Plugin) CreateDefaultDateAttachment(body *serializer.DefaultDate) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: body.Label,
		Actions: []*model.PostAction{
			{
				Name: fmt.Sprintf("Set %s", body.UIType),
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathSetDateTimeDialog),
					Context: map[string]interface{}{
						"type": body.UIType,
					},
				},
				Type: model.POST_ACTION_TYPE_BUTTON,
			},
		},
	}
}

func (p *Plugin) CreateOutputLinkAttachment(body *serializer.OutputLink) *model.SlackAttachment {
	return &model.SlackAttachment{
		Pretext: body.Header,
		Text:    fmt.Sprintf("[%s](%s)", body.Label, body.Value.Action),
	}
}

func (p *Plugin) CreateOutputCardImageAttachment(body *serializer.OutputCardImageData) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text:     fmt.Sprintf("**%s**\n%s", body.Title, body.Description),
		ImageURL: body.Image,
	}
}

func (p *Plugin) CreateOutputCardVideoAttachment(body *serializer.OutputCardVideoData) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: fmt.Sprintf("**[%s](%s)**\n%s", body.Title, body.Link, body.Description),
	}
}

func (p *Plugin) CreateOutputCardRecordAttachment(body *serializer.OutputCardRecordData) *model.SlackAttachment {
	fields := make([]*model.SlackAttachmentField, len(body.Fields)+1)
	fields[0] = &model.SlackAttachmentField{
		Title: body.Title,
		Value: fmt.Sprintf("[%s](%s)", body.Subtitle, body.URL),
	}
	for index, field := range body.Fields {
		fields[index+1] = &model.SlackAttachmentField{
			Title: field.FieldLabel,
			Value: field.FieldValue,
		}
	}
	return &model.SlackAttachment{
		Fields: fields,
	}
}

func (p *Plugin) CreateGroupedPartsOutputControlAttachment(body serializer.GroupedPartsOutputControlValue) *model.SlackAttachment {
	return &model.SlackAttachment{
		Title: fmt.Sprintf("[%s](%s)", body.Label, body.Action),
		Text:  body.Description,
	}
}

func (p *Plugin) CreateTopicPickerControlAttachment(body *serializer.TopicPickerControl) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: body.PromptMessage,
		Actions: []*model.PostAction{
			{
				Name: "Select an option...",
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathActionOptions),
				},
				Type:    model.POST_ACTION_TYPE_SELECT,
				Options: p.getPostActionOptions(body.Options),
			},
		},
	}
}

func (p *Plugin) CreatePickerAttachment(body *serializer.Picker) *model.SlackAttachment {
	return &model.SlackAttachment{
		Actions: []*model.PostAction{
			{
				Name: "Select an option...",
				Integration: &model.PostActionIntegration{
					URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathActionOptions),
				},
				Type:    model.POST_ACTION_TYPE_SELECT,
				Options: p.getPostActionOptions(body.Options),
			},
		},
	}
}

func (p *Plugin) handlePreviousCarouselPosts(userID string) {
	postIDs, err := p.store.LoadPostIDs(userID)
	if err != nil {
		p.API.LogDebug("Unable to load the post IDs from KV store", "UserID", userID, "Error", err.Error())
		return
	}

	if len(postIDs) == 0 {
		return
	}

	if err = p.store.StorePostIDs(userID, make([]string, 0)); err != nil {
		p.API.LogDebug("Unable to store the post IDs in KV store", "UserID", userID, "Error", err.Error())
	}

	// TODO: Think of a better way to do this
	// Adding wait group to avoid race conditions in unit tests
	wg := sync.WaitGroup{}
	for _, postID := range postIDs {
		wg.Add(1)
		go func(postID string) {
			defer wg.Done()
			post, err := p.API.GetPost(postID)
			if err != nil {
				p.API.LogDebug("Unable to get the post", "PostID", postID, "Error", err.Error())
			}

			if post == nil {
				return
			}

			attachments := post.Attachments()
			for _, attachment := range attachments {
				attachment.Actions = nil
			}

			model.ParseSlackAttachment(post, attachments)
			if _, err = p.API.UpdatePost(post); err != nil {
				p.API.LogDebug("Unable to update the post", "PostID", postID, "Error", err.Error())
			}
		}(postID)
	}
	wg.Wait()
}

func (p *Plugin) HandleCarouselInput(userID string, body *serializer.Picker) error {
	postIDs := make([]string, 0)
	idx := 0
	for {
		var attachments []*model.SlackAttachment
		for i := idx; i < len(body.Options); i++ {
			option := body.Options[i]
			attachments = append(attachments, &model.SlackAttachment{
				Title:    fmt.Sprintf("%v) %s", i+1, option.Label),
				Text:     option.Description,
				ImageURL: option.Attachment,
				Actions: []*model.PostAction{
					{
						Name: "Select",
						Type: model.POST_ACTION_TYPE_BUTTON,
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathActionOptions),
							Context: map[string]interface{}{
								constants.ContextKeySelectedLabel: fmt.Sprintf("%v) %s", i+1, option.Label),
								constants.ContextKeySelectedValue: option.Value,
								constants.StyleCarousel:           true,
							},
						},
					},
				},
			})

			if !p.IsCharCountSafe(attachments) {
				attachments = attachments[:len(attachments)-1]
				idx = i
				break
			}

			if i == len(body.Options)-1 {
				idx = 0
			}
		}

		postID, err := p.DMWithAttachments(userID, attachments...)
		if err != nil {
			return err
		}

		postIDs = append(postIDs, postID)
		if idx == 0 {
			break
		}
	}

	if err := p.store.StorePostIDs(userID, postIDs); err != nil {
		p.API.LogDebug("Unable to store the post IDs in KV store", "UserID", userID, "Error", err.Error())
	}

	return nil
}

func (p *Plugin) IsCharCountSafe(attachments []*model.SlackAttachment) bool {
	bytes, err := json.Marshal(attachments)
	if err != nil {
		p.API.LogDebug("Error in marshaling the attachments", "Error", err.Error())
	}
	// 35 is the approx. length of one line added by the MM server for post action IDs and 100 is a buffer
	return utf8.RuneCountInString(string(bytes)) < model.POST_PROPS_MAX_RUNES-100-(len(attachments)*35)
}

func (p *Plugin) getPostActionOptions(options []serializer.Option) []*model.PostActionOptions {
	var postOptions []*model.PostActionOptions
	for _, option := range options {
		postOptions = append(postOptions, &model.PostActionOptions{
			Text:  option.Label,
			Value: option.Label,
		})
	}

	return postOptions
}

func (p *Plugin) CreateMessageAttachment(fileID, userID string) (*serializer.MessageAttachment, error) {
	var attachment *serializer.MessageAttachment
	fileInfo, appErr := p.API.GetFileInfo(fileID)
	if appErr != nil {
		return nil, fmt.Errorf("error getting the file info. Error: %s", appErr.Message)
	}

	if fileInfo.DeleteAt != 0 {
		return nil, fmt.Errorf("file is deleted from the server")
	}

	if fileInfo.CreatorId != userID {
		return nil, fmt.Errorf("file does not belong to the Mattermost user: %s", userID)
	}

	//TODO: Add a configuration setting for expiry time
	expiryTime := time.Now().UTC().Add(time.Minute * constants.AttachmentLinkExpiryTimeInMinutes)

	file := &FileStruct{
		ID:     fileID,
		Expiry: expiryTime,
	}

	var jsonBytes []byte
	jsonBytes, err := json.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("error occurred while marshaling the file. Error: %w", err)
	}

	var encrypted []byte
	encrypted, err = encrypt(jsonBytes, []byte(p.getConfiguration().EncryptionSecret))
	if err != nil {
		return nil, fmt.Errorf("error occurred while encrypting the file. Error: %w", err)
	}

	attachment = &serializer.MessageAttachment{
		URL:         p.GetPluginURL() + "/file/" + encode(encrypted),
		ContentType: fileInfo.MimeType,
		FileName:    fileInfo.Name,
	}

	return attachment, nil
}

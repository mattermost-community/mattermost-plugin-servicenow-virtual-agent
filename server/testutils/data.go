package testutils

import (
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
)

func GetID() string {
	return "sfmq19kpztg5iy47ebe51hb31w"
}

func GetFile(deleted bool) *model.FileInfo {
	file := &model.FileInfo{
		CreatorId: GetID(),
		ChannelId: GetID(),
		MimeType:  "mockMimeType",
		Name:      "mockName",
	}

	if deleted {
		file.DeleteAt = 4324234
	}

	return file
}

func GetAppError(message string) *model.AppError {
	return &model.AppError{
		Message: message,
	}
}

func GetServiceNowSysID() string {
	return "d5d4f60807861110da0ef4be7c1ed0d6"
}

func GetSerializerUser() *serializer.User {
	return &serializer.User{
		MattermostUserID: GetID(),
		OAuth2Token:      "test-oauthtoken",
		ServiceNowUser: serializer.ServiceNowUser{
			UserID: GetServiceNowSysID(),
		},
	}
}

func GetPostWithAttachments(numOfAttachments int) *model.Post {
	post := &model.Post{
		Id:        GetID(),
		ChannelId: GetID(),
		UserId:    GetID(),
	}

	attachment := &model.SlackAttachment{
		Title: "mockLabel",
	}

	attachments := make([]*model.SlackAttachment, numOfAttachments)
	for i := 0; i < numOfAttachments; i++ {
		attachments[i] = attachment
	}

	model.ParseSlackAttachment(post, attachments)
	return post
}

func GetPickerBodyWithCarouselOptions(numOfOptions int) *serializer.Picker {
	body := &serializer.Picker{
		UIType:   constants.PickerUIType,
		ItemType: constants.ItemTypePicture,
		Style:    constants.StyleCarousel,
	}

	options := make([]serializer.Option, numOfOptions)
	for i := 0; i < numOfOptions; i++ {
		options[i] = serializer.Option{
			Label:       "mockLabel",
			Value:       "mockValue",
			Description: "mockDescription",
			Attachment:  "mockURL",
		}
	}

	body.Options = options
	return body
}

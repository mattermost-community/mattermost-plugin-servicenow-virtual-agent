package testutils

import "github.com/mattermost/mattermost/server/public/model"

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

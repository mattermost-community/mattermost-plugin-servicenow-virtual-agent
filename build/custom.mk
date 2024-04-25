# Include custom targets and environment variables here

## Generates mock golang interfaces for testing
.PHONY: mock
mock:
ifneq ($(HAS_SERVER),)
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -destination server/mocks/mock_store.go github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/plugin Store
	mockgen -destination server/mocks/mock_client.go github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/plugin Client
endif

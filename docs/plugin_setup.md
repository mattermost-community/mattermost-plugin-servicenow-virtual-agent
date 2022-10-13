# Configuration

- Go to the ServiceNow Virtual Agent plugin configuration page on Mattermost as **System Console > Plugins > ServiceNow Virtual Agent**.
- On the ServiceNow Virtual Agent plugin configuration page, you need to configure the following:
  - **ServiceNow URL**: Enter the URL of your ServiceNow instance.
  - **ServiceNow OAuth Client ID**: The clientID of your registered OAuth app on ServiceNow.
  - **ServiceNow OAuth Client Secret**: The client secret of your registered OAuth app on ServiceNow.
  - **Encryption Secret**: Regenerate a new encryption secret. This encryption secret will be used to encrypt and decrypt the OAuth token.
  - **ServiceNow Webhook Secret**: Regenerate a new webhook secret
    **Note:** Ensure that the webhook secret is configured in the outbound REST endpoint URL(URL where the Virtual Agent sends its responses) of ServiceNow so that the plugin can authenticate API calls from ServiceNow Virtual Agent.

**NOTE:** Please make sure that `Enable users to open Direct Message channels with` setting in **System Console > Site Configuration > Users and Teams** is set to `Any user on the Mattermost server` otherwise you will not be able to start a conversation with the Virtual Agent.

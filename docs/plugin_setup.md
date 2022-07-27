# Configuration

- Go to the ServiceNow Virtual Agent plugin configuration page on Mattermost as **System Console > Plugins > ServiceNow Virtual Agent**.
- On the ServiceNow Virtual Agent plugin configuration page, you need to configure the following:
  - **ServiceNow URL**: Enter the URL of your ServiceNow instance.
  - **ServiceNow OAuth Client ID**: The clientID of your registered OAuth app in servicenow.
  - **ServiceNow OAuth Client Secret**: The client secret of your registered OAuth app in servicenow.
  - **Encryption Secret**: Regenerate a new encryption secret.
  - **ServiceNow Webhook Secret**: Regenerate a new webhook secret
    **Note:** Ensure that the kwebhook secret is configured in the outbound URL(URL where the Virtual Agent send its responses) of serviceNow so that the plugin can authenticate API responses from ServiceNow Virtual Agent.
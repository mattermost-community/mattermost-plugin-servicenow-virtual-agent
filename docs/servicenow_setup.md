# Setting up your ServiceNow instance

We need to set up the virtual agent on a ServiceNow instance, to which we will send our API requests.

## 1. Create your ServiceNow instance 
  - Log in to [ServiceNow](https://developer.servicenow.com).
  - Then click on Create Instance in the top right corner. Basically, ServiceNow itself provides developer instances to anyone who wishes to develop on ServiceNow.
  - Once the instance is created, open the menu from the top right corner, navigate to `Manage Instance Password`, and log in to your dev instance in a new tab.

## 2. Install Glide Virtual Agent and Virtual Agent
  - Navigate to **All > System Applications > All Available Application > All** and install the Glide Virtual Agent and Virtual Agent API.

## 3. Configuring the Virtual Agent API

  - Navigate to **System Web Services > Outbound > Rest Message**
  - Go to **VA Bot to Bot > postMessage**.
  - Update the endpoint -> `https://<your-mattermost-url>/plugins/mattermost-plugin-servicenow-virtual-agent/api/v1/nowbot/processResponse?secret=<your-webhook-secret>`.
  Note: (Webhook secret can be generated from the Mattermost system console settings of virtual agent plugin.)
 
## 4. Creating an OAuth app in ServiceNow
  - Navigate to **All > System OAuth > Application Registry.**
  - Create on the New button on the top right corner and then go to "Create an OAuth API endpoint for external clients".
  - Enter the name for your app and set the redirect URL to `https://<your-mattermost-url>/plugins/mattermost-plugin-servicenow-virtual-agent/api/v1/oauth2/complete`.
  - The client secret will be generated automatically.
  - You will need the values `ClientID` and `ClientSecret` while configuring the plugin.

## 5. Setting up trusted domains for file uploads
  - Navigate to **sys_cs_provider_application.list > VA Bot to Bot Provider Application**.
  - Add your Mattermost URL in the "Trusted media domain" field.
  **Note-** The Provider Channel Identity form may not show the Trusted media domains field. Click on the menu icon in the top left and select "Configure -> Form Layout" to make it visible before adding trusted domains.

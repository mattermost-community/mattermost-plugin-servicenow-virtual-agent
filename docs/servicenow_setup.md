# Setting up your ServiceNow instance

We need to set up the virtual agent on a ServiceNow instance, to which we will send our API requests.

## 1. Get your ServiceNow developer instance (only for developers)
  - Log in to [ServiceNow](https://developer.servicenow.com) developer account.
  - Then click on Request Instance in the top right corner. Basically, ServiceNow itself provides developer instances to anyone who wishes to develop on ServiceNow.
  - Once the instance is created, open the menu from the top right corner, navigate to `Manage Instance Password`, and log in to your dev instance in a new tab.

  **Note-** Mattermost user should have the same email address which will be used to login on to the ServiceNow instance. On the ServiceNow instance, the email address of the user can be updated by going to the user's "Profile" from the top right corner. You can also create a new user on the instance. (Refer [this](https://docs.servicenow.com/en-US/bundle/tokyo-platform-administration/page/administer/users-and-groups/task/t_CreateAUser.html) for creating a new user)

## 2. Install Glide Virtual Agent and Virtual Agent
  - Navigate to **All > System Applications > All Available Application > All** and install the Glide Virtual Agent and Virtual Agent API.

## 3. Configuring the Virtual Agent API

  - Navigate to **System Web Services > Outbound > Rest Message**
  - Go to **VA Bot to Bot > postMessage**.
  - Update the endpoint:
    ```
    https://<your-mattermost-url>/plugins/mattermost-plugin-servicenow-virtual-agent/api/v1/nowbot/processResponse?secret=<your-webhook-secret>
    ```
    **Note**: (Webhook secret can be generated from the Mattermost system console settings of the Virtual agent plugin.)
  - Adding "User-Agent" header: ([Reason](https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB0720934))
  
    Add the "User-Agent" header as shown in the screenshot below with the value "ServiceNow"
    ![image](https://user-images.githubusercontent.com/55234496/201832569-9f11f919-b7c9-4192-a9cf-89a955da08c1.png)


    **Note-** If the "insert a new row..." (as shown in the screenshot below) option is not visible to you in the "HTTP Request" section:

    ![image](https://user-images.githubusercontent.com/55234496/201840807-f593a0cf-aa7a-4f34-bf29-4956f8b680e3.png)

      1. Change the application scope to "Virtual Agent API". (From the top right corner).

          ![image](https://user-images.githubusercontent.com/55234496/201833135-7907cdbc-5e00-4338-b81d-c48204eae614.png)

      2. Make sure that "Enable list edit" and "Double click to edit" options are checked in the "HTTP headers setting" and click "OK".

          ![image](https://user-images.githubusercontent.com/55234496/201832801-3883b457-93af-4d39-8ade-62545913dd2c.png)
            
          ![image](https://user-images.githubusercontent.com/55234496/201832780-40fcb982-aa20-4e81-81e0-e1a4e33160c5.png)

      3. Navigate to **System Web Services > Outbound > Rest Message > VA Bot to Bot > postMessage** again. (Just refreshing the page might not work, so make sure that you navigate again)

## 4. Creating an OAuth app in ServiceNow
  - Navigate to **All > System OAuth > Application Registry.**
  - Click on the New button in the top right corner and then go to "Create an OAuth API endpoint for external clients".
  - Enter the name for your app and set the redirect URL to `https://<your-mattermost-url>/plugins/mattermost-plugin-servicenow-virtual-agent/api/v1/oauth2/complete`.
  - The client secret will be generated automatically.
  - You will need the values `ClientID` and `ClientSecret` while configuring the plugin.

## 5. Setting up trusted domains for file uploads
  - Navigate to **sys_cs_provider_application.list > VA Bot to Bot Provider Application**.
    **Note:** For navigating to `sys_cs_provider_application.list` type "sys_cs_provider_application.list" in **All > Filter** and hit enter.
  - Add your Mattermost URL in the "Trusted media domains" field. Example: mattermost.example.com. **(Do not use the prefix http OR https)**
  **Note-** The Provider Channel Identity form may not show the "Trusted media domains" field. Click on the menu icon in the top left and select "Configure -> Form Layout" to make it visible before adding trusted domains.

## 6. Setting up a high value for "va.bot.to.bot.take.control_times": ([Reason](https://www.servicenow.com/community/virtual-agent-nlu-forum/getting-improper-response-from-virtual-agent-bot-integration-api/m-p/255032))
  - Navigate to "sys_properties.list". (Type "sys_properties.list" in **All > Filter** and hit enter)
  - Make sure that "va.bot.to.bot.take.control_times" is not present.
  
    ![image](https://user-images.githubusercontent.com/55234496/201834695-67077de3-ec76-4665-884b-55167cffa67e.png)

  - Then click on new in the top right corner.
  - Add a high value for this field as shown in the screenshot below and click "Submit".
    ![image](https://user-images.githubusercontent.com/55234496/201836342-2495f201-96e6-443e-97eb-a95dcd4ec09d.png)


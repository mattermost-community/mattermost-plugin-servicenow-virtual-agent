# Mattermost ServiceNow Virtual Agent Plugin

## Table of Contents
- [License](#license)
- [Overview](#overview)
- [Features](#features)
- [Basic Knowledge](#basic-knowledge)
- [Installation](#installation)
- [Setup](#setup)
- [Connecting to ServiceNow](#connecting-to-servicenow)
- [Development](#development)

## License

See the [LICENSE](./LICENSE) file for license rights and limitations.

## Overview

This plugin integrates the ServiceNow Virtual Agent in Mattermost. It is created using the official Virtual Agent Bot API documentation which can be found [here](https://docs.servicenow.com/bundle/sandiego-application-development/page/integrate/inbound-rest/concept/bot-api.html). For a stable production release, please download the latest version from the Plugin Marketplace and follow the instructions to [install](#installation) and [configure](#setup) the plugin.

## Features

- ### This plugin supports sending the below fields to the Virtual Agent through Mattermost:
  1. **Text messages**

        ![image](https://user-images.githubusercontent.com/55234496/196630251-c4332607-9181-483d-a55e-e5805ef36007.png)

  2. **File attachments**  

        ![image](https://user-images.githubusercontent.com/55234496/196138330-711b97da-e7f1-42d4-91d5-4e6f5c0dfcbd.png)

  3. **Date/Time**

        ![image](https://user-images.githubusercontent.com/55234496/196132228-03649985-4d30-423c-acd1-a5af894b25f7.png)

        ![image](https://user-images.githubusercontent.com/55234496/196132775-24ca6bb5-34bb-42fe-bdaf-e5661a46813a.png)

- ### Handling/Displaying the following types of responses from the Virtual Agent:

  1. **OutputText**

        ![image](https://user-images.githubusercontent.com/55234496/196124240-db7f8ed1-fe2d-457b-89c5-df9c28d09879.png)

  2. **OutputImage**

        ![image](https://user-images.githubusercontent.com/55234496/196133695-dca8e495-d37f-4c61-b882-63db197a7c99.png)

  3. **OutputLink**

        ![image](https://user-images.githubusercontent.com/55234496/196124712-19a1d2bd-b1cf-4018-95b6-eb5ef9920c1c.png)

  4. **Picker/Dropdown**

        ![image](https://user-images.githubusercontent.com/55234496/196125669-1e3f2461-d2f3-4028-9320-6cab71ecd27e.png)

  5. **OutputCard**

        ![image](https://user-images.githubusercontent.com/55234496/196125018-b4e0ecbd-4f2a-4e6d-9dc4-e3a08704d7cc.png)

**Note-** For sending file attachments to the Live Agent other than an image, you need to have ServiceNow version greater than or equal to "San Diego Patch 4". Also, the link of the file attachment sent to the Virtual Agent/Live Agent will be expired in 15 minutes.

## Basic Knowledge

- [What is ServiceNow?](https://www.servicenow.com/)
- [What is ServiceNow Virtual Agent?](https://www.servicenow.com/products/virtual-agent.html)
    - [Virtual Agent](https://docs.servicenow.com/bundle/paris-now-intelligence/page/administer/virtual-agent/concept/virtual-agent-overview.html)
    - [Activating Virtual Agent](https://docs.servicenow.com/bundle/sandiego-servicenow-platform/page/administer/virtual-agent/task/activate-virtual-agent.html)
    
- [Virtual Agent Bot Integration API](https://docs.servicenow.com/bundle/sandiego-application-development/page/integrate/inbound-rest/concept/bot-api.html)

- **Virtual Agent Designer**

    To start a conversation with a Virtual agent we need to select a conversation topic. Virtual Agent provides various predefined conversation flows/topics for some common conversations and we can design our own conversation flows as well as mentioned [here](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/reference/conversation-designer-virtual-agent.html).
    - [Virtual Agent Designer](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/reference/conversation-designer-virtual-agent.html)
    - [Designing a Virtual Agent Topic](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/concept/design-va-topic.html)

- **Pre-defined Conversation Flows/Topics**

    - [Pre-defined Conversation Flows/Topics](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/reference/prebuilt-topics-ITSM.html)
    - [Additional Plugins](https://docs.servicenow.com/bundle/sandiego-servicenow-platform/page/administer/virtual-agent/reference/additional-va-plugins.html)
    - Sometimes a conversation flow might not work because the user does not have access to some tables or APIs which are being used in that flow (You can see the errors in "All > System Log > Errors"). In such cases, you have to manually provide the access in "All > Application Cross-Scope Access".
      - [Create cross-scope access privileges for topic blocks and custom controls](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/task/configure-cross-scope-privileges.html)

- **File Upload**

    For sending file attachments to the Virtual Agent or the Live Agent we need to send a public link of the file in the request from where the Virtual Agent can download it. Virtual Agent does not support sending file attachments from any source, we have to specify the trusted domains ourselves so that the Virtual Agent knows that the specified domains can be trusted.
    - [Set up trusted media domains](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/task/ccif-secure-file-upload.html)
      - [Configuring the form layout](https://docs.servicenow.com/en-US/bundle/sandiego-platform-administration/page/administer/form-administration/concept/configure-form-layout.html)

- **Invalid response**
    We get an "undefined" response from Virtual Agent API in a specific case. Please refer to [this](https://www.servicenow.com/community/virtual-agent-nlu-forum/getting-improper-response-from-virtual-agent-bot-integration-api/m-p/255032) community question for more information about the issue. To bypass this issue, you should increase the value of `va.bot.to.bot.take.control_times` variable. You can read more about it [here](https://docs.servicenow.com/bundle/sandiego-servicenow-platform/page/administer/virtual-agent/concept/bot2bot.html).

## Installation

1. Go to the [releases page of this GitHub repository](https://github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/releases) and download the latest release for your Mattermost server.
2. Upload this file on the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Enable the plugin from **System Console > Plugins > ServiceNow Virtual Agent**.

## Setup

  - [ServiceNow Setup](./servicenow_setup.md)
  - [Plugin Setup](./plugin_setup.md)

## Connecting to ServiceNow
  - Send any direct message to `servicenow-virtual-agent`.
  **Note-** If `servicenow-virtual-agent` is not visible in your DMs, click on the plus(+) icon on the right side of "Direct Messages" and search for `servicenow-virtual-agent`.

    ![image](https://user-images.githubusercontent.com/55234496/203485550-f8d4a3c3-6667-4526-8993-48b54923a277.png)

  - You will get a response with a link to connect your ServiceNow account.

    ![image](https://user-images.githubusercontent.com/55234496/181167065-f1b93e3b-8963-484a-8dda-a980173191a0.png)

  - Click on that link. If it asks for login, enter your instance credentials and click `Allow` to connect your account.
    
## Development

This plugin contains server portion only. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/integrate/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/integrate/plugins/developer-setup/) for more information about developing and extending plugins.

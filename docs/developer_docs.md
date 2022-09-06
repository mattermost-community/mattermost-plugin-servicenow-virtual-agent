# Mattermost ServiceNow Virtual Agent Plugin

## Table of Contents
- [License](#license)
- [Overview](#overview)
- [Features](#features)
- [Basic Knowkedge](#basic-knowledge)
- [Installation](#installation)
- [Setup](#setup)
- [Connecting to ServiceNow](#connecting-to-servicenow)
- [Development](#development)

## License

See the [LICENSE](./LICENSE) file for license rights and limitations.

## Overview

This plugin integrates the ServiceNow Virtual Agent in Mattermost. It is created using the official Virtual Agent Bot API documentation which can be found [here](https://docs.servicenow.com/bundle/sandiego-application-development/page/integrate/inbound-rest/concept/bot-api.html). For a stable production rlease, please download the latest version from the Plugin Marketplace and follow the instructions to [install](#installation) and [configure](#setup) the plugin.

## Features

This plugin supports sending text messages to the Virtual Agent through Mattermost and handling/displaying different types of responses from the Virtual Agent.
**Note-** Currently we only support sending text messages and displaying text, picker/dropdown & link responses from the Virtual Agent API.
**Note-** For sending file attachments to the Live Agent other than an image, you need to have ServiceNow version >= "San Diego Patch 4". Also, the link of the file attachment sent to the Virtual Agent/Live Agent will be expired in 15 minutes.

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

- **File Upload**

    For sending file attachments to the Virtual Agent or the Live Agent we need to send a public link of the file in the request from where the Virtual Agent can download the file. Virtual Agent does not support sending file attachments from any source, we have to specify the trusted domains ourselves so that the Virtual Agent knows that these domains can be trusted.
    - [Set up trusted media domains](https://docs.servicenow.com/bundle/quebec-now-intelligence/page/administer/virtual-agent/task/ccif-secure-file-upload.html)
      - [Configuring the form layout](https://docs.servicenow.com/en-US/bundle/sandiego-platform-administration/page/administer/form-administration/concept/configure-form-layout.html)

## Installation

1. Go to the [releases page of this GitHub repository](https://github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/releases) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Enable the plugin from **System Console > Plugins > ServiceNow Virtual Agent**.

## Setup

  - [ServiceNow Setup](./servicenow_setup.md)
  - [Plugin Setup](./plugin_setup.md)

## Connecting to ServiceNow
  - Send any direct message to `servicenow-virtual-agent`.
  - You will get a response with a link to connect your ServiceNow account.
  ![Screenshot from 2022-07-26 14-55-24](https://user-images.githubusercontent.com/55234496/181167065-f1b93e3b-8963-484a-8dda-a980173191a0.png)
  - Click on that link. If it asks for login, enter your instance credentials and click `Allow` to connect your account.
    
## Development

### Setup

Make sure you have the following components installed:  

- Go - v1.16 - [Getting Started](https://golang.org/doc/install)
    > **Note:** If you have installed Go to a custom location, make sure the `$GOROOT` variable is set properly. Refer [Installing to a custom location](https://golang.org/doc/install#install).

- Make

### Building the plugin

Run the following command in the plugin repo to prepare a compiled, distributable plugin zip:

```bash
make dist
```

After a successful build, a `.tar.gz` file in `/dist` folder will be created which can be uploaded to Mattermost. To avoid having to manually install your plugin, deploy your plugin using one of the following options.

### Deploying with Local Mode

If your Mattermost server is running locally, you can enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying your plugin. Edit your server configuration as follows:

```
{
    "ServiceSettings": {
        ...
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
    }
}
```

and then deploy your plugin:

```bash
make deploy
```

You may also customize the Unix socket path:

```bash
export MM_LOCALSOCKETPATH=/var/tmp/alternate_local.socket
make deploy
```

If developing a plugin with a web app, watch for changes and deploy those automatically:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make watch
```

### Deploying with credentials

Alternatively, you can authenticate with the server's API with credentials:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

or with a [personal access token](https://docs.mattermost.com/developer/personal-access-tokens.html):

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=j44acwd8obn78cdcx7koid4jkr
make deploy
```

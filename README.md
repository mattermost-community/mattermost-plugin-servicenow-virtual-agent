# Mattermost ServiceNow Virtual Agent Plugin
## Table of Contents
- [License](#license)
- [Overview](#overview)
- [Features](#features)
- [Installation](#installation)
- [Setup](#setup)
- [Connecting to ServiceNow](#connecting-to-servicenow)

## License

See the [LICENSE](./LICENSE) file for license rights and limitations.

## Overview

This plugin integrates the ServiceNow Virtual Agent in Mattermost. For a stable production release, please download the latest version from the Plugin Marketplace and follow the instructions to [install](#installation) and [configure](#setup) the plugin. If you are a developer and want to work on this plugin, please switch to the [Developer docs](./docs/developer_docs.md).

## Features

This plugin supports sending text messages to the Virtual Agent through Mattermost and handling/displaying different types of responses from the Virtual Agent.
**Note-** Currently we only support sending text messages & file attachments and displaying text, picker/dropdown & link responses from the Virtual Agent API.
**Note-** For sending file attachments to the Live Agent other than an image, you need to have ServiceNow version >= "San Diego Patch 4". Also, the link of the file attachment sent to the Virtual Agent/Live Agent will be expired in 15 minutes.

## Installation

1. Go to the [releases page of this GitHub repository](https://github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/releases) and download the latest release for your Mattermost server.
2. Upload this file on the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Enable the plugin from **System Console > Plugins > ServiceNow Virtual Agent**.

## Setup

  - [ServiceNow Setup](./docs/servicenow_setup.md)
  - [Plugin Setup](./docs/plugin_setup.md)

## Connecting to ServiceNow
  - Send any direct message to the `ServiceNow Virtual Agent` bot with the username `servicenow-virtual-agent`.
  - You will get a response with a link to connect your ServiceNow account if you haven't already connected.

    ![Screenshot from 2022-07-26 14-55-24](https://user-images.githubusercontent.com/55234496/181167065-f1b93e3b-8963-484a-8dda-a980173191a0.png)
  
  - Click on that link. If it asks for login, enter your ServiceNow credentials and click `Allow` to connect your account.

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

- ### This plugin supports sending the below fields to the Virtual Agent through Mattermost-
  1. **Text messages**

        ![image](https://user-images.githubusercontent.com/55234496/196138915-9577372f-89a4-49f7-bc02-398ca42a7003.png)

  2. **File attachments**  

        ![Screenshot from 2022-10-17 14-41-41](https://user-images.githubusercontent.com/55234496/196138330-711b97da-e7f1-42d4-91d5-4e6f5c0dfcbd.png)

  3. **The selected dates**

        ![Screenshot from 2022-10-17 14-14-50](https://user-images.githubusercontent.com/55234496/196132228-03649985-4d30-423c-acd1-a5af894b25f7.png)

        ![Screenshot from 2022-10-17 14-17-28](https://user-images.githubusercontent.com/55234496/196132775-24ca6bb5-34bb-42fe-bdaf-e5661a46813a.png)

- ### Handling/Displaying the following types of responses from the Virtual Agent

  1. **OutputText**

        ![Screenshot from 2022-10-17 13-40-59](https://user-images.githubusercontent.com/55234496/196124240-db7f8ed1-fe2d-457b-89c5-df9c28d09879.png)

  2. **OutputImage**

        ![Screenshot from 2022-10-17 14-20-26](https://user-images.githubusercontent.com/55234496/196133695-dca8e495-d37f-4c61-b882-63db197a7c99.png)

  3. **OutputLink**

        ![Screenshot from 2022-10-17 13-45-11](https://user-images.githubusercontent.com/55234496/196124712-19a1d2bd-b1cf-4018-95b6-eb5ef9920c1c.png)

  4. **Picker/Dropdown**

        ![Screenshot from 2022-10-17 13-49-35](https://user-images.githubusercontent.com/55234496/196125669-1e3f2461-d2f3-4028-9320-6cab71ecd27e.png)

  5. **OutputCard**

        ![Screenshot from 2022-10-17 13-46-51](https://user-images.githubusercontent.com/55234496/196125018-b4e0ecbd-4f2a-4e6d-9dc4-e3a08704d7cc.png)

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

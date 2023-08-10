[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![GPL License][license-shield]][license-url]

[![Readme in English](https://img.shields.io/badge/Readme-English-blue)](README.md)
[![Readme in Turkish](https://img.shields.io/badge/Readme-Turkish-red)](README.tr.md)

<div align="center"> 
<a href="https://mono.net.tr/">
  <img src="https://monobilisim.com.tr/images/mono-bilisim.svg" width="340"/>
</a>

<h2 align="center">whatsapp-ws</h2>
<b>whatsapp-ws</b> utilizes the whatsmeow library to create WebSocket interfaces for integrating WhatsApp messaging capabilities and performing actions programmatically

</div>

---

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Introduction](#introduction)
- [API Usage](#api-usage)
  - [/ws Endpoint](#ws-endpoint)
  - [/status Endpoint](#status-endpoint)
  - [/qr Endpoint](#qr-endpoint)
  - [/upload Endpoint](#upload-endpoint)
- [Build](#build)
- [Endpoints](#endpoints)
- [License](#license)

---
## Introduction

whatsapp-ws is a project built upon the whatsmeow library, providing a WebSocket interface and endpoints to send daily messages to a database and serve as a WhatsApp bridge. It allows users to connect via WebSocket to interact with WhatsApp through programmatically sent commands.

The main objective of whatsapp-ws is to streamline the integration of WhatsApp messaging capabilities into various applications and systems. By utilizing the WebSocket interface, users can establish real-time connections and send commands to interact with WhatsApp in an automated fashion.

---

## API Usage

### /ws Endpoint

The `/ws` endpoint provides a WebSocket interface for real-time interaction with the WhatsApp messaging capabilities offered by whatsapp-ws. Users can connect to this endpoint and send commands in the form of JSON objects.

```json
{
  "cmd": "string",
  "args": ["string"],
  "user_id": int
}
```

- `cmd`: The command to be executed.
- `args`: An array of string arguments required for the command.
- `user_id`: An integer representing the user ID for context.

### /status Endpoint

The `/status` endpoint allows users to check if they are logged in. It returns an HTTP 200 response if the user is logged in and authenticated.

### /qr Endpoint

The `/qr` endpoint serves the login QR code for WhatsApp. Users can access this endpoint to view the QR code required for logging in to WhatsApp.

### /upload Endpoint

The `/upload` endpoint enables users to upload files to WhatsApp. It can be used with tools like `curl`. Here's an example command to upload a file:
```sh
curl -X POST -F file=@filepath -F jid=PHONE_NUMBER@s.whatsapp.net -F user_id=1 http://localhost:6023/upload
```

---

## Build

To build whatsapp-ws, use the following command:

```bash
go build -ldflags '-extldflags "-static"'
```

---

## Endpoints

- `/ws` - websocket endpoint
- `/status` - status endpoint
- `/qr` - qr endpoint
- `/upload` - upload endpoint

---

## License

whatsapp-ws is GPL-3.0 licensed. See [LICENSE](LICENSE) file for details.

[contributors-shield]: https://img.shields.io/github/contributors/monobilisim/whatsapp-ws.svg?style=for-the-badge
[contributors-url]: https://github.com/monobilisim/whatsapp-ws/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/monobilisim/whatsapp-ws.svg?style=for-the-badge
[forks-url]: https://github.com/monobilisim/whatsapp-ws/network/members
[stars-shield]: https://img.shields.io/github/stars/monobilisim/whatsapp-ws.svg?style=for-the-badge
[stars-url]: https://github.com/monobilisim/whatsapp-ws/stargazers
[issues-shield]: https://img.shields.io/github/issues/monobilisim/whatsapp-ws.svg?style=for-the-badge
[issues-url]: https://github.com/monobilisim/whatsapp-ws/issues
[license-shield]: https://img.shields.io/github/license/monobilisim/whatsapp-ws.svg?style=for-the-badge
[license-url]: https://github.com/monobilisim/whatsapp-ws/blob/master/LICENSE.txt
# 📧 MailChannels SMTP API

This project implements an API compatible with MailChannels, but uses SMTP for sending emails. It's built with Go and provides a simple way to send emails through your SMTP server.

## 🚀 Features

- 🔄 Compatible with MailChannels API
- 📨 Sends emails via SMTP
- 🔒 Supports SSL/TLS encryption
- 🧪 Dry-run mode for testing
- 🎛️ Configurable via environment variables

## 🛠️ Installation

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/mailchannels-smtp-api.git
   cd mailchannels-smtp-api
   ```

2. Install dependencies:

   ```
   make deps
   ```

3. Build the project:
   ```
   make build
   ```

## ⚙️ Configuration

Set the following environment variables:

- `SMTP_HOST`: Your SMTP server host
- `SMTP_USER`: Your SMTP username
- `SMTP_PASSWORD`: Your SMTP password
- `SMTP_PORT`: Your SMTP server port
- `SMTP_ENCRYPT`: Encryption type (SSL, TLS, or PLAIN)

You can set these in your environment or in a `.env` file in the project root.

## 🏃‍♂️ Running the API

To start the API server:

```
./mailchannels
```

By default, the server runs on port 8080. You can change this by setting the `PORT` environment variable.

## 🧪 Testing

To run tests:

```
make test
```

## 🐳 Docker

To build and run with Docker:

```
docker build -t mailchannels-smtp-api .
docker run -p 8080:8080 --env-file .env mailchannels-smtp-api
```

## 📚 API Documentation

The API is compatible with the MailChannels API. Send POST requests to `/tx/v1/send` with your email data.

For a dry run, add `?dry-run=true` to the URL.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

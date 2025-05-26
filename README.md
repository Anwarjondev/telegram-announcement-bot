# Telegram Announcement Bot

A Telegram bot that allows you to share announcements across multiple channels. The bot includes an admin panel for managing channels and viewing announcements.

## Features

- Add the bot to multiple Telegram channels
- Share announcements across all connected channels
- Web-based admin panel for managing channels
- View announcement history and status
- Secure admin-only access

## Prerequisites

- Go 1.21 or higher
- A Telegram Bot Token (get it from [@BotFather](https://t.me/botfather))
- SQLite (included in the project)

## Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/telegram-announcement-bot.git
cd telegram-announcement-bot
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export ADMIN_USERNAME="your_telegram_username"
export WEB_PORT="8080"
```

4. Build and run the bot:
```bash
go run cmd/main.go
```

## Usage

1. Add the bot to your Telegram channels as an administrator with permission to post messages.
2. **Obtain Channel IDs:** To add channels to the admin panel, you'll need their numerical chat IDs. The easiest way to get a channel's ID is to add a bot like [@raw_data_bot](https://t.me/raw_data_bot) to the channel and send any message. The bot will respond with the chat ID (it will be a negative number, usually starting with `-100`).
3. Access the admin panel at `http://localhost:8080`
4. Add your channels through the admin panel using the obtained numerical chat IDs and desired names.
5. Post announcements in any connected channel.
6. The bot will automatically share the announcement to all other connected channels.

## Admin Panel

The admin panel provides the following features:

- View and manage connected channels
- Add new channels
- Remove channels
- View announcement history
- Monitor announcement status

## Security

- Only the specified admin username can post announcements
- The bot must be an administrator in all channels
- Web interface is protected by admin-only access

## Contributing

Feel free to submit issues and enhancement requests! 
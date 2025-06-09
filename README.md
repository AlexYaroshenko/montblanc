# Mont Blanc Refuge Availability Checker

A command-line tool to monitor availability for refuges on Mont Blanc route. The tool checks availability for both Tête Rousse and du Goûter refuges and sends notifications via Telegram when changes are detected.

## Features

- Checks availability for both Tête Rousse and du Goûter refuges
- Monitors availability continuously with configurable check frequency
- Sends notifications via Telegram when changes are detected
- Supports multiple Telegram chat subscribers
- Shows availability status in the console
- Handles session expiration gracefully

## Prerequisites

- Go 1.16 or later
- A Telegram bot token (get it from [@BotFather](https://t.me/botfather))
- Your Telegram chat ID (get it from [@userinfobot](https://t.me/userinfobot))

## Installation

1. Clone the repository:
```bash
git clone https://github.com/AlexYaroshenko/montblanc.git
cd montblanc
```

2. Build the program:
```bash
go build -o check-booking cmd/check/main.go
```

## Usage

Basic usage:
```bash
./check-booking -date YYYY-MM-DD
```

This will:
- Check availability for the entire month of the given date
- Use the default Telegram bot token and chat IDs from environment variables
- Send notifications to all configured chat IDs
- Check availability every minute (default frequency)

Advanced usage:
```bash
./check-booking -date YYYY-MM-DD -pax NUMBER -chat-ids "ID1,ID2,..." -frequency MINUTES
```

Parameters:
- `-date`: Required. The date in YYYY-MM-DD format (will check the entire month)
- `-pax`: Optional. Number of people (default: 1)
- `-chat-ids`: Optional. Comma-separated list of Telegram chat IDs
- `-frequency`: Optional. Check frequency in minutes (default: 1)

Environment variables:
- `TELEGRAM_BOT_TOKEN`: Telegram bot token
- `TELEGRAM_CHAT_IDS`: Comma-separated list of Telegram chat IDs

## Setting up Telegram Notifications

1. Create a new bot:
   - Open Telegram
   - Search for "@BotFather"
   - Send `/newbot` command
   - Follow instructions to create your bot
   - Save the bot token

2. Get your chat ID:
   - Open Telegram
   - Search for "@userinfobot"
   - Send any message
   - The bot will reply with your chat ID

3. Start a chat with your bot:
   - Open Telegram
   - Search for your bot using its username
   - Start a chat by sending any message

4. Set up environment variables:
```bash
export TELEGRAM_BOT_TOKEN="your_bot_token"
export TELEGRAM_CHAT_IDS="your_chat_id,another_chat_id"
```

## Notes

- The program checks availability at the specified frequency (default: every minute)
- Notifications are sent when availability changes
- The program will notify you if the session expires
- You can add multiple chat IDs to receive notifications
- The program sends notifications for startup, shutdown, and errors

## License

MIT License 
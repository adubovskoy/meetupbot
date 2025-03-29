# Meetup registration Bot

A Telegram bot for managing event registrations and attendance tracking. This bot allows users to register for events, check their registration status, and provides administrators with tools to manage events.

## Features

- User registration for events
- Registration status checking
- Attendance tracking via QR codes
- Email collection from participants (not ready yet)
- Event capacity management
- Multiple event support with automatic archiving

## Prerequisites

- Go 1.24 or higher
- SQLite3
- Telegram Bot Token (obtained from [@BotFather](https://t.me/BotFather))

## Installation

1. Clone the repository:
   ```
   git clone <repository-url>
   cd rndphpbot
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set the required environment variables:
   ```
   # Required: Telegram Bot Token
   export BOT_TOKEN=your_telegram_bot_token
   
   # Optional: Admin users (comma-separated list of Telegram usernames)
   export ADMIN_USERS=admin1,admin2,admin3
   ```

4. Build and run the application:
   ```
   go build
   ./meetupbot
   ```

## Database Structure

The bot uses SQLite3 with two main tables:

- **users**: Stores user registration information
- **events**: Stores event details including capacity and registration count

## Available Commands

### User Commands

- `/start` - Welcome message and registration option
- `/register` - Register for the current active event
- `/state` - Check registration status and available spots
- `/addemail your_email@example.com` - Add or update your email address

### Admin Commands

- `/addevent EventName;YYYY-MM-DD;Capacity` - Create a new event (automatically marks previous events as past)
- `/qrcode` - Generate a QR code for event check-in

## QR Code Check-in

The bot supports a QR code-based check-in system:

1. Administrators generate a QR code for an event using `/qrcode`
2. The QR code is displayed at the event entrance
3. Attendees scan the QR code, which opens a Telegram deep link with the command `/start imhere`
4. When users click this link, their attendance is recorded in the system

## Dependencies

- [github.com/go-telegram-bot-api/telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram Bot API wrapper
- [github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite3 driver for Go
- [github.com/skip2/go-qrcode](https://github.com/skip2/go-qrcode) - QR code generation library

## License

MIT

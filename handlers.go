package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/skip2/go-qrcode"
)

// handleCommand routes the command to the corresponding handler.
func handleCommand(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		sendMessage(bot, msg.Chat.ID, "Добро пожаловать!")
	case "register":
		handleRegister(bot, db, msg)
	case "addevent":
		handleAddEvent(bot, db, msg)
	case "qrcode":
		handleQRCode(bot, db, msg)
	case "addemail":
		handleAddEmail(bot, db, msg)
	default:
		sendMessage(bot, msg.Chat.ID, "Unknown command")
	}
}

// sendMessage sends a text message to the given chat ID.
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	message := tgbotapi.NewMessage(chatID, text)
	bot.Send(message)
}

// handleRegister sends a message with an inline "Register" button.
func handleRegister(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	button := tgbotapi.NewInlineKeyboardButtonData("Register", "register")
	row := tgbotapi.NewInlineKeyboardRow(button)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	message := tgbotapi.NewMessage(msg.Chat.ID, "Press the button below to register.")
	message.ReplyMarkup = keyboard
	bot.Send(message)
}

// handleNoDialog processes any non-command message in "no dialog" mode.
func handleNoDialog(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	event, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Error retrieving event info")
		return
	}
	if event == nil || event.registrationCount >= event.capacity {
		sendMessage(bot, msg.Chat.ID, "Registration is closed")
		return
	}

	registered, _, err := isUserRegistered(db, msg.From.ID, event.id)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Error checking registration")
		return
	}

	var button tgbotapi.InlineKeyboardButton
	if registered {
		button = tgbotapi.NewInlineKeyboardButtonData("Changed my mind, remove me", "remove")
	} else {
		button = tgbotapi.NewInlineKeyboardButtonData("Register", "register")
	}
	row := tgbotapi.NewInlineKeyboardRow(button)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	message := tgbotapi.NewMessage(msg.Chat.ID, "Active event found. Please choose an option:")
	message.ReplyMarkup = keyboard
	bot.Send(message)
}

// handleCallbackQuery processes callback queries from inline buttons.
func handleCallbackQuery(bot *tgbotapi.BotAPI, db *sql.DB, cq *tgbotapi.CallbackQuery) {
	event, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, cq.Message.Chat.ID, "Error retrieving event info")
		return
	}
	if event == nil || event.registrationCount >= event.capacity {
		sendMessage(bot, cq.Message.Chat.ID, "Registration is closed")
		return
	}

	if cq.Data == "register" {
		registered, _, err := isUserRegistered(db, cq.From.ID, event.id)
		if err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Error checking registration")
			return
		}
		if registered {
			remaining := event.capacity - event.registrationCount
			sendMessage(bot, cq.Message.Chat.ID, "You're already registered. Remaining seats: "+strconv.Itoa(remaining))
			return
		}
		reg := UserRegistration{
			TelegramID:       cq.From.ID,
			Username:         cq.From.UserName,
			Name:             cq.From.FirstName + " " + cq.From.LastName,
			RegistrationDate: time.Now(),
			Email:            "",
			EventID:          event.id,
		}
		if err := registerUser(db, reg); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Error during registration")
			return
		}
		if err := updateEventRegistrationCount(db, event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Error updating event registration count")
			return
		}
		callback := tgbotapi.NewCallback(cq.ID, "Registration successful!")
		bot.AnswerCallbackQuery(callback)
	} else if cq.Data == "remove" {
		registered, _, err := isUserRegistered(db, cq.From.ID, event.id)
		if err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Error checking registration")
			return
		}
		if !registered {
			remaining := event.capacity - event.registrationCount
			sendMessage(bot, cq.Message.Chat.ID, "You're not registered. Remaining seats: "+strconv.Itoa(remaining))
			return
		}
		if err := removeRegistration(db, cq.From.ID, event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Error removing registration")
			return
		}
		if err := decrementEventRegistrationCount(db, event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Error updating event registration count")
			return
		}
		callback := tgbotapi.NewCallback(cq.ID, "Registration removed!")
		bot.AnswerCallbackQuery(callback)
	}

	updatedEvent, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, cq.Message.Chat.ID, "Error retrieving updated event info")
		return
	}
	remaining := updatedEvent.capacity - updatedEvent.registrationCount
	sendMessage(bot, cq.Message.Chat.ID, "Remaining seats: "+strconv.Itoa(remaining))
}

// handleAddEmail allows the user to optionally add an email to their registration.
func handleAddEmail(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		sendMessage(bot, msg.Chat.ID, "Please provide your email. Usage: /addemail your_email@example.com")
		return
	}
	email := strings.TrimSpace(args)
	if err := updateUserEmail(db, msg.From.ID, email); err != nil {
		sendMessage(bot, msg.Chat.ID, "Error updating email.")
		return
	}
	sendMessage(bot, msg.Chat.ID, "Email updated successfully!")
}

// handleAddEvent processes the /addevent command.
func handleAddEvent(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	parts := strings.Split(args, ";")
	if len(parts) < 3 {
		sendMessage(bot, msg.Chat.ID, "Usage: /addevent EventName;YYYY-MM-DD;Capacity")
		return
	}
	name := strings.TrimSpace(parts[0])
	dateStr := strings.TrimSpace(parts[1])
	capacityStr := strings.TrimSpace(parts[2])
	capacity, err := strconv.Atoi(capacityStr)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Invalid capacity number")
		return
	}
	eventDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Invalid date format. Use YYYY-MM-DD")
		return
	}
	stmt, err := db.Prepare("INSERT INTO events (name, date, capacity, state) VALUES (?, ?, ?, 'active')")
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Error preparing event insertion")
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(name, eventDate.Format(time.RFC3339), capacity); err != nil {
		sendMessage(bot, msg.Chat.ID, "Error inserting event")
		return
	}
	sendMessage(bot, msg.Chat.ID, "Event added successfully!")
}

// handleQRCode processes the /qrcode command.
func handleQRCode(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		sendMessage(bot, msg.Chat.ID, "Usage: /qrcode event_id")
		return
	}
	eventID, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Invalid event id")
		return
	}
	qrData := "event:" + strconv.Itoa(eventID)
	qrFile := "qrcode_event_" + strconv.Itoa(eventID) + ".png"
	if err := qrcode.WriteFile(qrData, qrcode.Medium, 256, qrFile); err != nil {
		sendMessage(bot, msg.Chat.ID, "Error generating QR code")
		return
	}
	photo := tgbotapi.NewPhotoUpload(msg.Chat.ID, qrFile)
	photo.Caption = "QR Code for event registration"
	bot.Send(photo)
}

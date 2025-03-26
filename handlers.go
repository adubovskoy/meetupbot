package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/skip2/go-qrcode"
)

// handleCommand routes commands to corresponding handlers.
func handleCommand(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	if msg.Command() == "start" && strings.ToLower(msg.CommandArguments()) == "imhere" {
		handleImhere(bot, db, msg)
		return
	}
	switch msg.Command() {
	case "start":
		sendMessage(bot, msg.Chat.ID, "Добро пожаловать! \nИспользуйте /start для регистрации или дерегистрации на митап."+
			"\nИспользуйте /state для получения статуса регистрации.")
		handleNoDialog(bot, db, msg)
	case "register":
		handleRegister(bot, db, msg)
	case "addevent":
		if IsAdmin(msg.From.UserName) {
			handleAddEvent(bot, db, msg)
		} else {
			sendMessage(bot, msg.Chat.ID, "У вас нет прав для выполнения этой команды. Только администраторы могут добавлять события.")
		}
	case "qrcode":
		if IsAdmin(msg.From.UserName) {
			handleQRCode(bot, db, msg)
		} else {
			sendMessage(bot, msg.Chat.ID, "У вас нет прав для выполнения этой команды. Только администраторы могут генерировать QR-коды.")
		}
	case "addemail":
		handleAddEmail(bot, db, msg)
	case "state":
		handleState(bot, db, msg)
	default:
		sendMessage(bot, msg.Chat.ID, "Неизвестная команда")
	}
}

// sendMessage sends a text message to the given chat.
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	message := tgbotapi.NewMessage(chatID, text)
	bot.Send(message)
}

// handleRegister sends the register button.
func handleRegister(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	button := tgbotapi.NewInlineKeyboardButtonData("Зарегистрироваться", "register")
	row := tgbotapi.NewInlineKeyboardRow(button)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	message := tgbotapi.NewMessage(msg.Chat.ID, "Нажмите кнопку ниже, чтобы зарегистрироваться.")
	message.ReplyMarkup = keyboard
	bot.Send(message)
}

// Provide event state
func handleState(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	event, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil {
		sendMessage(bot, msg.Chat.ID, "Нет активного события")
		return
	}
	remaining := event.capacity - event.registrationCount
	sendMessage(bot, msg.Chat.ID, "Осталось мест: "+strconv.Itoa(remaining))
	// Am I registred?
	registered, _, err := isUserRegistered(db, msg.From.ID, event.id)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка проверки регистрации")
		return
	}
	if registered {
		sendMessage(bot, msg.Chat.ID, "Вы зарегистрированы")
	} else {
		sendMessage(bot, msg.Chat.ID, "Вы не зарегистрированы")
	}
}

// handleNoDialog handles all non-command messages.
func handleNoDialog(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	event, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil || event.registrationCount >= event.capacity {
		sendMessage(bot, msg.Chat.ID, "Регистрация закрыта")
		return
	}

	registered, _, err := isUserRegistered(db, msg.From.ID, event.id)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка проверки регистрации")
		return
	}

	activeMeetupDate := event.date.Format("02.01.2006")

	var button tgbotapi.InlineKeyboardButton
	if registered {
		button = tgbotapi.NewInlineKeyboardButtonData("Передумал, удалите меня", "remove")
	} else {
		button = tgbotapi.NewInlineKeyboardButtonData("Зарегистрироваться", "register")
	}
	row := tgbotapi.NewInlineKeyboardRow(button)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	message := tgbotapi.NewMessage(msg.Chat.ID, "Идёте на митап "+activeMeetupDate+"?")
	message.ReplyMarkup = keyboard
	bot.Send(message)
}

// handleImhere handles the "/start imhere" command.
// If the user is registered, it updates visited = 1.
// If not, it creates a new record with visited = 1 and registred = 0.
func handleImhere(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	event, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil {
		sendMessage(bot, msg.Chat.ID, "Нет активного события")
		return
	}
	registered, _, err := isUserRegistered(db, msg.From.ID, event.id)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка проверки регистрации")
		return
	}
	if registered {
		err := updateVisitedStatus(db, msg.From.ID, event.id, 1)
		if err != nil {
			sendMessage(bot, msg.Chat.ID, "Ошибка обновления статуса посещения")
			return
		}
		sendMessage(bot, msg.Chat.ID, "Статус посещения обновлён. Спасибо, что пришли!")
	} else {
		// Add new user with visited = 1 and registred = 0
		newUser := UserRegistration{
			TelegramID:       msg.From.ID,
			Username:         msg.From.UserName,
			Name:             msg.From.FirstName + " " + msg.From.LastName,
			RegistrationDate: time.Now(),
			Email:            "",
			EventID:          event.id,
			Registred:        0,
			Visited:          1,
		}
		err := registerUser(db, newUser)
		if err != nil {
			sendMessage(bot, msg.Chat.ID, "Ошибка добавления пользователя")
			return
		}
		sendMessage(bot, msg.Chat.ID, "Вы добавлены как посетитель, но не зарегистрированы заранее.")
	}
}

// updateRegistration updates the registration row for an active event.
func updateRegistration(db *sql.DB, reg UserRegistration) error {
	stmt, err := db.Prepare("UPDATE users SET username = ?, name = ?, registration_date = ?, email = ?, registred = ? WHERE telegram_id = ? AND event_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(reg.Username, reg.Name, reg.RegistrationDate.Format(time.RFC3339), reg.Email, reg.Registred, reg.TelegramID, reg.EventID)
	return err
}

// handleCallbackQuery handles inline button callbacks.
func handleCallbackQuery(bot *tgbotapi.BotAPI, db *sql.DB, cq *tgbotapi.CallbackQuery) {
	event, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, cq.Message.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil || event.registrationCount >= event.capacity {
		sendMessage(bot, cq.Message.Chat.ID, "Регистрация закрыта")
		return
	}

	if cq.Data == "register" {
		registered, existingReg, err := isUserRegistered(db, cq.From.ID, event.id)
		if err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка проверки регистрации")
			return
		}
		if !registered {
			// First registration: add new row and update registration count.
			reg := UserRegistration{
				TelegramID:       cq.From.ID,
				Username:         cq.From.UserName,
				Name:             cq.From.FirstName + " " + cq.From.LastName,
				RegistrationDate: time.Now(),
				Email:            "",
				EventID:          event.id,
				Registred:        1, // Set to 1 when registered through the button.
				Visited:          0,
			}
			if err := registerUser(db, reg); err != nil {
				sendMessage(bot, cq.Message.Chat.ID, "Ошибка при регистрации")
				return
			}
			if err := updateEventRegistrationCount(db, event.id); err != nil {
				sendMessage(bot, cq.Message.Chat.ID, "Ошибка обновления количества регистраций")
				return
			}
			callback := tgbotapi.NewCallback(cq.ID, "Регистрация успешна!")
			bot.AnswerCallbackQuery(callback)
		} else {
			// Registration update: update the existing row.
			// Note: only active events can be updated.
			reg := UserRegistration{
				TelegramID:       cq.From.ID,
				Username:         cq.From.UserName,
				Name:             cq.From.FirstName + " " + cq.From.LastName,
				RegistrationDate: time.Now(), // Update registration date
				Email:            "",         // Could be updated if needed
				EventID:          event.id,
				Registred:        1,
				Visited:          existingReg.Visited, // Preserve visited status
			}
			if err := updateRegistration(db, reg); err != nil {
				sendMessage(bot, cq.Message.Chat.ID, "Ошибка обновления регистрации")
				return
			}
			callback := tgbotapi.NewCallback(cq.ID, "Регистрация обновлена!")
			bot.AnswerCallbackQuery(callback)
		}
	} else if cq.Data == "remove" {
		registered, _, err := isUserRegistered(db, cq.From.ID, event.id)
		if err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка проверки регистрации")
			return
		}
		if !registered {
			remaining := event.capacity - event.registrationCount
			sendMessage(bot, cq.Message.Chat.ID, "Вы не зарегистрированы. Осталось мест: "+strconv.Itoa(remaining))
			return
		}
		if err := removeRegistration(db, cq.From.ID, event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка при удалении регистрации")
			return
		}
		if err := decrementEventRegistrationCount(db, event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка обновления количества регистраций")
			return
		}
		callback := tgbotapi.NewCallback(cq.ID, "Регистрация удалена!")
		bot.AnswerCallbackQuery(callback)
	}

	updatedEvent, err := getLatestEvent(db)
	if err != nil {
		sendMessage(bot, cq.Message.Chat.ID, "Ошибка получения обновленной информации о событии")
		return
	}
	remaining := updatedEvent.capacity - updatedEvent.registrationCount
	sendMessage(bot, cq.Message.Chat.ID, "Осталось мест: "+strconv.Itoa(remaining))
}

// handleAddEmail allows the user to optionally add an email to their registration.
func handleAddEmail(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		sendMessage(bot, msg.Chat.ID, "Пожалуйста, укажите ваш email. Использование: /addemail your_email@example.com")
		return
	}
	email := strings.TrimSpace(args)
	if err := updateUserEmail(db, msg.From.ID, email); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка обновления email.")
		return
	}
	sendMessage(bot, msg.Chat.ID, "Email успешно обновлён!")
}

// handleAddEvent handles the /addevent command.
// Before inserting the new event, all old active events are marked as "past".
func handleAddEvent(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	parts := strings.Split(args, ";")
	if len(parts) < 3 {
		sendMessage(bot, msg.Chat.ID, "Использование: /addevent НазваниеСобытия;YYYY-MM-DD;Вместимость")
		return
	}
	name := strings.TrimSpace(parts[0])
	dateStr := strings.TrimSpace(parts[1])
	capacityStr := strings.TrimSpace(parts[2])
	capacity, err := strconv.Atoi(capacityStr)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Неверное число вместимости")
		return
	}
	eventDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Неверный формат даты. Используйте YYYY-MM-DD")
		return
	}

	// Update all active events to "past" (only for active events)
	stmt, err := db.Prepare("UPDATE events SET state = 'past' WHERE state = 'active'")
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка обновления состояния старых событий")
		return
	}
	_, err = stmt.Exec()
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка обновления состояния старых событий")
		return
	}

	stmt, err = db.Prepare("INSERT INTO events (name, date, capacity, state) VALUES (?, ?, ?, 'active')")
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка подготовки добавления события")
		return
	}
	defer stmt.Close()
	if _, err := stmt.Exec(name, eventDate.Format(time.RFC3339), capacity); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка добавления события")
		return
	}
	sendMessage(bot, msg.Chat.ID, "Событие успешно добавлено!")
}

// handleQRCode handles the /qrcode command.
func handleQRCode(bot *tgbotapi.BotAPI, db *sql.DB, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		sendMessage(bot, msg.Chat.ID, "Использование: /qrcode id_события")
		return
	}
	eventID, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Неверный id события")
		return
	}
	qrData := "event:" + strconv.Itoa(eventID)
	qrFile := "qrcode_event_" + strconv.Itoa(eventID) + ".png"
	if err := qrcode.WriteFile(qrData, qrcode.Medium, 256, qrFile); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка генерации QR-кода")
		return
	}
	photo := tgbotapi.NewPhotoUpload(msg.Chat.ID, qrFile)
	photo.Caption = "QR-код для регистрации на событие"
	bot.Send(photo)
}

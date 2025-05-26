package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/skip2/go-qrcode"
)

// handleCommand routes commands to corresponding handlers.
func handleCommand(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
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
		AdminCheckMiddleware(handleAddEvent)(bot, db, msg)
	case "qrcode":
		AdminCheckMiddleware(handleQRCode)(bot, db, msg)
	case "addemail":
		handleAddEmail(bot, db, msg)
	case "state":
		handleState(bot, db, msg)
	case "export":
		AdminCheckMiddleware(handleExport)(bot, db, msg)
	default:
		sendMessage(bot, msg.Chat.ID, "Неизвестная команда")
	}
}

// handleExport handles the /export command.
// Creates a CSV file with all registrations and sends it to the user
func handleExport(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	registrations, err := db.GetAllRegistrations()
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка получения данных о регистрациях: "+err.Error())
		return
	}

	if len(registrations) == 0 {
		sendMessage(bot, msg.Chat.ID, "Регистрации отсутствуют")
		return
	}

	// Create CSV file
	filename := "registrations_export_" + time.Now().Format("20060102_150405") + ".csv"
	file, err := os.Create(filename)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка создания файла: "+err.Error())
		return
	}

	// Write UTF-8 BOM for better Excel compatibility
	file.WriteString("\xEF\xBB\xBF")

	// Create CSV writer
	writer := csv.NewWriter(file)

	// Write header
	header := []string{
		"ID Telegram",
		"Имя пользователя",
		"Полное имя",
		"Email",
		"Дата регистрации",
		"Событие",
		"Дата события",
		"Зарегистрирован",
		"Посетил",
	}

	if err := writer.Write(header); err != nil {
		file.Close()
		sendMessage(bot, msg.Chat.ID, "Ошибка записи заголовка CSV: "+err.Error())
		return
	}

	// Write data
	for _, reg := range registrations {
		registeredStr := "Нет"
		if reg.Registred == 1 {
			registeredStr = "Да"
		}

		visitedStr := "Нет"
		if reg.Visited == 1 {
			visitedStr = "Да"
		}

		row := []string{
			strconv.Itoa(reg.TelegramID),
			reg.Username,
			reg.Name,
			reg.Email,
			reg.RegistrationDate.Format("02.01.2006 15:04"),
			reg.EventName,
			reg.EventDate.Format("02.01.2006"),
			registeredStr,
			visitedStr,
		}

		if err := writer.Write(row); err != nil {
			file.Close()
			sendMessage(bot, msg.Chat.ID, "Ошибка записи данных в CSV: "+err.Error())
			return
		}
	}

	// Flush the writer to ensure all data is written to the file
	writer.Flush()

	// Close the file before sending
	file.Close()

	// Send the file to the user
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка чтения файла: "+err.Error())
		return
	}

	fileDoc := tgbotapi.FileBytes{
		Name:  filename,
		Bytes: fileBytes,
	}

	doc := tgbotapi.NewDocumentUpload(msg.Chat.ID, fileDoc)
	doc.Caption = fmt.Sprintf("Экспорт данных регистраций (%d записей)", len(registrations))

	_, err = bot.Send(doc)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка отправки файла: "+err.Error())
		return
	}

	// Clean up the file after sending
	os.Remove(filename)
}

// sendMessage sends a text message to the given chat.
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	message := tgbotapi.NewMessage(chatID, text)
	bot.Send(message)
}

// handleRegister sends the register button.
func handleRegister(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	button := tgbotapi.NewInlineKeyboardButtonData("Зарегистрироваться", "register")
	row := tgbotapi.NewInlineKeyboardRow(button)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	message := tgbotapi.NewMessage(msg.Chat.ID, "Нажмите кнопку ниже, чтобы зарегистрироваться.")
	message.ReplyMarkup = keyboard
	bot.Send(message)
}

// Provide event state
func handleState(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	event, err := db.GetLatestEvent()
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
	registered, _, err := db.IsUserRegistered(msg.From.ID, event.id)
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
func handleNoDialog(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	event, err := db.GetLatestEvent()
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil || event.registrationCount >= event.capacity {
		sendMessage(bot, msg.Chat.ID, "Регистрация закрыта")
		return
	}

	registered, _, err := db.IsUserRegistered(msg.From.ID, event.id)
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
func handleImhere(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	event, err := db.GetLatestEvent()
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil {
		sendMessage(bot, msg.Chat.ID, "Нет активного события")
		return
	}
	registered, _, err := db.IsUserRegistered(msg.From.ID, event.id)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка проверки регистрации")
		return
	}
	if registered {
		err := db.UpdateVisitedStatus(msg.From.ID, event.id, 1)
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
		err := db.RegisterUser(newUser)
		if err != nil {
			sendMessage(bot, msg.Chat.ID, "Ошибка добавления пользователя")
			return
		}
		sendMessage(bot, msg.Chat.ID, "Спасибо что отметились! Это важно для нас, мы всегда рады гостям! Чтобы помочь нам лучше планировать митапы, регистрируйтесь на следующие события заранее. Спасибо!")
	}
}

// handleCallbackQuery handles inline button callbacks.
func handleCallbackQuery(bot *tgbotapi.BotAPI, db Repository, cq *tgbotapi.CallbackQuery) {
	event, err := db.GetLatestEvent()
	if err != nil {
		sendMessage(bot, cq.Message.Chat.ID, "Ошибка получения информации о событии")
		return
	}
	if event == nil || event.registrationCount >= event.capacity {
		sendMessage(bot, cq.Message.Chat.ID, "Регистрация закрыта")
		return
	}

	if cq.Data == "register" {
		registered, existingReg, err := db.IsUserRegistered(cq.From.ID, event.id)
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
			if err := db.RegisterUser(reg); err != nil {
				sendMessage(bot, cq.Message.Chat.ID, "Ошибка при регистрации")
				return
			}
			if err := db.UpdateEventRegistrationCount(event.id); err != nil {
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
			if err := db.UpdateRegistration(reg); err != nil {
				sendMessage(bot, cq.Message.Chat.ID, "Ошибка обновления регистрации")
				return
			}
			callback := tgbotapi.NewCallback(cq.ID, "Регистрация обновлена!")
			bot.AnswerCallbackQuery(callback)
		}
	} else if cq.Data == "remove" {
		registered, _, err := db.IsUserRegistered(cq.From.ID, event.id)
		if err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка проверки регистрации")
			return
		}
		if !registered {
			remaining := event.capacity - event.registrationCount
			sendMessage(bot, cq.Message.Chat.ID, "Вы не зарегистрированы. Осталось мест: "+strconv.Itoa(remaining))
			return
		}
		if err := db.RemoveRegistration(cq.From.ID, event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка при удалении регистрации")
			return
		}
		if err := db.DecrementEventRegistrationCount(event.id); err != nil {
			sendMessage(bot, cq.Message.Chat.ID, "Ошибка обновления количества регистраций")
			return
		}
		callback := tgbotapi.NewCallback(cq.ID, "Регистрация удалена!")
		bot.AnswerCallbackQuery(callback)
	}

	updatedEvent, err := db.GetLatestEvent()
	if err != nil {
		sendMessage(bot, cq.Message.Chat.ID, "Ошибка получения обновленной информации о событии")
		return
	}
	remaining := updatedEvent.capacity - updatedEvent.registrationCount
	sendMessage(bot, cq.Message.Chat.ID, "Осталось мест: "+strconv.Itoa(remaining))
}

// handleAddEmail allows the user to optionally add an email to their registration.
func handleAddEmail(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		sendMessage(bot, msg.Chat.ID, "Пожалуйста, укажите ваш email. Использование: /addemail your_email@example.com")
		return
	}
	email := strings.TrimSpace(args)
	if err := db.UpdateUserEmail(msg.From.ID, email); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка обновления email.")
		return
	}
	sendMessage(bot, msg.Chat.ID, "Email успешно обновлён!")
}

// handleAddEvent handles the /addevent command.
// Before inserting the new event, all old active events are marked as "past".
func handleAddEvent(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
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
	if err := db.MarkEventsAsPast(); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка обновления состояния старых событий")
		return
	}

	if err := db.AddEvent(name, eventDate, capacity); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка добавления события")
		return
	}
	sendMessage(bot, msg.Chat.ID, "Событие успешно добавлено!")
}

// handleQRCode handles the /qrcode command.
// Generates a QR code with a static link to the bot with the "imhere" parameter.
func handleQRCode(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
	qrData := "https://t.me/RndPHPbot?start=imhere"
	qrFile := "qrcode_event.png"
	if err := qrcode.WriteFile(qrData, qrcode.Medium, 256, qrFile); err != nil {
		sendMessage(bot, msg.Chat.ID, "Ошибка генерации QR-кода")
		return
	}
	photo := tgbotapi.NewPhotoUpload(msg.Chat.ID, qrFile)
	photo.Caption = "QR-код для отметки о посещении"
	bot.Send(photo)
	os.Remove(qrFile)
}

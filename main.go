package main

import (
	"database/sql"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
)

// Global variables
var (
	AppConfig *Config        // Application configuration
	DialogMgr *DialogManager // Dialog state manager
)

// IsAdmin checks if a username is in the list of admin users
func IsAdmin(username string) bool {
	if AppConfig == nil {
		return false
	}
	for _, admin := range AppConfig.AdminUsers {
		if admin == username {
			return true
		}
	}
	return false
}

func main() {
	// Initialize dialog manager
	DialogMgr = NewDialogManager()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}
	AppConfig = config

	log.Printf("Admin users: %v", AppConfig.AdminUsers)
	log.Printf("Mandatory fields: %v", AppConfig.MandatoryFields)

	bot, err := tgbotapi.NewBotAPI(AppConfig.BotToken)
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	db, err := sql.Open("sqlite3", "./bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize repository
	repo := NewSQLiteRepository(db)

	if err := repo.CreateTables(); err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.CallbackQuery != nil {
			handleCallbackQuery(bot, repo, update.CallbackQuery)
			continue
		}
		if update.Message != nil {
			// Check if user is in a dialog
			dialogState, eventID := DialogMgr.GetState(update.Message.From.ID)

			if dialogState != NoDialog && !update.Message.IsCommand() {
				// Handle dialog based on state
				handleDialog(bot, repo, update.Message, dialogState, eventID)
			} else if update.Message.IsCommand() {
				handleCommand(bot, repo, update.Message)
			} else {
				// No dialog mode: show appropriate button based on registration status
				handleNoDialog(bot, repo, update.Message)
			}
		}
	}
}

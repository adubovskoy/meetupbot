package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
)

// AdminUsers is a global variable that holds the list of admin usernames
var AdminUsers []string

// IsAdmin checks if a username is in the list of admin users
func IsAdmin(username string) bool {
	for _, admin := range AdminUsers {
		if admin == username {
			return true
		}
	}
	return false
}

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}
	
	// Initialize admin users from environment variable
	adminUsersEnv := os.Getenv("ADMIN_USERS")
	if adminUsersEnv != "" {
		AdminUsers = strings.Split(adminUsersEnv, ",")
		// Trim spaces from usernames
		for i, username := range AdminUsers {
			AdminUsers[i] = strings.TrimSpace(username)
		}
		log.Printf("Admin users: %v", AdminUsers)
	} else {
		log.Println("Warning: ADMIN_USERS environment variable not set. No users will have admin privileges.")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
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
			if update.Message.IsCommand() {
				handleCommand(bot, repo, update.Message)
			} else {
				// New dialog mode: show appropriate button based on registration status
				handleNoDialog(bot, repo, update.Message)
			}
		}
	}
}

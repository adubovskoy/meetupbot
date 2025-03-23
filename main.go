package main

import (
	"database/sql"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
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

	if err := createTables(db); err != nil {
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
			handleCallbackQuery(bot, db, update.CallbackQuery)
			continue
		}
		if update.Message != nil {
			if update.Message.IsCommand() {
				handleCommand(bot, db, update.Message)
			} else {
				// New dialog mode: show appropriate button based on registration status
				handleNoDialog(bot, db, update.Message)
			}
		}
	}
}

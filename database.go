package main

import (
	"database/sql"
	"time"
)

// createTables создаёт необходимые таблицы для пользователей и событий.
func createTables(db *sql.DB) error {
	userTable := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER,
		username TEXT,
		name TEXT,
		registration_date DATETIME,
		email TEXT,
		event_id INTEGER,
		registred INTEGER DEFAULT 0,
		visited INTEGER DEFAULT 0
	);`

	eventTable := `CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		date DATETIME,
		capacity INTEGER,
		registration_count INTEGER DEFAULT 0,
		state TEXT DEFAULT 'active'
	);`

	if _, err := db.Exec(userTable); err != nil {
		return err
	}
	if _, err := db.Exec(eventTable); err != nil {
		return err
	}
	return nil
}

// getLatestEvent возвращает последнее активное событие.
func getLatestEvent(db *sql.DB) (*Event, error) {
	row := db.QueryRow("SELECT id, name, date, capacity, registration_count FROM events WHERE state = 'active' ORDER BY date DESC LIMIT 1")
	var ev Event
	var dateStr string
	err := row.Scan(&ev.id, &ev.name, &dateStr, &ev.capacity, &ev.registrationCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ev.date, _ = time.Parse(time.RFC3339, dateStr)
	return &ev, nil
}

// registerUser сохраняет данные о регистрации пользователя.
func registerUser(db *sql.DB, reg UserRegistration) error {
	stmt, err := db.Prepare("INSERT INTO users (telegram_id, username, name, registration_date, email, event_id, registred, visited) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(reg.TelegramID, reg.Username, reg.Name, reg.RegistrationDate.Format(time.RFC3339), reg.Email, reg.EventID, reg.Registred, reg.Visited)
	return err
}

// updateUserEmail обновляет email пользователя.
func updateUserEmail(db *sql.DB, telegramID int, email string) error {
	stmt, err := db.Prepare("UPDATE users SET email = ? WHERE telegram_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(email, telegramID)
	return err
}

// updateEventRegistrationCount увеличивает количество регистраций для события.
func updateEventRegistrationCount(db *sql.DB, eventID int) error {
	stmt, err := db.Prepare("UPDATE events SET registration_count = registration_count + 1 WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(eventID)
	return err
}

// removeRegistration удаляет запись регистрации пользователя для события.
func removeRegistration(db *sql.DB, telegramID int, eventID int) error {
	stmt, err := db.Prepare("DELETE FROM users WHERE telegram_id = ? AND event_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(telegramID, eventID)
	return err
}

// decrementEventRegistrationCount уменьшает количество регистраций для события.
func decrementEventRegistrationCount(db *sql.DB, eventID int) error {
	stmt, err := db.Prepare("UPDATE events SET registration_count = registration_count - 1 WHERE id = ? AND registration_count > 0")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(eventID)
	return err
}

// isUserRegistered проверяет, зарегистрирован ли пользователь для события.
func isUserRegistered(db *sql.DB, telegramID int, eventID int) (bool, *UserRegistration, error) {
	row := db.QueryRow("SELECT telegram_id, username, name, registration_date, email, event_id, registred, visited FROM users WHERE telegram_id = ? AND event_id = ?", telegramID, eventID)
	var reg UserRegistration
	err := row.Scan(&reg.TelegramID, &reg.Username, &reg.Name, &reg.RegistrationDate, &reg.Email, &reg.EventID, &reg.Registred, &reg.Visited)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &reg, nil
}

// updateVisitedStatus обновляет статус посещения пользователя для события.
func updateVisitedStatus(db *sql.DB, telegramID int, eventID int, visited int) error {
	stmt, err := db.Prepare("UPDATE users SET visited = ? WHERE telegram_id = ? AND event_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(visited, telegramID, eventID)
	return err
}

package main

import (
	"database/sql"
	"time"
)

// Repository defines the interface for database operations
type Repository interface {
	CreateTables() error
	GetLatestEvent() (*Event, error)
	RegisterUser(reg UserRegistration) error
	UpdateUserEmail(telegramID int, email string) error
	UpdateEventRegistrationCount(eventID int) error
	RemoveRegistration(telegramID int, eventID int) error
	DecrementEventRegistrationCount(eventID int) error
	IsUserRegistered(telegramID int, eventID int) (bool, *UserRegistration, error)
	UpdateVisitedStatus(telegramID int, eventID int, visited int) error
	UpdateRegistration(reg UserRegistration) error
	MarkEventsAsPast() error
	AddEvent(name string, date time.Time, capacity int) error
	GetAllRegistrations() ([]UserRegistrationWithEvent, error)
	HasUserInfo(telegramID int) (bool, string, string, error)
	UpdateUserName(telegramID int, name string) error
	// Add method for SQL statement preparation
	Prepare(query string) (*sql.Stmt, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// SQLiteRepository implements the Repository interface
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLiteRepository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// CreateTables creates the necessary tables for users and events
func (r *SQLiteRepository) CreateTables() error {
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

	if _, err := r.db.Exec(userTable); err != nil {
		return err
	}
	if _, err := r.db.Exec(eventTable); err != nil {
		return err
	}
	return nil
}

// GetLatestEvent returns the latest active event
func (r *SQLiteRepository) GetLatestEvent() (*Event, error) {
	row := r.db.QueryRow("SELECT id, name, date, capacity, registration_count FROM events WHERE state = 'active' ORDER BY date DESC LIMIT 1")
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

// RegisterUser saves the user registration data or updates existing unregistered user
func (r *SQLiteRepository) RegisterUser(reg UserRegistration) error {
	// Check if user exists but is unregistered
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE telegram_id = ? AND event_id = ? AND registred = 0",
		reg.TelegramID, reg.EventID).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// User exists but is unregistered, update their registration status
		stmt, err := r.db.Prepare("UPDATE users SET username = ?, name = ?, registration_date = ?, email = ?, registred = 1 WHERE telegram_id = ? AND event_id = ?")
		if err != nil {
			return err
		}
		defer stmt.Close()
		_, err = stmt.Exec(reg.Username, reg.Name, reg.RegistrationDate.Format(time.RFC3339), reg.Email, reg.TelegramID, reg.EventID)
		return err
	}

	// User doesn't exist, insert new record
	stmt, err := r.db.Prepare("INSERT INTO users (telegram_id, username, name, registration_date, email, event_id, registred, visited) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(reg.TelegramID, reg.Username, reg.Name, reg.RegistrationDate.Format(time.RFC3339), reg.Email, reg.EventID, reg.Registred, reg.Visited)
	return err
}

// UpdateUserEmail updates the user's email
func (r *SQLiteRepository) UpdateUserEmail(telegramID int, email string) error {
	stmt, err := r.db.Prepare("UPDATE users SET email = ? WHERE telegram_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(email, telegramID)
	return err
}

// UpdateEventRegistrationCount increments the registration count for an event
func (r *SQLiteRepository) UpdateEventRegistrationCount(eventID int) error {
	stmt, err := r.db.Prepare("UPDATE events SET registration_count = registration_count + 1 WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(eventID)
	return err
}

// RemoveRegistration updates a user's registration status to unregistered
func (r *SQLiteRepository) RemoveRegistration(telegramID int, eventID int) error {
	stmt, err := r.db.Prepare("UPDATE users SET registred = 0 WHERE telegram_id = ? AND event_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(telegramID, eventID)
	return err
}

// DecrementEventRegistrationCount decreases the registration count for an event
func (r *SQLiteRepository) DecrementEventRegistrationCount(eventID int) error {
	stmt, err := r.db.Prepare("UPDATE events SET registration_count = registration_count - 1 WHERE id = ? AND registration_count > 0")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(eventID)
	return err
}

// IsUserRegistered checks if a user is registered for an event
func (r *SQLiteRepository) IsUserRegistered(telegramID int, eventID int) (bool, *UserRegistration, error) {
	row := r.db.QueryRow("SELECT telegram_id, username, name, registration_date, email, event_id, registred, visited FROM users WHERE telegram_id = ? AND event_id = ?", telegramID, eventID)
	var reg UserRegistration
	var dateStr string
	err := row.Scan(&reg.TelegramID, &reg.Username, &reg.Name, &dateStr, &reg.Email, &reg.EventID, &reg.Registred, &reg.Visited)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		return false, nil, err
	}
	reg.RegistrationDate, _ = time.Parse(time.RFC3339, dateStr)
	return reg.Registred == 1, &reg, nil
}

// UpdateVisitedStatus updates a user's visited status for an event
func (r *SQLiteRepository) UpdateVisitedStatus(telegramID int, eventID int, visited int) error {
	stmt, err := r.db.Prepare("UPDATE users SET visited = ? WHERE telegram_id = ? AND event_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(visited, telegramID, eventID)
	return err
}

// UpdateRegistration updates a user's registration for an event
func (r *SQLiteRepository) UpdateRegistration(reg UserRegistration) error {
	stmt, err := r.db.Prepare("UPDATE users SET username = ?, name = ?, registration_date = ?, email = ?, registred = ? WHERE telegram_id = ? AND event_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(reg.Username, reg.Name, reg.RegistrationDate.Format(time.RFC3339), reg.Email, reg.Registred, reg.TelegramID, reg.EventID)
	return err
}

// MarkEventsAsPast updates all active events to past status
func (r *SQLiteRepository) MarkEventsAsPast() error {
	stmt, err := r.db.Prepare("UPDATE events SET state = 'past' WHERE state = 'active'")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec()
	return err
}

// AddEvent adds a new active event
func (r *SQLiteRepository) AddEvent(name string, date time.Time, capacity int) error {
	stmt, err := r.db.Prepare("INSERT INTO events (name, date, capacity, state) VALUES (?, ?, ?, 'active')")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(name, date.Format(time.RFC3339), capacity)
	return err
}

// Prepare forwards the prepare statement to the underlying database
func (r *SQLiteRepository) Prepare(query string) (*sql.Stmt, error) {
	return r.db.Prepare(query)
}

// Exec forwards the exec statement to the underlying database
func (r *SQLiteRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	return r.db.Exec(query, args...)
}

// HasUserInfo checks if a user has previously registered with name and email
func (r *SQLiteRepository) HasUserInfo(telegramID int) (bool, string, string, error) {
	query := `
		SELECT name, email FROM users 
		WHERE telegram_id = ? 
		AND name IS NOT NULL AND name != '' 
		AND email IS NOT NULL AND email != '' 
		LIMIT 1
	`
	
	row := r.db.QueryRow(query, telegramID)
	
	var name, email string
	err := row.Scan(&name, &email)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", "", nil
		}
		return false, "", "", err
	}
	
	return true, name, email, nil
}

// UpdateUserName updates the user's name
func (r *SQLiteRepository) UpdateUserName(telegramID int, name string) error {
	stmt, err := r.db.Prepare("UPDATE users SET name = ? WHERE telegram_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(name, telegramID)
	return err
}

// GetAllRegistrations retrieves all user registrations with event details
func (r *SQLiteRepository) GetAllRegistrations() ([]UserRegistrationWithEvent, error) {
	query := `
        SELECT u.telegram_id, u.username, u.name, u.registration_date, u.email, u.event_id, u.registred, u.visited,
               e.name, e.date
        FROM users u
        JOIN events e ON u.event_id = e.id
        ORDER BY e.date DESC, u.name ASC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registrations []UserRegistrationWithEvent

	for rows.Next() {
		var reg UserRegistrationWithEvent
		var regDateStr string
		var eventDateStr, eventName sql.NullString

		err := rows.Scan(
			&reg.TelegramID,
			&reg.Username,
			&reg.Name,
			&regDateStr,
			&reg.Email,
			&reg.EventID,
			&reg.Registred,
			&reg.Visited,
			&eventName,
			&eventDateStr,
		)
		if err != nil {
			return nil, err
		}

		reg.RegistrationDate, _ = time.Parse(time.RFC3339, regDateStr)

		if eventName.Valid {
			reg.EventName = eventName.String
		} else {
			reg.EventName = "Unknown Event"
		}

		if eventDateStr.Valid {
			reg.EventDate, _ = time.Parse(time.RFC3339, eventDateStr.String)
		} else {
			reg.EventDate = time.Time{} // Zero time
		}

		registrations = append(registrations, reg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return registrations, nil
}

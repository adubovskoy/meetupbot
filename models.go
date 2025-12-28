package main

import "time"

// UserRegistration represents a user registration record.
type UserRegistration struct {
	TelegramID       int       // TelegramID is the unique identifier for the user on Telegram.
	Username         string    // Username is the user's Telegram username.
	Name             string    // Name is the user's full name.
	RegistrationDate time.Time // RegistrationDate is the date and time when the user registered.
	Email            string    // Email is the user's email address.
	EventID          int       // EventID is the identifier of the event the user registered for.
	Registred        int       // Registred indicates whether the user is registered (1) or not (0).
	Visited          int       // Visited indicates whether the user has visited the event (1) or not (0).
}

// Event represents an event record.
type Event struct {
	id                int       // id is the unique identifier for the event.
	name              string    // name is the name of the event.
	date              time.Time // date is the date and time when the event is scheduled.
	capacity          int       // capacity is the maximum number of participants allowed.
	registrationCount int       // registrationCount is the number of participants registered for the event.
}

// UserRegistrationWithEvent extends UserRegistration with event information
type UserRegistrationWithEvent struct {
	UserRegistration           // Embedded UserRegistration
	EventName        string    // Name of the event
	EventDate        time.Time // Date of the event
}

// WaitlistEntry represents a user in the waitlist for an event.
type WaitlistEntry struct {
	TelegramID int       // TelegramID is the unique identifier for the user on Telegram.
	ChatID     int64     // ChatID is the chat ID for sending proactive messages.
	Username   string    // Username is the user's Telegram username.
	EventID    int       // EventID is the identifier of the event.
	JoinedDate time.Time // JoinedDate is when the user joined the waitlist.
}

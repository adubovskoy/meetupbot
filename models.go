package main

import "time"

// UserRegistration represents a user registration record.
type UserRegistration struct {
	TelegramID       int
	Username         string
	Name             string
	RegistrationDate time.Time
	Email            string
	EventID          int
}

// Event represents an event record.
type Event struct {
	id                int
	name              string
	date              time.Time
	capacity          int
	registrationCount int
}

package main

import "time"

// UserRegistration представляет запись о регистрации пользователя.
type UserRegistration struct {
	TelegramID       int
	Username         string
	Name             string
	RegistrationDate time.Time
	Email            string
	EventID          int
	Registred        int // 1, если зарегистрирован (предварительно), 0 — если нет
	Visited          int // 1, если посетил событие, 0 — если нет
}

// Event представляет запись о событии.
type Event struct {
	id                int
	name              string
	date              time.Time
	capacity          int
	registrationCount int
}

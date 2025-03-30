package main

import (
	"regexp"
	"strings"
	"sync"
)

// DialogState represents the current state of a user's dialog with the bot
type DialogState int

const (
	NoDialog DialogState = iota
	WaitingForName
	WaitingForEmail
)

// UserDialogState stores the dialog state for a user
type UserDialogState struct {
	State    DialogState
	EventID  int
	UserData map[string]string // For storing temporary data during dialog
}

// DialogManager manages dialog states for users
type DialogManager struct {
	userStates map[int]*UserDialogState // Map of telegram_id to dialog state
	mu         sync.RWMutex             // Mutex for thread safety
}

// NewDialogManager creates a new DialogManager
func NewDialogManager() *DialogManager {
	return &DialogManager{
		userStates: make(map[int]*UserDialogState),
	}
}

// SetState sets the dialog state for a user
func (dm *DialogManager) SetState(telegramID int, state DialogState, eventID int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.userStates[telegramID]; !exists {
		dm.userStates[telegramID] = &UserDialogState{
			UserData: make(map[string]string),
		}
	}

	dm.userStates[telegramID].State = state
	dm.userStates[telegramID].EventID = eventID
}

// GetState gets the dialog state for a user
func (dm *DialogManager) GetState(telegramID int) (DialogState, int) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if state, exists := dm.userStates[telegramID]; exists {
		return state.State, state.EventID
	}
	return NoDialog, 0
}

// SetUserData sets temporary data for a user during dialog
func (dm *DialogManager) SetUserData(telegramID int, key, value string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.userStates[telegramID]; !exists {
		dm.userStates[telegramID] = &UserDialogState{
			UserData: make(map[string]string),
		}
	}

	dm.userStates[telegramID].UserData[key] = value
}

// GetUserData gets temporary data for a user during dialog
func (dm *DialogManager) GetUserData(telegramID int, key string) string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if state, exists := dm.userStates[telegramID]; exists {
		if value, ok := state.UserData[key]; ok {
			return value
		}
	}
	return ""
}

// ClearState clears the dialog state for a user
func (dm *DialogManager) ClearState(telegramID int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	delete(dm.userStates, telegramID)
}

// ValidateName validates that the name is in the format "Surname Name"
func ValidateName(name string) bool {
	// Trim spaces and check if there's at least one space between words
	trimmedName := strings.TrimSpace(name)
	parts := strings.Fields(trimmedName)
	
	// We need at least 2 parts (surname and name)
	return len(parts) >= 2
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	// Simple regex for email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

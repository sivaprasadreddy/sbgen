package main

import (
	"context"
	"errors"
	"strings"

	"github.com/egoist/mygo"
)

// ConfirmOptions describes a confirmation dialog.
type ConfirmOptions struct {
	// Message is the question, e.g. "Delete “REST API”?".
	Message string `json:"message"`
	// Detail explains the consequence, below Message in a smaller font.
	Detail string `json:"detail,omitempty"`
	// ConfirmLabel names the button that goes ahead, e.g. "Delete".
	ConfirmLabel string `json:"confirmLabel"`
	// Destructive marks an action that discards data. Its button is set apart,
	// and Cancel answers both Enter and Escape.
	Destructive bool `json:"destructive,omitempty"`
}

// DialogService shows native dialogs for the page.
type DialogService struct {
	// show displays a message dialog: mygo.Dialog.Message, or a fake in tests.
	show func(mygo.MessageOptions) (mygo.MessageResult, error)
}

// NewDialogService returns a service that shows MyGo's native dialogs.
func NewDialogService() *DialogService {
	return &DialogService{show: mygo.Dialog.Message}
}

// Confirm asks the user to confirm an action, as a sheet of the calling
// window on macOS, and reports whether they chose to go ahead.
func (s *DialogService) Confirm(ctx context.Context, options ConfirmOptions) (bool, error) {
	message := strings.TrimSpace(options.Message)
	label := strings.TrimSpace(options.ConfirmLabel)
	if message == "" || label == "" {
		return false, errors.New("a confirmation needs a message and a confirm label")
	}
	const confirm, cancel = 0, 1
	dialog := mygo.MessageOptions{
		Parent:  mygo.CallerWindow(ctx),
		Type:    mygo.MessageQuestion,
		Message: message,
		Detail:  options.Detail,
		// macOS shows the first button rightmost, as its alerts do.
		Buttons:       []string{label, "Cancel"},
		DefaultButton: confirm,
		CancelButton:  cancel,
	}
	if options.Destructive {
		dialog.Type = mygo.MessageWarning
		dialog.DefaultButton = cancel
		dialog.DestructiveButtons = []int{confirm}
	}
	result, err := s.show(dialog)
	if err != nil {
		return false, err
	}
	return result.Button == confirm, nil
}

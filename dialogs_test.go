package main

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/egoist/mygo"
)

// fakeDialogs records the dialog it was asked to show and clicks button.
func fakeDialogs(button int, err error) (*DialogService, *mygo.MessageOptions) {
	shown := &mygo.MessageOptions{}
	return &DialogService{show: func(o mygo.MessageOptions) (mygo.MessageResult, error) {
		*shown = o
		return mygo.MessageResult{Button: button}, err
	}}, shown
}

func TestConfirmReportsTheChosenButton(t *testing.T) {
	for button, want := range map[int]bool{0: true, 1: false} {
		s, _ := fakeDialogs(button, nil)
		got, err := s.Confirm(context.Background(), ConfirmOptions{Message: "Go?", ConfirmLabel: "Go"})
		if err != nil || got != want {
			t.Errorf("button %d: got %v, %v; want %v", button, got, err, want)
		}
	}
}

func TestConfirmShowsAQuestionWithConfirmAndCancel(t *testing.T) {
	s, shown := fakeDialogs(1, nil)
	if _, err := s.Confirm(context.Background(), ConfirmOptions{Message: " Save? ", Detail: "Details", ConfirmLabel: " Save "}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if shown.Type != mygo.MessageQuestion || shown.Message != "Save?" || shown.Detail != "Details" {
		t.Errorf("unexpected dialog: %+v", *shown)
	}
	if !slices.Equal(shown.Buttons, []string{"Save", "Cancel"}) || shown.DefaultButton != 0 || shown.CancelButton != 1 || len(shown.DestructiveButtons) != 0 {
		t.Errorf("unexpected buttons: %+v", *shown)
	}
}

func TestConfirmMakesCancelTheDefaultOfDestructiveActions(t *testing.T) {
	s, shown := fakeDialogs(1, nil)
	if _, err := s.Confirm(context.Background(), ConfirmOptions{Message: "Delete?", ConfirmLabel: "Delete", Destructive: true}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if shown.Type != mygo.MessageWarning || shown.DefaultButton != 1 || shown.CancelButton != 1 || !slices.Equal(shown.DestructiveButtons, []int{0}) {
		t.Errorf("unexpected destructive dialog: %+v", *shown)
	}
}

func TestConfirmRejectsIncompleteOptionsAndReportsDialogErrors(t *testing.T) {
	s, _ := fakeDialogs(0, nil)
	for _, options := range []ConfirmOptions{{ConfirmLabel: "Go"}, {Message: "Go?", ConfirmLabel: " "}} {
		if _, err := s.Confirm(context.Background(), options); err == nil {
			t.Errorf("expected an error for %+v", options)
		}
	}
	failing, _ := fakeDialogs(0, errors.New("no window"))
	if ok, err := failing.Confirm(context.Background(), ConfirmOptions{Message: "Go?", ConfirmLabel: "Go"}); err == nil || ok {
		t.Errorf("expected the dialog's error and no confirmation, got %v, %v", ok, err)
	}
}

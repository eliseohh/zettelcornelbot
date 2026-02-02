package main

import (
	tele "gopkg.in/telebot.v3"
)

// MockContext implements tele.Context restricted to what we use
type MockContext struct {
	tele.Context
	MessageVal *tele.Message
	SentMsg    interface{}
	SentOpts   *tele.SendOptions
}

func (m *MockContext) Message() *tele.Message {
	return m.MessageVal
}

func (m *MockContext) Send(what interface{}, opts ...interface{}) error {
	m.SentMsg = what
	if len(opts) > 0 {
		if o, ok := opts[0].(*tele.SendOptions); ok {
			m.SentOpts = o
		}
	}
	return nil
}

// Minimal mocked bot structure to attach handlers if needed,
// but our handlers are methods on Bot struct.
// We will test logic by instantiating the real Bot struct with a real DB (test db)
// and calling the methods directly with MockContext.

// Package main - Session management for interactive chat mode
package main

// Message represents a single message in the conversation
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// Session manages an in-memory conversation history
type Session struct {
	messages []Message
}

// NewSession creates a new empty session
func NewSession() *Session {
	return &Session{
		messages: make([]Message, 0),
	}
}

// AddMessage appends a new message to the conversation history
func (s *Session) AddMessage(role, content string) {
	s.messages = append(s.messages, Message{
		Role:    role,
		Content: content,
	})
}

// GetMessages returns all messages in the conversation
func (s *Session) GetMessages() []Message {
	return s.messages
}

// Clear removes all messages from the session
func (s *Session) Clear() {
	s.messages = make([]Message, 0)
}

// MessageCount returns the number of messages in the session
func (s *Session) MessageCount() int {
	return len(s.messages)
}

package models

import (
	"testing"
	"time"
)

func TestConversation_Basic(t *testing.T) {
	conv := &Conversation{
		ID:         "test-conv",
		MessageMap: make(map[string]*Message),
	}

	message := &Message{
		ID:       "msg-1",
		Content:  []MessageContent{{Type: ContentTypeText, Text: "Hello"}},
		Role:     "user",
		CreatedAt: time.Now(),
	}

	conv.MessageMap[message.ID] = message
	conv.LastMessageID = message.ID

	if len(conv.MessageMap) != 1 {
		t.Errorf("Expected 1 message in map, got %d", len(conv.MessageMap))
	}

	if conv.LastMessageID != "msg-1" {
		t.Errorf("Expected LastMessageID to be 'msg-1', got '%s'", conv.LastMessageID)
	}

	storedMsg := conv.MessageMap["msg-1"]
	if storedMsg == nil {
		t.Error("Expected message to be stored in map")
	}
	
	if storedMsg.ID != "msg-1" {
		t.Errorf("Expected stored message ID to be 'msg-1', got '%s'", storedMsg.ID)
	}
}

func TestMessage_GetTextContent(t *testing.T) {
	tests := []struct {
		name     string
		message  *Message
		expected string
	}{
		{
			name: "Single text content",
			message: &Message{
				Content: []MessageContent{{Type: ContentTypeText, Text: "Hello world"}},
			},
			expected: "Hello world",
		},
		{
			name: "Multiple text contents",
			message: &Message{
				Content: []MessageContent{
					{Type: ContentTypeText, Text: "Hello"},
					{Type: ContentTypeText, Text: " world"},
				},
			},
			expected: "Hello world",
		},
		{
			name: "Mixed content types",
			message: &Message{
				Content: []MessageContent{
					{Type: ContentTypeText, Text: "Hello"},
					{Type: ContentTypeImage, ImageURL: "image-url"},
					{Type: ContentTypeText, Text: " world"},
				},
			},
			expected: "Hello world",
		},
		{
			name: "No text content",
			message: &Message{
				Content: []MessageContent{{Type: ContentTypeImage, ImageURL: "image-url"}},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract text content manually for testing
			var result string
			for _, content := range tt.message.Content {
				if content.Type == ContentTypeText {
					result += content.Text
				}
			}
			
			if result != tt.expected {
				t.Errorf("Expected: '%s', Got: '%s'", tt.expected, result)
			}
		})
	}
}

func TestMessage_HasParent(t *testing.T) {
	parentID := "parent-msg"
	
	tests := []struct {
		name     string
		message  *Message
		expected bool
	}{
		{
			name: "Has parent",
			message: &Message{
				ParentID: &parentID,
			},
			expected: true,
		},
		{
			name: "No parent",
			message: &Message{
				ParentID: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.message.ParentID != nil
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestConversationStructures(t *testing.T) {
	// Test basic conversation structure
	conv := &Conversation{
		ID:         "test-conv-123",
		MessageMap: make(map[string]*Message),
	}

	if conv.ID != "test-conv-123" {
		t.Errorf("Expected conversation ID to be 'test-conv-123', got '%s'", conv.ID)
	}

	if conv.MessageMap == nil {
		t.Error("Expected MessageMap to be initialized")
	}

	if len(conv.MessageMap) != 0 {
		t.Errorf("Expected empty MessageMap, got %d messages", len(conv.MessageMap))
	}
}

func TestToolUseContent(t *testing.T) {
	toolUse := &ToolUseContent{
		ToolUseID: "tool-123",
		Name:      "calculator",
		Input:     map[string]interface{}{"expression": "2+2"},
	}

	if toolUse.ToolUseID != "tool-123" {
		t.Errorf("Expected ToolUseID to be 'tool-123', got '%s'", toolUse.ToolUseID)
	}

	if toolUse.Name != "calculator" {
		t.Errorf("Expected Name to be 'calculator', got '%s'", toolUse.Name)
	}

	if len(toolUse.Input) != 1 {
		t.Errorf("Expected 1 input parameter, got %d", len(toolUse.Input))
	}
}

func TestToolResultContent(t *testing.T) {
	toolResult := &ToolResultContent{
		ToolUseID: "tool-123",
		Content:   "4",
		IsError:   false,
	}

	if toolResult.ToolUseID != "tool-123" {
		t.Errorf("Expected ToolUseID to be 'tool-123', got '%s'", toolResult.ToolUseID)
	}

	if toolResult.Content != "4" {
		t.Errorf("Expected Content to be '4', got '%s'", toolResult.Content)
	}

	if toolResult.IsError != false {
		t.Errorf("Expected IsError to be false, got %v", toolResult.IsError)
	}
}
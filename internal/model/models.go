package model

import "time"

// Direction of a message relative to the WhatsApp account.
const (
	DirectionIncoming = "incoming"
	DirectionOutgoing = "outgoing"
)

// Message types supported by the platform.
const (
	MessageTypeText     = "text"
	MessageTypeImage    = "image"
	MessageTypeDocument = "document"
	MessageTypeVideo    = "video"
	MessageTypeAudio    = "audio"
	MessageTypeSticker  = "sticker"
)

// Message delivery status values.
const (
	StatusPending   = "pending"
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusRead      = "read"
	StatusFailed    = "failed"
)

// Device / session connection status values.
const (
	DeviceConnected    = "connected"
	DeviceConnecting   = "connecting"
	DeviceDisconnected = "disconnected"
	DeviceLoggedOut    = "logged_out"
)

// Broadcast status values.
const (
	BroadcastPending = "pending"
	BroadcastSending = "sending"
	BroadcastSuccess = "success"
	BroadcastFailed  = "failed"
)

// Session represents a persisted WhatsApp account/session record. The actual
// whatsmeow device credentials live in whatsmeow's own sqlstore tables; this
// row stores display metadata and connection status for the dashboard.
type Session struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Phone     string     `gorm:"index" json:"phone"`
	PushName  string     `json:"push_name"`
	Status    string     `gorm:"default:logged_out" json:"status"`
	JID       string     `gorm:"column:jid" json:"jid"`
	LastSeen  *time.Time `json:"last_seen"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Contact is a WhatsApp contact the platform has interacted with.
type Contact struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Phone         string     `gorm:"uniqueIndex" json:"phone"`
	JID           string     `gorm:"column:jid;index" json:"jid"`
	Name          string     `json:"name"`
	Avatar        string     `json:"avatar"`
	LastMessageAt *time.Time `json:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Conversation groups messages exchanged with a single contact.
type Conversation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ContactID     uint       `gorm:"index" json:"contact_id"`
	Contact       *Contact   `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
	LastMessage   string     `json:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at"`
	UnreadCount   int        `gorm:"default:0" json:"unread_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Message is a single inbound or outbound WhatsApp message.
type Message struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ConversationID uint       `gorm:"index" json:"conversation_id"`
	MessageID      string     `gorm:"index" json:"message_id"`
	Direction      string     `json:"direction"`
	MessageType    string     `json:"message_type"`
	Content        string     `json:"content"`
	MediaURL       string     `json:"media_url"`
	FileName       string     `json:"file_name"`
	Status         string     `gorm:"default:pending" json:"status"`
	SentAt         *time.Time `json:"sent_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Broadcast is a bulk-send campaign.
type Broadcast struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	Message      string    `json:"message"`
	Status       string    `gorm:"default:pending" json:"status"`
	TotalTarget  int       `json:"total_target"`
	TotalSuccess int       `json:"total_success"`
	TotalFailed  int       `json:"total_failed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BroadcastDetail is the per-recipient record of a broadcast.
type BroadcastDetail struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	BroadcastID  uint      `gorm:"index" json:"broadcast_id"`
	Phone        string    `json:"phone"`
	Status       string    `gorm:"default:pending" json:"status"`
	ErrorMessage string    `json:"error_message"`
	RetryCount   int       `gorm:"default:0" json:"retry_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// Setting is a generic key/value store for app settings (e.g. the API key).
type Setting struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AllModels returns every model for auto-migration.
func AllModels() []interface{} {
	return []interface{}{
		&Session{},
		&Contact{},
		&Conversation{},
		&Message{},
		&Broadcast{},
		&BroadcastDetail{},
		&Setting{},
	}
}

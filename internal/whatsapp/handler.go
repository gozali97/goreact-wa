package whatsapp

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"wa-proxy/internal/logsvc"
	"wa-proxy/internal/model"
	"wa-proxy/internal/webhook"
)

// eventHandler is registered with the whatsmeow client and routes events.
func (m *Manager) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		// Only treat as fully connected once we're actually logged in. During
		// QR pairing the socket connects before login; don't clear the QR yet.
		m.mu.RLock()
		loggedIn := m.client != nil && m.client.IsLoggedIn()
		m.mu.RUnlock()
		if loggedIn {
			m.setStatus(model.DeviceConnected)
			m.persistSession()
			// Ensure an API key exists once connected (generate if missing).
			if key, err := m.keys.EnsureExists(); err == nil && key != "" {
				logsvc.Info("apikey", "api key ready on connect", nil)
			}
			// Clean up any legacy LID-keyed contacts now that the LID map is
			// available (runs in background; no-op once clean).
			go m.BackfillLIDContacts()
		}
	case *events.PairSuccess:
		m.setStatus(model.DeviceConnected)
		m.persistSession()
	case *events.Disconnected:
		// Auto-reconnect is enabled; report connecting rather than fully down.
		if m.Status() != model.DeviceLoggedOut {
			m.setStatus(model.DeviceConnecting)
		}
	case *events.LoggedOut:
		m.mu.Lock()
		m.client = nil
		m.mu.Unlock()
		_ = m.repo.ClearSessions()
		_ = m.keys.Clear()
		m.setStatus(model.DeviceLoggedOut)
	case *events.StreamReplaced:
		m.setStatus(model.DeviceDisconnected)
	case *events.Message:
		m.handleIncomingMessage(v)
	case *events.Receipt:
		m.handleReceipt(v)
	}
}

// handleIncomingMessage persists inbound messages and emits a WS event.
func (m *Manager) handleIncomingMessage(evt *events.Message) {
	// Ignore our own messages and group chats (single-device scope per PRD).
	if evt.Info.IsFromMe || evt.Info.IsGroup {
		return
	}

	phone, chatJID := m.resolveSender(evt.Info.Sender, evt.Info.SenderAlt)
	pushName := evt.Info.PushName

	content, msgType, mediaURL, fileName := m.extractContent(evt)
	if content == "" && mediaURL == "" {
		return // unsupported message type
	}

	contact, err := m.repo.UpsertContact(phone, pushName, chatJID)
	if err != nil {
		m.log.Warnf("incoming upsert contact: %v", err)
		return
	}
	conv, err := m.repo.GetOrCreateConversation(contact.ID)
	if err != nil {
		m.log.Warnf("incoming conversation: %v", err)
		return
	}

	ts := evt.Info.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	msg := &model.Message{
		ConversationID: conv.ID,
		MessageID:      evt.Info.ID,
		Direction:      model.DirectionIncoming,
		MessageType:    msgType,
		Content:        content,
		MediaURL:       mediaURL,
		FileName:       fileName,
		Status:         model.StatusDelivered,
		SentAt:         &ts,
	}
	if err := m.repo.CreateMessage(msg); err != nil {
		m.log.Warnf("incoming create message: %v", err)
		return
	}

	preview := content
	if preview == "" {
		preview = "[" + msgType + "]"
	}
	_ = m.repo.TouchConversation(conv.ID, preview, true)
	m.emitNewMessage(conv.ID, contact, msg)
	logsvc.Success("whatsapp.incoming", "received "+msgType+" message", map[string]interface{}{
		"from": phone, "type": msgType, "message_id": msg.MessageID,
	})

	// Outbound webhook (WAHA-style "message" event).
	if m.webhook != nil {
		m.webhook.Emit(webhook.EventMessage, map[string]interface{}{
			"conversation_id": conv.ID,
			"from":            phone,
			"name":            pushName,
			"message_id":      msg.MessageID,
			"type":            msg.MessageType,
			"content":         msg.Content,
			"media_url":       msg.MediaURL,
			"file_name":       msg.FileName,
			"timestamp":       ts.Unix(),
		})
	}
}

// extractContent pulls text/media out of a message, downloading media to disk.
func (m *Manager) extractContent(evt *events.Message) (content, msgType, mediaURL, fileName string) {
	msg := evt.Message
	switch {
	case msg.GetConversation() != "":
		return msg.GetConversation(), model.MessageTypeText, "", ""
	case msg.GetExtendedTextMessage() != nil:
		return msg.GetExtendedTextMessage().GetText(), model.MessageTypeText, "", ""
	case msg.GetImageMessage() != nil:
		img := msg.GetImageMessage()
		url := m.downloadMedia(evt, "image")
		return img.GetCaption(), model.MessageTypeImage, url, ""
	case msg.GetVideoMessage() != nil:
		vid := msg.GetVideoMessage()
		url := m.downloadMedia(evt, "video")
		return vid.GetCaption(), model.MessageTypeVideo, url, ""
	case msg.GetAudioMessage() != nil:
		url := m.downloadMedia(evt, "audio")
		return "", model.MessageTypeAudio, url, ""
	case msg.GetStickerMessage() != nil:
		url := m.downloadMedia(evt, "sticker")
		return "", model.MessageTypeSticker, url, ""
	case msg.GetDocumentMessage() != nil:
		doc := msg.GetDocumentMessage()
		url := m.downloadMedia(evt, "document")
		return doc.GetCaption(), model.MessageTypeDocument, url, doc.GetFileName()
	}
	return "", "", "", ""
}

// downloadMedia downloads an incoming media message and stores it locally,
// returning the public relative URL (empty on failure).
func (m *Manager) downloadMedia(evt *events.Message, kind string) string {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()
	if client == nil {
		return ""
	}

	data, err := client.DownloadAny(context.Background(), evt.Message)
	if err != nil {
		m.log.Warnf("download media: %v", err)
		return ""
	}

	switch kind {
	case "image":
		ext := "jpg"
		if mt := evt.Message.GetImageMessage().GetMimetype(); mt != "" {
			ext = mimeExt(mt, ext)
		}
		url, err := m.store.SaveImage(data, ext)
		if err != nil {
			m.log.Warnf("save image: %v", err)
			return ""
		}
		return url
	case "document":
		name := evt.Message.GetDocumentMessage().GetFileName()
		if name == "" {
			name = "file.bin"
		}
		url, err := m.store.SaveDocument(data, name)
		if err != nil {
			m.log.Warnf("save document: %v", err)
			return ""
		}
		return url
	case "video":
		ext := mimeExt(evt.Message.GetVideoMessage().GetMimetype(), "mp4")
		url, err := m.store.SaveMedia(data, ext)
		if err != nil {
			m.log.Warnf("save video: %v", err)
			return ""
		}
		return url
	case "audio":
		ext := mimeExt(evt.Message.GetAudioMessage().GetMimetype(), "ogg")
		url, err := m.store.SaveMedia(data, ext)
		if err != nil {
			m.log.Warnf("save audio: %v", err)
			return ""
		}
		return url
	case "sticker":
		ext := mimeExt(evt.Message.GetStickerMessage().GetMimetype(), "webp")
		url, err := m.store.SaveMedia(data, ext)
		if err != nil {
			m.log.Warnf("save sticker: %v", err)
			return ""
		}
		return url
	}
	return ""
}

// handleReceipt updates outgoing message delivery status.
func (m *Manager) handleReceipt(evt *events.Receipt) {
	var status string
	switch evt.Type {
	case types.ReceiptTypeDelivered:
		status = model.StatusDelivered
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		status = model.StatusRead
	default:
		return
	}
	for _, id := range evt.MessageIDs {
		_ = m.repo.UpdateMessageStatus(id, status)
		if m.webhook != nil {
			m.webhook.Emit(webhook.EventMessageAck, map[string]interface{}{
				"message_id": id,
				"status":     status,
			})
		}
	}
}

// emitNewMessage broadcasts a new message event to dashboards.
func (m *Manager) emitNewMessage(convID uint, contact *model.Contact, msg *model.Message) {
	m.hub.Emit("message.new", map[string]interface{}{
		"conversation_id": convID,
		"contact": map[string]interface{}{
			"id":    contact.ID,
			"name":  contact.Name,
			"phone": contact.Phone,
		},
		"message": msg,
	})
}

func mimeExt(mime, fallback string) string {
	if i := strings.Index(mime, "/"); i >= 0 {
		sub := mime[i+1:]
		if j := strings.Index(sub, ";"); j >= 0 {
			sub = sub[:j]
		}
		if sub == "jpeg" {
			return "jpg"
		}
		if sub != "" {
			return sub
		}
	}
	return fallback
}

// resolveSender determines the best phone number and routable chat JID for an
// incoming message. WhatsApp may address the sender by LID (hidden id on the
// "lid" server) rather than the phone number. We always resolve to the phone
// number (PN) so a contact has one stable identity regardless of addressing.
func (m *Manager) resolveSender(sender, senderAlt types.JID) (phone, chatJID string) {
	pn := sender
	if sender.Server == types.HiddenUserServer {
		switch {
		case !senderAlt.IsEmpty():
			pn = senderAlt
			m.storeLIDMapping(sender, senderAlt)
		default:
			// No alt provided — look up the PN from whatsmeow's LID map.
			if mapped := m.resolveLID(sender); mapped.Server != types.HiddenUserServer {
				pn = mapped
			}
		}
	}
	pnJID := types.NewJID(pn.User, pn.Server)
	return pn.User, pnJID.String()
}

// storeLIDMapping persists a LID↔PN mapping for future resolution.
func (m *Manager) storeLIDMapping(lid, pn types.JID) {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c != nil {
		c.StoreLIDPNMapping(context.Background(), lid, pn)
	}
}

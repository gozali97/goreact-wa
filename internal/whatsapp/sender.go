package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"wa-proxy/internal/logsvc"
	"wa-proxy/internal/model"
)

// SendResult is returned by the send helpers.
type SendResult struct {
	MessageID string
	Timestamp time.Time
}

// RateLimitError indicates a send was rejected by the rate limiter. RetryAfter
// is how long the caller should wait before retrying.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "rate limited: retry after " + e.RetryAfter.Round(time.Second).String()
}

// IsRateLimited reports whether err is a RateLimitError and returns it.
func IsRateLimited(err error) (*RateLimitError, bool) {
	rl, ok := err.(*RateLimitError)
	return rl, ok
}

// checkRate enforces the per-recipient / global send rate limit. Returns a
// *RateLimitError when the send must be delayed.
func (m *Manager) checkRate(phone string) error {
	if ok, retry := m.limiter.Acquire(phone); !ok {
		logsvc.Issue("whatsapp.ratelimit", "send throttled", map[string]interface{}{
			"phone": phone, "retry_after_sec": int(retry.Seconds()),
		})
		return &RateLimitError{RetryAfter: retry}
	}
	return nil
}

// SendText sends a plain text message to a phone number. It resolves the
// routable JID via the stored contact (handles LID-addressed chats correctly).
// Enforces the anti-ban rate limiter.
func (m *Manager) SendText(phone, text string) (*SendResult, error) {
	if err := m.checkRate(phone); err != nil {
		return nil, err
	}
	return m.sendTextCore(phone, text, true)
}

// SendTextBulk sends a text without the per-API rate limiter. Used by the
// broadcast worker, which applies its own randomized pacing between distinct
// recipients and must be able to retry without tripping the per-recipient lock.
func (m *Manager) SendTextBulk(phone, text string) (*SendResult, error) {
	return m.sendTextCore(phone, text, false)
}

// sendTextCore performs the actual send. humanize controls the anti-bot dance.
func (m *Manager) sendTextCore(phone, text string, humanize bool) (*SendResult, error) {
	client, err := m.requireClient()
	if err != nil {
		return nil, err
	}
	if humanize {
		m.humanize(phone)
	}
	to := m.resolveSendJID(phone)
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}
	resp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		m.log.Errorf("send text to %s (phone=%s): %v", to.String(), phone, err)
		logsvc.Errf("whatsapp.SendText", "failed to send text", err, map[string]interface{}{"phone": phone})
		return nil, fmt.Errorf("send text: %w", err)
	}
	m.recordOutgoing(phone, text, "", "", model.MessageTypeText, resp.ID, resp.Timestamp)
	logsvc.Success("whatsapp.SendText", "text message sent", map[string]interface{}{
		"phone": phone, "message_id": resp.ID,
	})
	return &SendResult{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// resolveSendJID returns the JID to deliver to for a phone number, preferring
// the routable JID stored on the contact. It also handles legacy contacts whose
// stored "phone" is actually a LID by resolving it to a phone number via
// whatsmeow's LID↔PN mapping.
func (m *Manager) resolveSendJID(phone string) types.JID {
	var jidStr, ph string
	if contact, err := m.repo.FindContactByPhone(phone); err == nil && contact != nil {
		jidStr = contact.JID
		ph = contact.Phone
	} else {
		ph = phone
	}

	// If we have a stored routable JID, use it (resolving LID → PN if needed).
	if jidStr != "" {
		if jid, err := types.ParseJID(jidStr); err == nil && !jid.IsEmpty() {
			return m.resolveLID(jid)
		}
	}
	// Otherwise treat the stored phone as a user id; it may be a real phone or a
	// legacy LID, so resolve via the mapping.
	return m.resolvePhoneOrLID(ph)
}

// resolveLID converts a LID JID to its phone-number JID using whatsmeow's
// stored mapping. Non-LID JIDs are returned unchanged.
func (m *Manager) resolveLID(jid types.JID) types.JID {
	if jid.Server != types.HiddenUserServer {
		return jid
	}
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil || c.Store == nil || c.Store.LIDs == nil {
		return jid
	}
	pn, err := c.Store.LIDs.GetPNForLID(context.Background(), jid)
	if err != nil || pn.IsEmpty() {
		return jid
	}
	return types.NewJID(pn.User, pn.Server)
}

// looksLikeLID reports whether a stored phone is likely a WhatsApp LID rather
// than a real phone number, by checking the LID mapping for a PN.
func (m *Manager) resolvePhoneOrLID(user string) types.JID {
	// Try as a normal phone-number JID first.
	pnJID := types.NewJID(user, types.DefaultUserServer)

	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil || c.Store == nil || c.Store.LIDs == nil {
		return pnJID
	}
	// If this user id is actually a known LID, prefer the mapped phone number.
	lidJID := types.NewJID(user, types.HiddenUserServer)
	if pn, err := c.Store.LIDs.GetPNForLID(context.Background(), lidJID); err == nil && !pn.IsEmpty() {
		return types.NewJID(pn.User, pn.Server)
	}
	return pnJID
}

// SendImage uploads and sends an image with an optional caption.
func (m *Manager) SendImage(phone, caption string, data []byte, mimeType string) (*SendResult, error) {
	client, err := m.requireClient()
	if err != nil {
		return nil, err
	}
	if err := m.checkRate(phone); err != nil {
		return nil, err
	}
	uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
		},
	}
	to := m.resolveSendJID(phone)
	resp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		return nil, fmt.Errorf("send image: %w", err)
	}
	// Persist a local copy so the dashboard can render the outgoing image.
	mediaURL, serr := m.store.SaveImage(data, mimeExt(mimeType, "jpg"))
	if serr != nil {
		m.log.Warnf("save outgoing image: %v", serr)
	}
	m.recordOutgoing(phone, caption, mediaURL, "", model.MessageTypeImage, resp.ID, resp.Timestamp)
	return &SendResult{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// SendDocument uploads and sends a document/file.
func (m *Manager) SendDocument(phone, caption, fileName string, data []byte, mimeType string) (*SendResult, error) {
	client, err := m.requireClient()
	if err != nil {
		return nil, err
	}
	if err := m.checkRate(phone); err != nil {
		return nil, err
	}
	uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("upload document: %w", err)
	}
	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			Caption:       proto.String(caption),
			FileName:      proto.String(fileName),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
		},
	}
	to := m.resolveSendJID(phone)
	resp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		return nil, fmt.Errorf("send document: %w", err)
	}
	mediaURL, serr := m.store.SaveDocument(data, fileName)
	if serr != nil {
		m.log.Warnf("save outgoing document: %v", serr)
	}
	m.recordOutgoing(phone, caption, mediaURL, fileName, model.MessageTypeDocument, resp.ID, resp.Timestamp)
	return &SendResult{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// URLButton describes a "call-to-action" URL button (e.g. "Lihat Hasil").
type URLButton struct {
	Text string // button label
	URL  string // destination URL
}

// SendButtons sends a message with call-to-action links. WhatsApp does not
// reliably render native template/interactive buttons from unofficial
// (whatsmeow) clients — they either get dropped or show "couldn't load" — so
// this delivers a normal text message whose links are clickable, with a rich
// link-preview card for the first URL. Reliable + tappable. Rate-limited.
func (m *Manager) SendButtons(phone, text, footer string, buttons []URLButton) (*SendResult, error) {
	client, err := m.requireClient()
	if err != nil {
		return nil, err
	}
	if err := m.checkRate(phone); err != nil {
		return nil, err
	}
	if len(buttons) == 0 {
		return nil, fmt.Errorf("at least one button is required")
	}

	// Compose the visible body: text, then each "label:\nurl" line, then footer.
	body := text
	for _, b := range buttons {
		body += "\n\n" + b.Text + ":\n" + b.URL
	}
	if footer != "" {
		body += "\n\n" + footer
	}

	// Rich link preview for the first button's URL (tappable card).
	first := buttons[0]
	ext := &waE2E.ExtendedTextMessage{
		Text:        proto.String(body),
		MatchedText: proto.String(first.URL),
		Title:       proto.String(first.Text),
		PreviewType: waE2E.ExtendedTextMessage_NONE.Enum(),
	}
	if d := firstLine(text); d != "" {
		ext.Description = proto.String(d)
	}

	msg := &waE2E.Message{ExtendedTextMessage: ext}

	to := m.resolveSendJID(phone)
	resp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		logsvc.Errf("whatsapp.SendButtons", "failed to send link message", err, map[string]interface{}{"phone": phone})
		return nil, fmt.Errorf("send buttons: %w", err)
	}
	m.recordOutgoing(phone, body, "", "", model.MessageTypeText, resp.ID, resp.Timestamp)
	logsvc.Success("whatsapp.SendButtons", "link message sent", map[string]interface{}{
		"phone": phone, "message_id": resp.ID, "buttons": len(buttons),
	})
	return &SendResult{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// firstLine returns the first non-empty line of s (used as a link description).
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return s
}

// recordOutgoing persists an outgoing message and updates the conversation.
func (m *Manager) recordOutgoing(phone, content, mediaURL, fileName, msgType, messageID string, ts time.Time) {
	// Try to attach a known display name (from whatsmeow's contact store).
	contact, err := m.repo.UpsertContact(phone, m.LookupName(phone), "")
	if err != nil {
		m.log.Warnf("record outgoing contact: %v", err)
		return
	}
	conv, err := m.repo.GetOrCreateConversation(contact.ID)
	if err != nil {
		m.log.Warnf("record outgoing conversation: %v", err)
		return
	}
	sentAt := ts
	msg := &model.Message{
		ConversationID: conv.ID,
		MessageID:      messageID,
		Direction:      model.DirectionOutgoing,
		MessageType:    msgType,
		Content:        content,
		MediaURL:       mediaURL,
		FileName:       fileName,
		Status:         model.StatusSent,
		SentAt:         &sentAt,
	}
	if err := m.repo.CreateMessage(msg); err != nil {
		m.log.Warnf("record outgoing message: %v", err)
		return
	}
	preview := content
	if preview == "" {
		preview = "[" + msgType + "]"
	}
	_ = m.repo.TouchConversation(conv.ID, preview, false)
	m.emitNewMessage(conv.ID, contact, msg)
}

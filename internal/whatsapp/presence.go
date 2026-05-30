package whatsapp

import (
	"context"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// CheckOnWhatsApp reports, for each supplied phone number, whether it is
// registered on WhatsApp (WAHA-style check-exists). Helps avoid sending to
// invalid numbers, which is a common ban trigger.
func (m *Manager) CheckOnWhatsApp(phones []string) ([]types.IsOnWhatsAppResponse, error) {
	client, err := m.requireClient()
	if err != nil {
		return nil, err
	}
	return client.IsOnWhatsApp(context.Background(), phones)
}

// IsRegistered is a convenience wrapper returning whether a single phone is on
// WhatsApp.
func (m *Manager) IsRegistered(phone string) (bool, error) {
	resp, err := m.CheckOnWhatsApp([]string{phone})
	if err != nil {
		return false, err
	}
	if len(resp) == 0 {
		return false, nil
	}
	return resp[0].IsIn, nil
}

// SendTyping toggles the "typing…" presence in a chat (WAHA startTyping /
// stopTyping). state true = composing, false = paused.
func (m *Manager) SendTyping(phone string, typing bool) error {
	client, err := m.requireClient()
	if err != nil {
		return err
	}
	state := types.ChatPresencePaused
	if typing {
		state = types.ChatPresenceComposing
	}
	to := parseJID(phone)
	return client.SendChatPresence(context.Background(), to, state, types.ChatPresenceMediaText)
}

// MarkSeen sends read receipts for the given message IDs in a chat (WAHA
// sendSeen). Required before replying to avoid being flagged as a bot.
func (m *Manager) MarkSeen(phone string, messageIDs []string) error {
	client, err := m.requireClient()
	if err != nil {
		return err
	}
	if len(messageIDs) == 0 {
		return nil
	}
	chat := parseJID(phone)
	ids := make([]types.MessageID, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = types.MessageID(id)
	}
	return client.MarkRead(context.Background(), ids, time.Now(), chat, chat)
}

// humanize performs the WAHA anti-blocking dance before a send: mark unread
// messages seen, show typing for a short random interval, then stop. Errors are
// non-fatal (best effort).
func (m *Manager) humanize(phone string) {
	if !m.cfg.HumanizeSend {
		return
	}
	// Mark recent inbound messages as seen.
	if ids := m.recentInboundIDs(phone); len(ids) > 0 {
		_ = m.MarkSeen(phone, ids)
	}
	// Show typing for 1–3 seconds.
	_ = m.SendTyping(phone, true)
	delay := 1000 + rand.Intn(2000)
	time.Sleep(time.Duration(delay) * time.Millisecond)
	_ = m.SendTyping(phone, false)
}

// LookupName tries to find a display name for a phone number from whatsmeow's
// contact store (push name, full name, or business name). Returns "" if none.
func (m *Manager) LookupName(phone string) string {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil || c.Store == nil || c.Store.Contacts == nil {
		return ""
	}
	ctx := context.Background()
	pnJID := types.NewJID(phone, types.DefaultUserServer)
	info, err := c.Store.Contacts.GetContact(ctx, pnJID)
	if err == nil && info.Found {
		if n := pickName(info); n != "" {
			return n
		}
	}
	// Also try the LID form (the contact may be cached under its LID).
	if c.Store.LIDs != nil {
		if lid, e := c.Store.LIDs.GetLIDForPN(ctx, pnJID); e == nil && !lid.IsEmpty() {
			if info, e := c.Store.Contacts.GetContact(ctx, lid); e == nil && info.Found {
				return pickName(info)
			}
		}
	}
	return ""
}

func pickName(info types.ContactInfo) string {
	switch {
	case info.FullName != "":
		return info.FullName
	case info.PushName != "":
		return info.PushName
	case info.BusinessName != "":
		return info.BusinessName
	case info.FirstName != "":
		return info.FirstName
	}
	return ""
}

// recentInboundIDs returns up to a few recent incoming message IDs for a phone,
// used to mark them seen before replying.
func (m *Manager) recentInboundIDs(phone string) []string {
	contact, err := m.repo.UpsertContact(phone, "", "")
	if err != nil {
		return nil
	}
	conv, err := m.repo.GetOrCreateConversation(contact.ID)
	if err != nil {
		return nil
	}
	msgs, err := m.repo.RecentIncomingMessageIDs(conv.ID, 20)
	if err != nil {
		return nil
	}
	return msgs
}

// BackfillLIDContacts fixes legacy contacts whose phone was stored as a LID
// (created before LID→PN resolution existed). For each such contact it looks up
// the real phone number and either renames the contact or merges it into the
// existing phone-keyed contact. Safe to run on every startup (no-op once clean).
func (m *Manager) BackfillLIDContacts() {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil || c.Store == nil || c.Store.LIDs == nil {
		return
	}

	contacts, err := m.repo.ListAllContacts()
	if err != nil {
		m.log.Warnf("backfill: list contacts: %v", err)
		return
	}
	ctx := context.Background()
	for i := range contacts {
		ct := contacts[i]
		// Heuristic: a LID is a large numeric id that resolves to a PN in the
		// whatsmeow LID map. Real phone numbers won't be found as a LID key.
		lidJID := types.NewJID(ct.Phone, types.HiddenUserServer)
		pn, err := c.Store.LIDs.GetPNForLID(ctx, lidJID)
		if err != nil || pn.IsEmpty() || pn.User == ct.Phone {
			continue // not a LID-keyed contact
		}

		// This contact is keyed by a LID; pn.User is the real phone.
		if existing, _ := m.repo.FindContactByPhone(pn.User); existing != nil && existing.ID != ct.ID {
			// Merge the LID contact into the existing phone contact.
			if err := m.repo.MergeContacts(existing.ID, ct.ID); err != nil {
				m.log.Warnf("backfill: merge %d→%d: %v", ct.ID, existing.ID, err)
			} else {
				m.log.Infof("backfill: merged LID contact %s into %s", ct.Phone, pn.User)
			}
		} else {
			// No phone contact yet — rename this one to the real phone + JID.
			if err := m.repo.RenameContactPhone(ct.ID, pn.User, types.NewJID(pn.User, pn.Server).String()); err != nil {
				m.log.Warnf("backfill: rename %d: %v", ct.ID, err)
			} else {
				m.log.Infof("backfill: renamed LID contact %s → %s", ct.Phone, pn.User)
			}
		}
	}

	// Second pass: fill in any missing contact names from whatsmeow's store.
	m.BackfillContactNames()
}

// BackfillContactNames fills in display names for contacts that have none,
// using whatsmeow's cached contact store. Runs in the background.
func (m *Manager) BackfillContactNames() {
	contacts, err := m.repo.ListAllContacts()
	if err != nil {
		return
	}
	for i := range contacts {
		ct := contacts[i]
		if ct.Name != "" {
			continue
		}
		if name := m.LookupName(ct.Phone); name != "" {
			if err := m.repo.UpdateContactName(ct.ID, name); err != nil {
				m.log.Warnf("backfill name %d: %v", ct.ID, err)
			} else {
				m.log.Infof("backfill: set name %q for %s", name, ct.Phone)
			}
		}
	}
}

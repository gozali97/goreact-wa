package repository

import (
	"time"

	"gorm.io/gorm"

	"wa-proxy/internal/model"
)

// Repository provides data-access methods for all application models.
type Repository struct {
	db *gorm.DB
}

// New creates a Repository backed by the given GORM connection.
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// DB exposes the underlying GORM handle for advanced queries.
func (r *Repository) DB() *gorm.DB { return r.db }

// ─── Sessions ─────────────────────────────────────────────

// UpsertSession creates or updates the single session row by JID.
func (r *Repository) UpsertSession(s *model.Session) error {
	var existing model.Session
	err := r.db.Where("jid = ?", s.JID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(s).Error
	}
	if err != nil {
		return err
	}
	s.ID = existing.ID
	return r.db.Model(&existing).Updates(s).Error
}

// GetSession returns the most recent session row, if any.
func (r *Repository) GetSession() (*model.Session, error) {
	var s model.Session
	err := r.db.Order("updated_at desc").First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &s, err
}

// UpdateSessionStatus updates status (and last seen) for all session rows.
func (r *Repository) UpdateSessionStatus(status string) error {
	now := time.Now()
	return r.db.Model(&model.Session{}).
		Where("1 = 1").
		Updates(map[string]interface{}{"status": status, "last_seen": now}).Error
}

// ClearSessions removes all session metadata rows (on logout).
func (r *Repository) ClearSessions() error {
	return r.db.Where("1 = 1").Delete(&model.Session{}).Error
}

// ─── Contacts ─────────────────────────────────────────────

// UpsertContact finds a contact by phone or creates it, updating name/jid/last
// message time.
func (r *Repository) UpsertContact(phone, name, jid string) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("phone = ?", phone).First(&c).Error
	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		c = model.Contact{Phone: phone, Name: name, JID: jid, LastMessageAt: &now}
		if err := r.db.Create(&c).Error; err != nil {
			return nil, err
		}
		return &c, nil
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{"last_message_at": now}
	if name != "" && c.Name == "" {
		updates["name"] = name
	}
	if jid != "" && c.JID != jid {
		updates["jid"] = jid
	}
	if err := r.db.Model(&c).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ListContacts returns contacts, optionally filtered by a search term.
func (r *Repository) ListContacts(search string) ([]model.Contact, error) {
	var contacts []model.Contact
	q := r.db.Order("last_message_at desc nulls last")
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR phone ILIKE ?", like, like)
	}
	err := q.Find(&contacts).Error
	return contacts, err
}

// GetContact returns a single contact by id.
func (r *Repository) GetContact(id uint) (*model.Contact, error) {
	var c model.Contact
	err := r.db.First(&c, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &c, err
}

// FindContactByPhone returns a contact by phone, or nil if not found.
func (r *Repository) FindContactByPhone(phone string) (*model.Contact, error) {
	var c model.Contact
	err := r.db.Where("phone = ?", phone).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &c, err
}

// ListAllContacts returns every contact (used for maintenance/backfill).
func (r *Repository) ListAllContacts() ([]model.Contact, error) {
	var contacts []model.Contact
	err := r.db.Find(&contacts).Error
	return contacts, err
}

// RenameContactPhone updates a contact's phone (and JID) — used by the LID
// backfill when no phone-keyed duplicate exists.
func (r *Repository) RenameContactPhone(id uint, phone, jid string) error {
	return r.db.Model(&model.Contact{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"phone": phone, "jid": jid}).Error
}

// UpdateContactName sets a contact's display name.
func (r *Repository) UpdateContactName(id uint, name string) error {
	return r.db.Model(&model.Contact{}).
		Where("id = ?", id).
		Update("name", name).Error
}

// MergeContacts moves a duplicate contact's conversation/messages into the
// canonical contact, then deletes the duplicate. Used to fix legacy contacts
// that were keyed by LID instead of phone number. If the canonical contact has
// no conversation, the duplicate's conversation is re-pointed; otherwise the
// duplicate's messages are moved into the canonical conversation.
func (r *Repository) MergeContacts(canonicalID, duplicateID uint) error {
	if canonicalID == duplicateID {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var dupConv model.Conversation
		err := tx.Where("contact_id = ?", duplicateID).First(&dupConv).Error
		if err == gorm.ErrRecordNotFound {
			// Nothing to move; just delete the duplicate contact.
			return tx.Delete(&model.Contact{}, duplicateID).Error
		}
		if err != nil {
			return err
		}

		var canonConv model.Conversation
		err = tx.Where("contact_id = ?", canonicalID).First(&canonConv).Error
		if err == gorm.ErrRecordNotFound {
			// Re-point the duplicate's conversation to the canonical contact.
			if e := tx.Model(&model.Conversation{}).
				Where("id = ?", dupConv.ID).
				Update("contact_id", canonicalID).Error; e != nil {
				return e
			}
		} else if err != nil {
			return err
		} else {
			// Move messages into the canonical conversation, then drop the dup.
			if e := tx.Model(&model.Message{}).
				Where("conversation_id = ?", dupConv.ID).
				Update("conversation_id", canonConv.ID).Error; e != nil {
				return e
			}
			// Carry over the duplicate's last-message preview if it is newer.
			if dupConv.LastMessageAt != nil &&
				(canonConv.LastMessageAt == nil || dupConv.LastMessageAt.After(*canonConv.LastMessageAt)) {
				if e := tx.Model(&model.Conversation{}).Where("id = ?", canonConv.ID).
					Updates(map[string]interface{}{
						"last_message":    dupConv.LastMessage,
						"last_message_at": dupConv.LastMessageAt,
					}).Error; e != nil {
					return e
				}
			}
			if e := tx.Delete(&model.Conversation{}, dupConv.ID).Error; e != nil {
				return e
			}
		}

		// Preserve the duplicate's name if the canonical one is missing it.
		var dup, canon model.Contact
		if e := tx.First(&dup, duplicateID).Error; e == nil {
			if e := tx.First(&canon, canonicalID).Error; e == nil {
				if canon.Name == "" && dup.Name != "" {
					_ = tx.Model(&model.Contact{}).Where("id = ?", canonicalID).
						Update("name", dup.Name).Error
				}
			}
		}

		return tx.Delete(&model.Contact{}, duplicateID).Error
	})
}

// CountContacts returns the total number of contacts.
func (r *Repository) CountContacts() (int64, error) {
	var n int64
	err := r.db.Model(&model.Contact{}).Count(&n).Error
	return n, err
}

// AllContactPhones returns every known contact phone number.
func (r *Repository) AllContactPhones() ([]string, error) {
	var phones []string
	err := r.db.Model(&model.Contact{}).
		Where("phone <> ''").
		Order("last_message_at desc nulls last").
		Pluck("phone", &phones).Error
	return phones, err
}

// ─── Conversations ────────────────────────────────────────

// GetOrCreateConversation ensures a conversation exists for a contact.
func (r *Repository) GetOrCreateConversation(contactID uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.Where("contact_id = ?", contactID).First(&conv).Error
	if err == gorm.ErrRecordNotFound {
		conv = model.Conversation{ContactID: contactID}
		if err := r.db.Create(&conv).Error; err != nil {
			return nil, err
		}
		return &conv, nil
	}
	return &conv, err
}

// TouchConversation updates last message preview/time and unread count.
func (r *Repository) TouchConversation(convID uint, preview string, incUnread bool) error {
	now := time.Now()
	updates := map[string]interface{}{
		"last_message":    preview,
		"last_message_at": now,
	}
	if incUnread {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}
	return r.db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(updates).Error
}

// MarkConversationRead resets the unread counter.
func (r *Repository) MarkConversationRead(convID uint) error {
	return r.db.Model(&model.Conversation{}).Where("id = ?", convID).
		Update("unread_count", 0).Error
}

// ListConversations returns conversations with contact preloaded, newest first.
func (r *Repository) ListConversations() ([]model.Conversation, error) {
	var convs []model.Conversation
	err := r.db.Preload("Contact").
		Order("last_message_at desc nulls last").
		Find(&convs).Error
	return convs, err
}

// GetConversation returns a conversation with its contact.
func (r *Repository) GetConversation(id uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.Preload("Contact").First(&conv, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &conv, err
}

// GetConversationByPhone returns the conversation (with contact) for a phone
// number, or nil if none exists yet.
func (r *Repository) GetConversationByPhone(phone string) (*model.Conversation, error) {
	var contact model.Contact
	err := r.db.Where("phone = ?", phone).First(&contact).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var conv model.Conversation
	err = r.db.Preload("Contact").Where("contact_id = ?", contact.ID).First(&conv).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &conv, err
}

// CountConversations returns the total number of conversations.
func (r *Repository) CountConversations() (int64, error) {
	var n int64
	err := r.db.Model(&model.Conversation{}).Count(&n).Error
	return n, err
}

// ─── Messages ─────────────────────────────────────────────

// CreateMessage inserts a message row.
func (r *Repository) CreateMessage(m *model.Message) error {
	return r.db.Create(m).Error
}

// ListMessages returns messages for a conversation in chronological order.
func (r *Repository) ListMessages(convID uint, limit int) ([]model.Message, error) {
	var msgs []model.Message
	q := r.db.Where("conversation_id = ?", convID).Order("created_at asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&msgs).Error
	return msgs, err
}

// UpdateMessageStatus updates the delivery status by whatsmeow message id.
func (r *Repository) UpdateMessageStatus(messageID, status string) error {
	return r.db.Model(&model.Message{}).
		Where("message_id = ?", messageID).
		Update("status", status).Error
}

// RecentIncomingMessageIDs returns the newest incoming message IDs (whatsmeow
// IDs) for a conversation, used to mark them seen before replying.
func (r *Repository) RecentIncomingMessageIDs(convID uint, limit int) ([]string, error) {
	var ids []string
	q := r.db.Model(&model.Message{}).
		Where("conversation_id = ? AND direction = ? AND message_id <> ''", convID, model.DirectionIncoming).
		Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Pluck("message_id", &ids).Error
	return ids, err
}

// ─── Broadcasts ───────────────────────────────────────────

// CreateBroadcast inserts a broadcast plus its per-recipient details.
func (r *Repository) CreateBroadcast(b *model.Broadcast, phones []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(b).Error; err != nil {
			return err
		}
		for _, p := range phones {
			d := model.BroadcastDetail{BroadcastID: b.ID, Phone: p, Status: model.BroadcastPending}
			if err := tx.Create(&d).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListBroadcasts returns broadcasts newest first.
func (r *Repository) ListBroadcasts() ([]model.Broadcast, error) {
	var bs []model.Broadcast
	err := r.db.Order("created_at desc").Find(&bs).Error
	return bs, err
}

// GetBroadcast returns a single broadcast by id.
func (r *Repository) GetBroadcast(id uint) (*model.Broadcast, error) {
	var b model.Broadcast
	err := r.db.First(&b, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &b, err
}

// ListBroadcastDetails returns recipients for a broadcast.
func (r *Repository) ListBroadcastDetails(broadcastID uint) ([]model.BroadcastDetail, error) {
	var ds []model.BroadcastDetail
	err := r.db.Where("broadcast_id = ?", broadcastID).Find(&ds).Error
	return ds, err
}

// PendingBroadcasts returns broadcasts that still need processing.
func (r *Repository) PendingBroadcasts() ([]model.Broadcast, error) {
	var bs []model.Broadcast
	err := r.db.Where("status IN ?", []string{model.BroadcastPending, model.BroadcastSending}).
		Find(&bs).Error
	return bs, err
}

// PendingBroadcastDetails returns unfinished recipients for a broadcast.
func (r *Repository) PendingBroadcastDetails(broadcastID uint) ([]model.BroadcastDetail, error) {
	var ds []model.BroadcastDetail
	err := r.db.Where("broadcast_id = ? AND status IN ?", broadcastID,
		[]string{model.BroadcastPending, model.BroadcastFailed}).Find(&ds).Error
	return ds, err
}

// UpdateBroadcastDetail saves a detail row.
func (r *Repository) UpdateBroadcastDetail(d *model.BroadcastDetail) error {
	return r.db.Save(d).Error
}

// UpdateBroadcast saves a broadcast row.
func (r *Repository) UpdateBroadcast(b *model.Broadcast) error {
	return r.db.Save(b).Error
}

// CountBroadcasts returns the total number of broadcasts.
func (r *Repository) CountBroadcasts() (int64, error) {
	var n int64
	err := r.db.Model(&model.Broadcast{}).Count(&n).Error
	return n, err
}

// ─── Settings ─────────────────────────────────────────────

// GetSetting returns the value for a key, or "" if absent.
func (r *Repository) GetSetting(key string) (string, error) {
	var s model.Setting
	err := r.db.Where("key = ?", key).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// SetSetting upserts a key/value setting.
func (r *Repository) SetSetting(key, value string) error {
	return r.db.Save(&model.Setting{Key: key, Value: value}).Error
}

// DeleteSetting removes a setting by key (no error if absent).
func (r *Repository) DeleteSetting(key string) error {
	return r.db.Where("key = ?", key).Delete(&model.Setting{}).Error
}

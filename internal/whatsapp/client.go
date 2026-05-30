package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wa-proxy/internal/apikey"
	"wa-proxy/internal/config"
	"wa-proxy/internal/logsvc"
	"wa-proxy/internal/model"
	"wa-proxy/internal/ratelimit"
	"wa-proxy/internal/repository"
	"wa-proxy/internal/storage"
	"wa-proxy/internal/webhook"
	"wa-proxy/internal/ws"
)

// Manager owns the whatsmeow client/session lifecycle and bridges WhatsApp
// events to the database and WebSocket hub.
type Manager struct {
	cfg       *config.Config
	repo      *repository.Repository
	hub       *ws.Hub
	store     *storage.Store
	webhook   *webhook.Dispatcher
	limiter   *ratelimit.Limiter
	keys      *apikey.Service
	container *sqlstore.Container
	log       waLog.Logger

	mu       sync.RWMutex
	client   *whatsmeow.Client
	status   string
	lastQR   string
	qrCancel context.CancelFunc
}

// NewManager builds a Manager and initializes the whatsmeow sqlstore container
// against the same PostgreSQL database used by the app.
func NewManager(cfg *config.Config, repo *repository.Repository, hub *ws.Hub, store *storage.Store, wh *webhook.Dispatcher, keys *apikey.Service) (*Manager, error) {
	dbLog := waLog.Stdout("WADB", "WARN", true)
	// Use the pgx stdlib driver (registered as "pgx"); whatsmeow's dbutil maps
	// the "pgx" dialect to Postgres.
	container, err := sqlstore.New(context.Background(), "pgx", cfg.DSN(), dbLog)
	if err != nil {
		return nil, fmt.Errorf("whatsapp sqlstore: %w", err)
	}

	m := &Manager{
		cfg:     cfg,
		repo:    repo,
		hub:     hub,
		store:   store,
		webhook: wh,
		limiter: ratelimit.New(
			cfg.RateLimitEnabled,
			time.Duration(cfg.RateMinIntervalSec)*time.Second,
			time.Duration(cfg.RateGlobalIntervalSec)*time.Second,
		),
		keys:      keys,
		container: container,
		log:       waLog.Stdout("WA", "INFO", true),
		status:    model.DeviceDisconnected,
	}
	return m, nil
}

// Status returns the current connection status.
func (m *Manager) Status() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// LastQR returns the most recently generated QR code string (may be empty).
func (m *Manager) LastQR() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastQR
}

func (m *Manager) setStatus(status string) {
	m.mu.Lock()
	m.status = status
	if status != model.DeviceConnecting {
		m.lastQR = ""
	}
	m.mu.Unlock()

	_ = m.repo.UpdateSessionStatus(status)
	m.hub.Emit(ws.EventDeviceStatus, map[string]interface{}{"status": status})
	logsvc.Info("whatsapp.status", "device status: "+status, map[string]interface{}{"status": status})
	if m.webhook != nil {
		m.webhook.Emit(webhook.EventSessionStatus, map[string]interface{}{"status": status})
	}
}

// getDeviceStore returns an existing device or a new one.
func (m *Manager) getDeviceStore(ctx context.Context) (*store.Device, error) {
	device, err := m.container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// Init loads an existing session (if any) and connects automatically.
func (m *Manager) Init() {
	ctx := context.Background()
	device, err := m.getDeviceStore(ctx)
	if err != nil {
		m.log.Errorf("init get device: %v", err)
		return
	}
	if device.ID == nil {
		// No stored session yet; wait for user to Connect via dashboard.
		m.setStatus(model.DeviceLoggedOut)
		return
	}
	m.startClient(device)
	m.setStatus(model.DeviceConnecting)
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if err := c.Connect(); err != nil {
		m.log.Errorf("init connect: %v", err)
		m.setStatus(model.DeviceDisconnected)
	}
}

// startClient creates a fresh whatsmeow client for the device and wires
// handlers. Any existing client is disposed first so repeated Connect calls
// never reuse a stale (e.g. timed-out QR) client.
func (m *Manager) startClient(device *store.Device) {
	m.disposeClient()
	m.mu.Lock()
	client := whatsmeow.NewClient(device, m.log)
	client.EnableAutoReconnect = true
	client.AddEventHandler(m.eventHandler)
	m.client = client
	m.mu.Unlock()
}

// disposeClient cancels any in-flight QR flow and disconnects/clears the
// current client. Safe to call when there is no client.
func (m *Manager) disposeClient() {
	m.mu.Lock()
	client := m.client
	cancel := m.qrCancel
	m.client = nil
	m.qrCancel = nil
	m.lastQR = ""
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if client != nil {
		client.Disconnect()
	}
}

// Connect initiates pairing. If a session already exists it reconnects;
// otherwise it starts a fresh QR login flow.
func (m *Manager) Connect() error {
	ctx := context.Background()

	// If we already have a logged-in client, just make sure it's connected.
	m.mu.RLock()
	existing := m.client
	m.mu.RUnlock()
	if existing != nil && existing.IsLoggedIn() {
		if existing.IsConnected() {
			m.setStatus(model.DeviceConnected)
			return nil
		}
		m.setStatus(model.DeviceConnecting)
		return existing.Connect()
	}

	device, err := m.getDeviceStore(ctx)
	if err != nil {
		return err
	}

	// Paired device persisted in the store → reconnect (no QR needed).
	if device.ID != nil {
		m.startClient(device)
		m.setStatus(model.DeviceConnecting)
		m.mu.RLock()
		c := m.client
		m.mu.RUnlock()
		return c.Connect()
	}

	// New device → QR login. Always start from a clean client + device.
	device = m.container.NewDevice()
	m.startClient(device)

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	qrCtx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.qrCancel = cancel
	m.mu.Unlock()

	// GetQRChannel must be called before Connect and before the socket is up.
	qrChan, err := client.GetQRChannel(qrCtx)
	if err != nil {
		cancel()
		return fmt.Errorf("get qr channel: %w", err)
	}
	m.setStatus(model.DeviceConnecting)
	if err := client.Connect(); err != nil {
		cancel()
		m.setStatus(model.DeviceDisconnected)
		return fmt.Errorf("connect for qr: %w", err)
	}

	go m.consumeQR(qrChan)
	return nil
}

// consumeQR forwards QR codes / pairing results to the dashboard.
func (m *Manager) consumeQR(qrChan <-chan whatsmeow.QRChannelItem) {
	for item := range qrChan {
		switch item.Event {
		case "code":
			m.mu.Lock()
			m.lastQR = item.Code
			m.mu.Unlock()
			m.hub.Emit(ws.EventQR, map[string]interface{}{"code": item.Code})
		case "success":
			m.hub.Emit(ws.EventQR, map[string]interface{}{"code": "", "paired": true})
		case "timeout":
			m.hub.Emit(ws.EventQR, map[string]interface{}{"code": "", "timeout": true})
			m.setStatus(model.DeviceDisconnected)
		default:
			if item.Error != nil {
				m.log.Warnf("qr event error: %v", item.Error)
			}
		}
	}
}

// Logout disconnects and removes the stored session.
func (m *Manager) Logout() error {
	m.mu.Lock()
	client := m.client
	cancel := m.qrCancel
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if client == nil {
		m.setStatus(model.DeviceLoggedOut)
		_ = m.repo.ClearSessions()
		_ = m.keys.Clear()
		return nil
	}

	ctx := context.Background()
	if client.IsLoggedIn() {
		if err := client.Logout(ctx); err != nil {
			m.log.Warnf("logout: %v", err)
			client.Disconnect()
		}
	} else {
		client.Disconnect()
	}

	m.mu.Lock()
	m.client = nil
	m.mu.Unlock()

	_ = m.repo.ClearSessions()
	_ = m.keys.Clear()
	m.setStatus(model.DeviceLoggedOut)
	return nil
}

// client access helper that errors if not ready to send.
func (m *Manager) requireClient() (*whatsmeow.Client, error) {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil || !c.IsConnected() || !c.IsLoggedIn() {
		return nil, fmt.Errorf("whatsapp not connected")
	}
	return c, nil
}

// IsConnected reports whether the device is fully usable: the socket is
// connected AND the account is logged in (paired). During QR pairing the socket
// connects before login, so we must check IsLoggedIn too — otherwise the UI
// would show "connected" before the QR is even scanned.
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	return c != nil && c.IsConnected() && c.IsLoggedIn()
}

// HasSession reports whether a paired device exists in the store.
func (m *Manager) HasSession() bool {
	device, err := m.getDeviceStore(context.Background())
	if err != nil {
		return false
	}
	return device.ID != nil
}

// Reconnect attempts to (re)establish the connection for an existing session.
func (m *Manager) Reconnect() error {
	device, err := m.getDeviceStore(context.Background())
	if err != nil {
		return err
	}
	if device.ID == nil {
		return fmt.Errorf("no session to reconnect")
	}
	m.startClient(device)
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil {
		return fmt.Errorf("client not initialized")
	}
	if c.IsConnected() {
		return nil
	}
	return c.Connect()
}

// parseJID converts a plain phone number to a WhatsApp user JID.
func parseJID(phone string) types.JID {
	return types.NewJID(phone, types.DefaultUserServer)
}

// persistSession stores account metadata after a successful pairing.
func (m *Manager) persistSession() {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c == nil || c.Store.ID == nil {
		return
	}
	jid := c.Store.ID
	now := time.Now()
	s := &model.Session{
		Phone:    jid.User,
		PushName: c.Store.PushName,
		Status:   model.DeviceConnected,
		JID:      jid.String(),
		LastSeen: &now,
	}
	if err := m.repo.UpsertSession(s); err != nil {
		m.log.Warnf("persist session: %v", err)
	}
}

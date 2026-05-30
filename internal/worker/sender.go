package worker

import (
	"log"

	"wa-proxy/internal/whatsapp"
)

// SendJob is a unit of outgoing-message work processed asynchronously.
type SendJob struct {
	Type     string // "text" | "image" | "document"
	Phone    string
	Message  string
	Caption  string
	FileName string
	MimeType string
	Data     []byte

	// Result channel (optional) lets synchronous callers wait for the outcome.
	Result chan SendJobResult
}

// SendJobResult carries the outcome of a SendJob.
type SendJobResult struct {
	MessageID string
	Err       error
}

// MessageSender is a background worker that serializes outgoing messages so the
// WhatsApp client is never hit concurrently from many HTTP requests.
type MessageSender struct {
	wa    *whatsapp.Manager
	queue chan SendJob
}

// NewMessageSender creates a MessageSender with a buffered queue.
func NewMessageSender(wa *whatsapp.Manager) *MessageSender {
	return &MessageSender{
		wa:    wa,
		queue: make(chan SendJob, 256),
	}
}

// Start launches the worker loop in a goroutine.
func (s *MessageSender) Start() {
	go s.run()
}

// Enqueue submits a job for asynchronous processing.
func (s *MessageSender) Enqueue(job SendJob) {
	s.queue <- job
}

func (s *MessageSender) run() {
	for job := range s.queue {
		res := s.process(job)
		if job.Result != nil {
			job.Result <- res
		} else if res.Err != nil {
			log.Printf("worker: send %s to %s failed: %v", job.Type, job.Phone, res.Err)
		}
	}
}

func (s *MessageSender) process(job SendJob) SendJobResult {
	var (
		id  string
		err error
	)
	switch job.Type {
	case "text":
		r, e := s.wa.SendText(job.Phone, job.Message)
		if e == nil {
			id = r.MessageID
		}
		err = e
	case "image":
		r, e := s.wa.SendImage(job.Phone, job.Caption, job.Data, job.MimeType)
		if e == nil {
			id = r.MessageID
		}
		err = e
	case "document":
		r, e := s.wa.SendDocument(job.Phone, job.Caption, job.FileName, job.Data, job.MimeType)
		if e == nil {
			id = r.MessageID
		}
		err = e
	}
	return SendJobResult{MessageID: id, Err: err}
}

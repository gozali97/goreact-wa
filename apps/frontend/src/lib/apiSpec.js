// Structured API specification that drives the interactive API Docs page.
// Each module groups endpoints. Each endpoint describes its method, path,
// auth, parameters, request body, samples and response examples, plus a `try`
// descriptor that powers the live request runner.
//
// `try.fields` define the inputs rendered in the "Try it" panel:
//   { name, in: 'path'|'query'|'body'|'form', type: 'text'|'textarea'|'file'|'number'|'json',
//     required, placeholder, default, help }

export const BASE_URL = '/api'

export const API_SPEC = [
  {
    id: 'auth',
    name: 'Authentication',
    icon: 'icon-[tabler--lock]',
    description:
      'All endpoints require authentication. Integration endpoints (messaging, presence) accept an API key via the X-Api-Key header; the key is generated and managed on the Device page (stored in the database, with copy/regenerate). Dashboard Basic Auth also works as a fallback. The interactive "Try it" runner below uses your current dashboard session automatically.',
    endpoints: [
      {
        id: 'auth-check',
        method: 'GET',
        path: '/api/auth/check',
        title: 'Verify credentials',
        description: 'Returns 200 when the supplied credentials are valid. Used by the dashboard login screen.',
        auth: 'Basic / X-Api-Key',
        responses: [
          { status: 200, body: { success: true } },
          { status: 401, body: { success: false, error: 'unauthorized' } },
        ],
        try: { method: 'GET', path: '/auth/check', fields: [] },
      },
    ],
  },

  {
    id: 'device',
    name: 'Device & Session',
    icon: 'icon-[tabler--device-mobile]',
    description: 'Manage the WhatsApp connection: start a QR login, check status, and log out.',
    endpoints: [
      {
        id: 'device-connect',
        method: 'POST',
        path: '/api/device/connect',
        title: 'Connect / start QR login',
        description:
          'Starts a session. If no device is paired it begins a QR login (poll /device/qr or listen on the WebSocket for the code). If already paired, it reconnects.',
        responses: [{ status: 200, body: { success: true, status: 'connecting' } }],
        try: { method: 'POST', path: '/device/connect', fields: [] },
      },
      {
        id: 'device-qr',
        method: 'GET',
        path: '/api/device/qr',
        title: 'Get current QR code',
        description: 'Returns the latest QR string (also pushed over WebSocket as a "qr" event). Empty once connected.',
        responses: [{ status: 200, body: { success: true, code: '2@abc...', status: 'connecting' } }],
        try: { method: 'GET', path: '/device/qr', fields: [] },
      },
      {
        id: 'device-status',
        method: 'GET',
        path: '/api/device/status',
        title: 'Device status',
        description: 'Returns connection status and the linked account metadata.',
        responses: [
          {
            status: 200,
            body: {
              success: true,
              status: 'connected',
              connected: true,
              phone: '628123456789',
              push_name: 'John',
              jid: '628123456789@s.whatsapp.net',
            },
          },
        ],
        try: { method: 'GET', path: '/device/status', fields: [] },
      },
      {
        id: 'device-logout',
        method: 'POST',
        path: '/api/device/logout',
        title: 'Logout',
        description: 'Terminates the WhatsApp session and clears the stored credentials.',
        responses: [{ status: 200, body: { success: true, status: 'logged_out' } }],
        try: { method: 'POST', path: '/device/logout', fields: [] },
      },
    ],
  },

  {
    id: 'messages',
    name: 'Messaging',
    icon: 'icon-[tabler--send]',
    description: 'Send text, images and documents to any WhatsApp number.',
    endpoints: [
      {
        id: 'send-text',
        method: 'POST',
        path: '/api/messages/send',
        title: 'Send text message',
        description: 'Sends a plain text message to a phone number (international format, no +).',
        auth: 'Basic / X-Api-Key',
        body: {
          phone: '628123456789',
          message: 'Halo Dunia',
        },
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Recipient number, e.g. 628123456789' },
          { name: 'message', type: 'string', required: true, desc: 'Message text' },
        ],
        responses: [
          { status: 200, body: { success: true, message_id: '3EB0XXXX' } },
          { status: 429, body: { success: false, error: 'rate limited: please wait before messaging this number again', retry_after: 25 } },
          { status: 503, body: { success: false, error: 'whatsapp not connected' } },
        ],
        try: {
          method: 'POST',
          path: '/messages/send',
          fields: [
            { name: 'phone', in: 'body', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'message', in: 'body', type: 'textarea', required: true, placeholder: 'Halo Dunia' },
          ],
        },
      },
      {
        id: 'send-image',
        method: 'POST',
        path: '/api/messages/send-image',
        title: 'Send image',
        description: 'Sends an image with an optional caption. Uses multipart/form-data.',
        auth: 'Basic / X-Api-Key',
        contentType: 'multipart/form-data',
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Recipient number' },
          { name: 'caption', type: 'string', required: false, desc: 'Optional caption' },
          { name: 'file', type: 'file', required: true, desc: 'Image file (jpg/png/webp)' },
        ],
        responses: [{ status: 200, body: { success: true, message_id: '3EB0XXXX' } }],
        try: {
          method: 'POST',
          path: '/messages/send-image',
          contentType: 'multipart/form-data',
          fields: [
            { name: 'phone', in: 'form', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'caption', in: 'form', type: 'text', required: false, placeholder: 'My photo' },
            { name: 'file', in: 'form', type: 'file', required: true },
          ],
        },
      },
      {
        id: 'send-file',
        method: 'POST',
        path: '/api/messages/send-file',
        title: 'Send document',
        description: 'Sends a document/file with an optional caption. Uses multipart/form-data.',
        auth: 'Basic / X-Api-Key',
        contentType: 'multipart/form-data',
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Recipient number' },
          { name: 'caption', type: 'string', required: false, desc: 'Optional caption' },
          { name: 'file', type: 'file', required: true, desc: 'Any document (pdf, docx, ...)' },
        ],
        responses: [{ status: 200, body: { success: true, message_id: '3EB0XXXX' } }],
        try: {
          method: 'POST',
          path: '/messages/send-file',
          contentType: 'multipart/form-data',
          fields: [
            { name: 'phone', in: 'form', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'caption', in: 'form', type: 'text', required: false, placeholder: 'Invoice' },
            { name: 'file', in: 'form', type: 'file', required: true },
          ],
        },
      },
      {
        id: 'send-buttons',
        method: 'POST',
        path: '/api/messages/send-buttons',
        title: 'Send message with link buttons',
        description:
          'Sends a message with call-to-action links (e.g. "Lihat Hasil Pemeriksaan"). NOTE: WhatsApp does not reliably render native buttons from unofficial clients, so this delivers a normal text message whose links are clickable, with a rich link-preview card for the first URL. Reliably reaches the recipient and stays tappable.',
        auth: 'Basic / X-Api-Key',
        body: {
          phone: '628123456789',
          message:
            'Hi Bapak/Ibu Sri Sumiwaty,SE,\n\nTerima kasih atas kepercayaan Bapak/Ibu.\n\nSalam Sehat,\nHI-LAB Laboratorium & Klinik',
          footer: 'www.hilab.co.id',
          buttons: [{ text: 'Lihat Hasil Pemeriksaan', url: 'https://hilab.co.id/hasil/abc123' }],
        },
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Recipient number' },
          { name: 'message', type: 'string', required: true, desc: 'Body text (supports newlines)' },
          { name: 'footer', type: 'string', required: false, desc: 'Small footer text under the body' },
          { name: 'buttons', type: 'object[]', required: true, desc: 'Array of { text, url }; 1–3 buttons' },
        ],
        responses: [
          { status: 200, body: { success: true, message_id: '3EB0XXXX' } },
          { status: 429, body: { success: false, error: 'rate limited: please wait before messaging this number again', retry_after: 25 } },
          { status: 503, body: { success: false, error: 'whatsapp not connected' } },
        ],
        try: {
          method: 'POST',
          path: '/messages/send-buttons',
          fields: [
            { name: 'phone', in: 'body', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'message', in: 'body', type: 'textarea', required: true, placeholder: 'Hasil pemeriksaan Anda sudah tersedia.' },
            { name: 'footer', in: 'body', type: 'text', required: false, placeholder: 'www.hilab.co.id' },
            {
              name: 'buttons',
              in: 'body',
              type: 'json',
              required: true,
              default: '[{"text":"Lihat Hasil Pemeriksaan","url":"https://hilab.co.id/hasil/abc123"}]',
              help: 'JSON array of { text, url }',
            },
          ],
        },
      },
    ],
  },

  {
    id: 'presence',
    name: 'Presence & Anti-blocking',
    icon: 'icon-[tabler--shield-check]',
    description:
      'Helpers that reduce ban risk: verify a number is on WhatsApp, show a typing indicator, and mark messages as read before replying.',
    endpoints: [
      {
        id: 'check-exists',
        method: 'POST',
        path: '/api/messages/check-exists',
        title: 'Check numbers on WhatsApp',
        description: 'Returns which of the supplied numbers are registered on WhatsApp.',
        auth: 'Basic / X-Api-Key',
        body: { phones: ['628111111111', '628999999999'] },
        bodySchema: [{ name: 'phones', type: 'string[]', required: true, desc: 'Array of numbers to check' }],
        responses: [
          {
            status: 200,
            body: {
              success: true,
              results: [
                { phone: '628111111111', exists: true, jid: '628111111111@s.whatsapp.net' },
                { phone: '628999999999', exists: false, jid: '' },
              ],
            },
          },
        ],
        try: {
          method: 'POST',
          path: '/messages/check-exists',
          fields: [
            {
              name: 'phones',
              in: 'body',
              type: 'json',
              required: true,
              default: '["628111111111","628999999999"]',
              help: 'JSON array of phone numbers',
            },
          ],
        },
      },
      {
        id: 'typing',
        method: 'POST',
        path: '/api/messages/typing',
        title: 'Typing indicator',
        description: 'Shows or hides the "typing…" indicator in a chat.',
        auth: 'Basic / X-Api-Key',
        body: { phone: '628123456789', typing: true },
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Recipient number' },
          { name: 'typing', type: 'boolean', required: false, desc: 'true = composing, false = paused' },
        ],
        responses: [{ status: 200, body: { success: true } }],
        try: {
          method: 'POST',
          path: '/messages/typing',
          fields: [
            { name: 'phone', in: 'body', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'typing', in: 'body', type: 'boolean', default: true },
          ],
        },
      },
      {
        id: 'seen',
        method: 'POST',
        path: '/api/messages/seen',
        title: 'Mark messages as read',
        description: 'Sends read receipts for the given message IDs in a chat.',
        auth: 'Basic / X-Api-Key',
        body: { phone: '628123456789', message_ids: ['3EB0XXXX'] },
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Recipient number' },
          { name: 'message_ids', type: 'string[]', required: true, desc: 'Message IDs to mark as read' },
        ],
        responses: [{ status: 200, body: { success: true } }],
        try: {
          method: 'POST',
          path: '/messages/seen',
          fields: [
            { name: 'phone', in: 'body', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'message_ids', in: 'body', type: 'json', required: true, default: '["3EB0XXXX"]' },
          ],
        },
      },
    ],
  },

  {
    id: 'conversations',
    name: 'Conversations & Chat',
    icon: 'icon-[tabler--messages]',
    description: 'Read the inbox, fetch a conversation history, and reply with text or media.',
    endpoints: [
      {
        id: 'start-chat',
        method: 'POST',
        path: '/api/chats/start',
        title: 'Start a new chat (single target)',
        description:
          'Begins a 1:1 conversation with a single number and sends the first message. If the number is already a contact, its existing conversation is reused (auto-synced). Verifies the number is on WhatsApp first. Returns the conversation so a UI can open it.',
        auth: 'Basic / X-Api-Key',
        body: { phone: '628123456789', message: 'Halo, ada yang bisa kami bantu?' },
        bodySchema: [
          { name: 'phone', type: 'string', required: true, desc: 'Target number (international, no +)' },
          { name: 'message', type: 'string', required: true, desc: 'First message text' },
        ],
        responses: [
          {
            status: 200,
            body: {
              success: true,
              message_id: '3EB0XXXX',
              conversation_id: 1,
              conversation: { id: 1, contact: { name: 'John', phone: '628123456789' } },
            },
          },
          { status: 422, body: { success: false, error: 'number is not registered on WhatsApp' } },
          { status: 429, body: { success: false, error: 'rate limited: please wait before messaging this number again', retry_after: 25 } },
        ],
        try: {
          method: 'POST',
          path: '/chats/start',
          fields: [
            { name: 'phone', in: 'body', type: 'text', required: true, placeholder: '628123456789' },
            { name: 'message', in: 'body', type: 'textarea', required: true, placeholder: 'Halo!' },
          ],
        },
      },
      {
        id: 'list-conversations',
        method: 'GET',
        path: '/api/conversations',
        title: 'List conversations',
        description: 'Returns conversations with the last message preview, newest first.',
        responses: [
          {
            status: 200,
            body: {
              success: true,
              conversations: [
                {
                  id: 1,
                  contact: { id: 1, name: 'John', phone: '628123456789' },
                  last_message: 'Hello',
                  unread_count: 2,
                },
              ],
            },
          },
        ],
        try: { method: 'GET', path: '/conversations', fields: [] },
      },
      {
        id: 'chat-detail',
        method: 'GET',
        path: '/api/conversations/{id}',
        title: 'Conversation history',
        description: 'Returns the conversation and its messages (marks it read).',
        params: [{ name: 'id', in: 'path', type: 'number', required: true, desc: 'Conversation ID' }],
        responses: [
          {
            status: 200,
            body: {
              success: true,
              conversation: { id: 1, contact: { name: 'John', phone: '628123456789' } },
              messages: [
                { id: 10, direction: 'incoming', message_type: 'text', content: 'Hi', status: 'delivered' },
              ],
            },
          },
        ],
        try: {
          method: 'GET',
          path: '/conversations/{id}',
          fields: [{ name: 'id', in: 'path', type: 'number', required: true, placeholder: '1' }],
        },
      },
      {
        id: 'reply',
        method: 'POST',
        path: '/api/conversations/{id}/reply',
        title: 'Reply (text)',
        description: 'Sends a text reply within an existing conversation.',
        params: [{ name: 'id', in: 'path', type: 'number', required: true, desc: 'Conversation ID' }],
        body: { message: 'Thanks!' },
        bodySchema: [{ name: 'message', type: 'string', required: true, desc: 'Reply text' }],
        responses: [{ status: 200, body: { success: true, message_id: '3EB0XXXX' } }],
        try: {
          method: 'POST',
          path: '/conversations/{id}/reply',
          fields: [
            { name: 'id', in: 'path', type: 'number', required: true, placeholder: '1' },
            { name: 'message', in: 'body', type: 'textarea', required: true, placeholder: 'Thanks!' },
          ],
        },
      },
      {
        id: 'reply-media',
        method: 'POST',
        path: '/api/conversations/{id}/reply-media',
        title: 'Reply (image / file)',
        description:
          'Sends an image or document within a conversation. Type is chosen from the file MIME: image/* → image, otherwise document. multipart/form-data.',
        contentType: 'multipart/form-data',
        params: [{ name: 'id', in: 'path', type: 'number', required: true, desc: 'Conversation ID' }],
        bodySchema: [
          { name: 'caption', type: 'string', required: false, desc: 'Optional caption' },
          { name: 'file', type: 'file', required: true, desc: 'Image or document' },
        ],
        responses: [{ status: 200, body: { success: true, message_id: '3EB0XXXX' } }],
        try: {
          method: 'POST',
          path: '/conversations/{id}/reply-media',
          contentType: 'multipart/form-data',
          fields: [
            { name: 'id', in: 'path', type: 'number', required: true, placeholder: '1' },
            { name: 'caption', in: 'form', type: 'text', required: false },
            { name: 'file', in: 'form', type: 'file', required: true },
          ],
        },
      },
    ],
  },

  {
    id: 'contacts',
    name: 'Contacts',
    icon: 'icon-[tabler--users]',
    description: 'List and search contacts the platform has interacted with.',
    endpoints: [
      {
        id: 'list-contacts',
        method: 'GET',
        path: '/api/contacts',
        title: 'List / search contacts',
        description: 'Returns contacts, optionally filtered by a search term over name and phone.',
        params: [{ name: 'search', in: 'query', type: 'text', required: false, desc: 'Search term' }],
        responses: [
          {
            status: 200,
            body: { success: true, contacts: [{ id: 1, name: 'John', phone: '628123456789' }] },
          },
        ],
        try: {
          method: 'GET',
          path: '/contacts',
          fields: [{ name: 'search', in: 'query', type: 'text', required: false, placeholder: 'john' }],
        },
      },
      {
        id: 'contact-detail',
        method: 'GET',
        path: '/api/contacts/{id}',
        title: 'Contact detail',
        description: 'Returns a single contact by ID.',
        params: [{ name: 'id', in: 'path', type: 'number', required: true, desc: 'Contact ID' }],
        responses: [{ status: 200, body: { success: true, contact: { id: 1, name: 'John', phone: '628123456789' } } }],
        try: {
          method: 'GET',
          path: '/contacts/{id}',
          fields: [{ name: 'id', in: 'path', type: 'number', required: true, placeholder: '1' }],
        },
      },
    ],
  },

  {
    id: 'broadcast',
    name: 'Broadcast',
    icon: 'icon-[tabler--speakerphone]',
    description:
      'Queue bulk campaigns. Sends with random delay + retry, skips numbers not on WhatsApp. Only message is required — leave phones empty to target all contacts.',
    endpoints: [
      {
        id: 'create-broadcast',
        method: 'POST',
        path: '/api/broadcasts',
        title: 'Create broadcast',
        description: 'Queues a broadcast. If phones is omitted, all saved contacts are targeted.',
        body: { name: 'Promo', message: 'Promo Hari Ini', phones: ['628111111111', '628222222222'] },
        bodySchema: [
          { name: 'name', type: 'string', required: false, desc: 'Campaign name' },
          { name: 'message', type: 'string', required: true, desc: 'Message to send' },
          { name: 'phones', type: 'string[]', required: false, desc: 'Recipients (empty = all contacts)' },
        ],
        responses: [{ status: 200, body: { success: true, broadcast_id: 1, total_target: 2 } }],
        try: {
          method: 'POST',
          path: '/broadcasts',
          fields: [
            { name: 'name', in: 'body', type: 'text', required: false, placeholder: 'Promo' },
            { name: 'message', in: 'body', type: 'textarea', required: true, placeholder: 'Promo Hari Ini' },
            { name: 'phones', in: 'body', type: 'json', required: false, default: '["628111111111"]', help: 'JSON array (optional)' },
          ],
        },
      },
      {
        id: 'list-broadcasts',
        method: 'GET',
        path: '/api/broadcasts',
        title: 'Broadcast history',
        description: 'Returns broadcasts newest first with success/failure counts.',
        responses: [
          {
            status: 200,
            body: {
              success: true,
              broadcasts: [{ id: 1, name: 'Promo', status: 'success', total_target: 2, total_success: 2, total_failed: 0 }],
            },
          },
        ],
        try: { method: 'GET', path: '/broadcasts', fields: [] },
      },
      {
        id: 'broadcast-detail',
        method: 'GET',
        path: '/api/broadcasts/{id}',
        title: 'Broadcast detail',
        description: 'Returns a broadcast and its per-recipient delivery details.',
        params: [{ name: 'id', in: 'path', type: 'number', required: true, desc: 'Broadcast ID' }],
        responses: [
          {
            status: 200,
            body: {
              success: true,
              broadcast: { id: 1, name: 'Promo', status: 'success' },
              details: [{ phone: '628111111111', status: 'success', error_message: '' }],
            },
          },
        ],
        try: {
          method: 'GET',
          path: '/broadcasts/{id}',
          fields: [{ name: 'id', in: 'path', type: 'number', required: true, placeholder: '1' }],
        },
      },
    ],
  },

  {
    id: 'dashboard',
    name: 'Dashboard',
    icon: 'icon-[tabler--layout-dashboard]',
    description: 'Aggregate counters for the dashboard home.',
    endpoints: [
      {
        id: 'stats',
        method: 'GET',
        path: '/api/dashboard/stats',
        title: 'Dashboard stats',
        description: 'Returns total counts and device summary.',
        responses: [
          {
            status: 200,
            body: {
              success: true,
              total_contacts: 12,
              total_chats: 8,
              total_broadcasts: 3,
              device: { status: 'connected', connected: true },
            },
          },
        ],
        try: { method: 'GET', path: '/dashboard/stats', fields: [] },
      },
    ],
  },

  {
    id: 'logs',
    name: 'Logs',
    icon: 'icon-[tabler--file-text]',
    description: 'Read the daily activity logs (audit of every operation and event).',
    endpoints: [
      {
        id: 'log-days',
        method: 'GET',
        path: '/api/logs/days',
        title: 'List log days',
        description: 'Returns available log dates (YYYY-MM-DD), newest first.',
        responses: [{ status: 200, body: { success: true, days: ['2026-05-30', '2026-05-29'] } }],
        try: { method: 'GET', path: '/logs/days', fields: [] },
      },
      {
        id: 'log-entries',
        method: 'GET',
        path: '/api/logs',
        title: 'Query log entries',
        description: 'Returns filtered, paginated log entries for a day.',
        params: [
          { name: 'day', in: 'query', type: 'text', required: false, desc: 'YYYY-MM-DD (default today)' },
          { name: 'search', in: 'query', type: 'text', required: false, desc: 'Substring filter' },
          { name: 'status', in: 'query', type: 'text', required: false, desc: 'success|failed|issue|info' },
          { name: 'page', in: 'query', type: 'number', required: false, desc: 'Page (default 1)' },
          { name: 'per_page', in: 'query', type: 'number', required: false, desc: 'Page size (default 50)' },
        ],
        responses: [
          {
            status: 200,
            body: {
              success: true,
              result: { day: '2026-05-30', total: 541, page: 1, per_page: 50, pages: 11, entries: [] },
            },
          },
        ],
        try: {
          method: 'GET',
          path: '/logs',
          fields: [
            { name: 'search', in: 'query', type: 'text', required: false, placeholder: 'send' },
            { name: 'status', in: 'query', type: 'text', required: false, placeholder: 'failed' },
            { name: 'per_page', in: 'query', type: 'number', required: false, default: 10 },
          ],
        },
      },
    ],
  },

  {
    id: 'webhooks',
    name: 'Webhooks',
    icon: 'icon-[tabler--webhook]',
    description:
      'Set WEBHOOK_URL to receive events via HTTP POST. If WEBHOOK_SECRET is set, the body is signed with HMAC-SHA256 in the X-Webhook-Signature header (sha256=...). Events: message, message.ack, session.status.',
    endpoints: [
      {
        id: 'webhook-payload',
        method: 'POST',
        path: '(your WEBHOOK_URL)',
        title: 'Incoming message event',
        description: 'Example payload your server receives when a message arrives.',
        noTry: true,
        responses: [
          {
            status: 200,
            label: 'Payload sent to your endpoint',
            body: {
              event: 'message',
              timestamp: 1730000000,
              session: 'default',
              data: {
                from: '628123456789',
                name: 'John',
                type: 'text',
                content: 'Hello',
                message_id: '3EB0XXXX',
              },
            },
          },
        ],
      },
    ],
  },
]

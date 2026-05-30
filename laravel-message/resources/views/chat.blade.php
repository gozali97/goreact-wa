<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>WA Tester · Laravel</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        wa: {
                            green: '#00a884',
                            dark: '#111b21',
                            panel: '#202c33',
                            hover: '#2a3942',
                            bubble: '#005c4b',
                            incoming: '#202c33',
                            teal: '#008069',
                        },
                    },
                },
            },
        }
    </script>
    <style>
        .chat-bg { background-color: #0b141a; background-image: url("data:image/svg+xml,%3Csvg width='40' height='40' viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='%23131e26' fill-opacity='0.4'%3E%3Cpath d='M0 38.59l2.83-2.83 1.41 1.41L1.41 40H0v-1.41zM0 1.4l2.83 2.83 1.41-1.41L1.41 0H0v1.41zM38.59 40l-2.83-2.83 1.41-1.41L40 38.59V40h-1.41zM40 1.41l-2.83 2.83-1.41-1.41L38.59 0H40v1.41z'/%3E%3C/g%3E%3C/svg%3E"); }
        ::-webkit-scrollbar { width: 7px; }
        ::-webkit-scrollbar-thumb { background: #374248; border-radius: 9999px; }
    </style>
</head>
<body class="h-screen overflow-hidden bg-wa-dark text-gray-100">
<div id="app" x-data="chatApp" class="flex h-full">

    <!-- Sidebar: conversations -->
    <aside class="flex w-full flex-col border-r border-black/30 bg-wa-dark md:w-[30%] md:min-w-[320px]"
           :class="(activeId && isMobile) ? 'hidden' : 'flex'">
        <!-- Header -->
        <header class="flex items-center justify-between bg-wa-panel px-4 py-3">
            <div class="flex items-center gap-2">
                <div class="flex h-9 w-9 items-center justify-center rounded-full bg-wa-green text-sm font-bold">WA</div>
                <div>
                    <p class="text-sm font-semibold">WA Tester</p>
                    <p class="text-[11px]" :class="connected ? 'text-wa-green' : 'text-gray-400'"
                       x-text="connected ? ('Connected · ' + (device.phone || '')) : (device.status || 'disconnected')"></p>
                </div>
            </div>
            <div class="flex items-center gap-1">
                <button @click="openNew = true" title="New chat"
                        class="flex h-9 w-9 items-center justify-center rounded-full hover:bg-wa-hover">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
                </button>
                <button @click="loadConversations(true)" title="Refresh"
                        class="flex h-9 w-9 items-center justify-center rounded-full hover:bg-wa-hover">
                    <svg class="h-5 w-5" :class="loadingConvs && 'animate-spin'" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h5M20 20v-5h-5M4 9a8 8 0 0114-3M20 15a8 8 0 01-14 3"/></svg>
                </button>
            </div>
        </header>

        <!-- Search -->
        <div class="bg-wa-dark px-3 py-2">
            <div class="flex items-center gap-2 rounded-lg bg-wa-panel px-3 py-1.5">
                <svg class="h-4 w-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z"/></svg>
                <input x-model="search" type="text" placeholder="Search or start new chat"
                       class="w-full bg-transparent text-sm outline-none placeholder:text-gray-500">
            </div>
        </div>

        <!-- Conversation list -->
        <div class="flex-1 overflow-y-auto">
            <template x-if="filteredConvs.length === 0">
                <p class="p-6 text-center text-sm text-gray-500">No conversations.</p>
            </template>
            <template x-for="c in filteredConvs" :key="c.id">
                <button @click="openChat(c.id)"
                        class="flex w-full items-center gap-3 border-b border-black/20 px-4 py-3 text-left hover:bg-wa-hover"
                        :class="activeId === c.id && 'bg-wa-hover'">
                    <div class="flex h-12 w-12 flex-none items-center justify-center rounded-full bg-gradient-to-br from-wa-green to-emerald-700 font-semibold"
                         x-text="initials(c.contact?.name || c.contact?.phone)"></div>
                    <div class="min-w-0 flex-1">
                        <div class="flex justify-between">
                            <span class="truncate text-sm font-medium" x-text="c.contact?.name || c.contact?.phone"></span>
                            <span class="ml-2 flex-none text-[10px] text-gray-400" x-text="fmtTime(c.last_message_at)"></span>
                        </div>
                        <div class="flex items-center justify-between">
                            <span class="truncate text-xs text-gray-400" x-text="c.last_message || '—'"></span>
                            <template x-if="c.unread_count > 0">
                                <span class="ml-2 flex h-5 min-w-5 items-center justify-center rounded-full bg-wa-green px-1.5 text-[10px] font-bold text-wa-dark" x-text="c.unread_count"></span>
                            </template>
                        </div>
                    </div>
                </button>
            </template>
        </div>
    </aside>

    <!-- Chat panel -->
    <main class="flex flex-1 flex-col" :class="!activeId && isMobile ? 'hidden' : 'flex'">
        <template x-if="!activeId">
            <div class="flex h-full flex-col items-center justify-center gap-3 bg-wa-panel text-center">
                <div class="flex h-20 w-20 items-center justify-center rounded-full bg-wa-hover">
                    <svg class="h-10 w-10 text-wa-green" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2a10 10 0 00-8.94 14.47L2 22l5.66-1.48A10 10 0 1012 2z"/></svg>
                </div>
                <h2 class="text-xl font-light text-gray-200">WA Proxy Tester</h2>
                <p class="max-w-sm text-sm text-gray-500">Select a conversation, or start a new chat to test sending messages through your wa-proxy server.</p>
            </div>
        </template>

        <template x-if="activeId">
            <div class="flex h-full flex-col">
                <!-- Chat header -->
                <header class="flex items-center gap-3 bg-wa-panel px-4 py-2.5">
                    <button @click="activeId = null" class="md:hidden flex h-9 w-9 items-center justify-center rounded-full hover:bg-wa-hover">
                        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>
                    </button>
                    <div class="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-wa-green to-emerald-700 font-semibold"
                         x-text="initials(activeContact?.name || activeContact?.phone)"></div>
                    <div class="min-w-0">
                        <p class="truncate text-sm font-medium" x-text="activeContact?.name || activeContact?.phone"></p>
                        <p class="truncate text-[11px] text-gray-400" x-text="activeContact?.phone"></p>
                    </div>
                </header>

                <!-- Messages -->
                <div id="thread" class="chat-bg flex-1 space-y-1.5 overflow-y-auto px-4 py-4 md:px-12">
                    <template x-for="m in messages" :key="m.id">
                        <div class="flex" :class="m.direction === 'outgoing' ? 'justify-end' : 'justify-start'">
                            <div class="max-w-[75%] rounded-lg px-2.5 py-1.5 text-sm shadow"
                                 :class="m.direction === 'outgoing' ? 'bg-wa-bubble' : 'bg-wa-incoming'">
                                <!-- image -->
                                <template x-if="m.media_url && m.message_type === 'image'">
                                    <img :src="mediaSrc(m.media_url)" @click="lightbox = mediaSrc(m.media_url)"
                                         class="mb-1 max-h-64 cursor-pointer rounded-md">
                                </template>
                                <!-- video -->
                                <template x-if="m.media_url && m.message_type === 'video'">
                                    <video :src="mediaSrc(m.media_url)" controls class="mb-1 max-h-64 rounded-md"></video>
                                </template>
                                <!-- audio -->
                                <template x-if="m.media_url && m.message_type === 'audio'">
                                    <audio :src="mediaSrc(m.media_url)" controls class="mb-1 w-56"></audio>
                                </template>
                                <!-- document / sticker fallback -->
                                <template x-if="m.media_url && (m.message_type === 'document' || m.message_type === 'sticker')">
                                    <a :href="mediaSrc(m.media_url)" target="_blank"
                                       class="mb-1 flex items-center gap-2 rounded-md bg-black/20 px-2 py-1.5 underline">
                                        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h10M7 12h10M7 17h6"/></svg>
                                        <span x-text="m.file_name || m.message_type"></span>
                                    </a>
                                </template>
                                <template x-if="m.content">
                                    <p class="whitespace-pre-wrap break-words" x-text="m.content"></p>
                                </template>
                                <div class="mt-0.5 flex items-center justify-end gap-1 text-[10px] text-gray-300/70">
                                    <span x-text="fmtTime(m.sent_at || m.created_at)"></span>
                                    <template x-if="m.direction === 'outgoing'">
                                        <span x-text="ackIcon(m.status)"></span>
                                    </template>
                                </div>
                            </div>
                        </div>
                    </template>
                </div>

                <!-- Composer -->
                <footer class="bg-wa-panel px-3 py-2.5">
                    <template x-if="attachmentName">
                        <div class="mb-2 flex items-center gap-2 rounded-lg bg-wa-hover px-3 py-2 text-xs">
                            <span class="text-wa-green">📎</span>
                            <span class="flex-1 truncate" x-text="attachmentName"></span>
                            <button @click="clearAttachment()" class="text-gray-400 hover:text-white">✕</button>
                        </div>
                    </template>
                    <form @submit.prevent="send()" class="flex items-end gap-2">
                        <label class="flex h-10 w-10 flex-none cursor-pointer items-center justify-center rounded-full text-gray-300 hover:bg-wa-hover">
                            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20 11"/></svg>
                            <input type="file" class="hidden" @change="pickFile($event)">
                        </label>
                        <input x-model="draft" type="text"
                               :placeholder="attachmentName ? 'Add a caption…' : 'Type a message'"
                               class="flex-1 rounded-lg bg-wa-hover px-4 py-2.5 text-sm outline-none placeholder:text-gray-500">
                        <button type="submit" :disabled="sending"
                                class="flex h-10 w-10 flex-none items-center justify-center rounded-full bg-wa-green text-wa-dark hover:bg-emerald-400 disabled:opacity-50">
                            <template x-if="!sending">
                                <svg class="h-5 w-5" fill="currentColor" viewBox="0 0 24 24"><path d="M2 21l21-9L2 3v7l15 2-15 2v7z"/></svg>
                            </template>
                            <template x-if="sending">
                                <svg class="h-5 w-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-width="2" d="M12 3a9 9 0 109 9"/></svg>
                            </template>
                        </button>
                    </form>
                </footer>
            </div>
        </template>
    </main>

    <!-- New chat modal -->
    <template x-if="openNew">
        <div class="fixed inset-0 z-40 flex items-center justify-center bg-black/60 p-4" @click="openNew = false">
            <div class="w-full max-w-sm rounded-xl bg-wa-panel p-5" @click.stop>
                <h3 class="mb-3 text-lg font-semibold">New chat</h3>
                <label class="mb-1 block text-xs text-gray-400">Phone number</label>
                <input x-model="newPhone" placeholder="628123456789"
                       class="mb-3 w-full rounded-lg bg-wa-hover px-3 py-2 text-sm outline-none">
                <label class="mb-1 block text-xs text-gray-400">Message</label>
                <textarea x-model="newMessage" rows="3" placeholder="Hello!"
                          class="mb-3 w-full rounded-lg bg-wa-hover px-3 py-2 text-sm outline-none"></textarea>
                <p x-show="newError" x-text="newError" class="mb-2 text-xs text-red-400"></p>
                <div class="flex justify-end gap-2">
                    <button @click="openNew = false" class="rounded-lg px-4 py-2 text-sm hover:bg-wa-hover">Cancel</button>
                    <button @click="sendNew()" :disabled="newSending"
                            class="rounded-lg bg-wa-green px-4 py-2 text-sm font-semibold text-wa-dark hover:bg-emerald-400 disabled:opacity-50">
                        <span x-text="newSending ? 'Sending…' : 'Send'"></span>
                    </button>
                </div>
            </div>
        </div>
    </template>

    <!-- Lightbox -->
    <template x-if="lightbox">
        <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-4" @click="lightbox = null">
            <img :src="lightbox" class="max-h-[90vh] max-w-[90vw] rounded-lg">
        </div>
    </template>

    <!-- Toast -->
    <template x-if="toastMsg">
        <div class="fixed bottom-5 left-1/2 z-[60] -translate-x-1/2 rounded-xl bg-wa-panel px-4 py-2.5 text-sm shadow-xl ring-1 ring-white/10"
             x-text="toastMsg"></div>
    </template>
</div>

<script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
<script>
    const CSRF = document.querySelector('meta[name="csrf-token"]').content
    async function jget(url) { const r = await fetch(url, { headers: { 'Accept': 'application/json' } }); return r.json() }
    async function jpost(url, body, isForm) {
        const opts = { method: 'POST', headers: { 'X-CSRF-TOKEN': CSRF, 'Accept': 'application/json' } }
        if (isForm) { opts.body = body }
        else { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body) }
        const r = await fetch(url, opts)
        return { ok: r.ok, status: r.status, data: await r.json().catch(() => ({})) }
    }

    function chatApp() {
        return {
            isMobile: window.innerWidth < 768,
            connected: false,
            device: {},
            conversations: [],
            loadingConvs: false,
            search: '',
            activeId: null,
            messages: [],
            draft: '',
            file: null,
            attachmentName: '',
            sending: false,
            lightbox: null,
            // new chat
            openNew: false,
            newPhone: '', newMessage: '', newSending: false, newError: '',

            get filteredConvs() {
                const s = this.search.toLowerCase().trim()
                if (!s) return this.conversations
                return this.conversations.filter(c =>
                    (c.contact?.name || '').toLowerCase().includes(s) ||
                    (c.contact?.phone || '').toLowerCase().includes(s))
            },
            get activeContact() {
                return this.conversations.find(c => c.id === this.activeId)?.contact || {}
            },

            init() {
                this.loadStatus(); this.loadConversations()
                setInterval(() => { this.loadStatus(); this.loadConversations() }, 8000)
                setInterval(() => { if (this.activeId) this.loadMessages(this.activeId, true) }, 5000)
                window.addEventListener('resize', () => this.isMobile = window.innerWidth < 768)
            },

            initials(t) { return (t || '?').trim().charAt(0).toUpperCase() },
            fmtTime(ts) { if (!ts) return ''; const d = new Date(ts); return d.toLocaleString([], { hour: '2-digit', minute: '2-digit' }) },
            ackIcon(s) { return s === 'read' ? '✓✓' : s === 'delivered' ? '✓✓' : s === 'sent' ? '✓' : s === 'pending' ? '🕓' : s === 'failed' ? '⚠' : '' },
            mediaSrc(p) { return '/chat/media?path=' + encodeURIComponent(p) },

            async loadStatus() {
                try { const d = await jget('/chat/status'); this.device = d; this.connected = !!d.connected } catch (e) {}
            },
            async loadConversations(manual) {
                if (manual) this.loadingConvs = true
                try { const d = await jget('/chat/conversations'); this.conversations = d.conversations || [] } catch (e) {}
                finally { this.loadingConvs = false }
            },
            async openChat(id) { this.activeId = id; this.messages = []; await this.loadMessages(id); this.markRead(id) },
            markRead(id) { const c = this.conversations.find(x => x.id === id); if (c) c.unread_count = 0 },
            async loadMessages(id, silent) {
                try {
                    const d = await jget('/chat/conversations/' + id)
                    this.messages = d.messages || []
                    if (!silent) this.$nextTick(() => this.scrollBottom())
                    else this.$nextTick(() => this.scrollBottom())
                } catch (e) {}
            },
            scrollBottom() { const el = document.getElementById('thread'); if (el) el.scrollTop = el.scrollHeight },

            pickFile(e) { this.file = e.target.files[0] || null; this.attachmentName = this.file?.name || '' },
            clearAttachment() { this.file = null; this.attachmentName = '' },

            async send() {
                if (this.sending || !this.activeId) return
                if (!this.draft.trim() && !this.file) return
                this.sending = true
                try {
                    let res
                    if (this.file) {
                        const fd = new FormData()
                        fd.append('file', this.file)
                        if (this.draft.trim()) fd.append('caption', this.draft.trim())
                        res = await jpost('/chat/conversations/' + this.activeId + '/reply-media', fd, true)
                        if (res.ok && res.data.success !== false) this.clearAttachment()
                    } else {
                        res = await jpost('/chat/conversations/' + this.activeId + '/reply', { message: this.draft.trim() })
                    }
                    if (res.status === 429) {
                        this.toast('⏳ Rate limited: wait ' + (res.data.retry_after || 30) + 's before messaging this number again.')
                        return
                    }
                    if (!res.ok || res.data.success === false) {
                        this.toast('⚠ ' + (res.data.error || 'Failed to send.'))
                        return
                    }
                    this.draft = ''
                    await this.loadMessages(this.activeId)
                    this.loadConversations()
                } catch (e) { this.toast('⚠ Request failed.') } finally { this.sending = false }
            },

            toastMsg: '',
            toast(msg) {
                this.toastMsg = msg
                clearTimeout(this._toastT)
                this._toastT = setTimeout(() => { this.toastMsg = '' }, 4000)
            },

            async sendNew() {
                this.newError = ''
                if (!this.newPhone.trim() || !this.newMessage.trim()) { this.newError = 'Phone and message are required.'; return }
                this.newSending = true
                try {
                    const res = await jpost('/chat/send-new', { phone: this.newPhone.trim(), message: this.newMessage.trim() })
                    if (!res.ok || res.data.success === false) { this.newError = res.data.error || 'Failed to send.'; return }
                    this.openNew = false; this.newPhone = ''; this.newMessage = ''
                    await this.loadConversations()
                } catch (e) { this.newError = 'Request failed.' } finally { this.newSending = false }
            },
        }
    }
</script>
<script>document.addEventListener('alpine:init', () => { Alpine.data('chatApp', chatApp) })</script>
</body>
</html>

<?php

namespace App\Services;

use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;

/**
 * Thin client for the wa-proxy REST API. Keeps credentials server-side so the
 * browser never sees the API key.
 */
class WaProxy
{
    protected string $base;

    public function __construct()
    {
        $this->base = config('waproxy.url');
    }

    /**
     * Build a pre-authenticated HTTP client. Sends both Basic Auth (accepted by
     * dashboard endpoints) and X-Api-Key (accepted by integration endpoints),
     * so every wa-proxy route authenticates regardless of which scheme it uses.
     */
    protected function client(): PendingRequest
    {
        $req = Http::timeout(config('waproxy.timeout'))
            ->acceptJson()
            ->baseUrl($this->base.'/api')
            ->withBasicAuth(config('waproxy.username'), config('waproxy.password'));

        $apiKey = config('waproxy.api_key');
        if (! empty($apiKey)) {
            $req = $req->withHeaders(['X-Api-Key' => $apiKey]);
        }

        return $req;
    }

    // ─── Device ───────────────────────────────────────────
    public function status(): array
    {
        return $this->client()->get('/device/status')->json() ?? [];
    }

    public function connect(): array
    {
        return $this->client()->post('/device/connect')->json() ?? [];
    }

    public function qr(): array
    {
        return $this->client()->get('/device/qr')->json() ?? [];
    }

    public function logout(): array
    {
        return $this->client()->post('/device/logout')->json() ?? [];
    }

    // ─── Conversations ────────────────────────────────────
    public function conversations(): array
    {
        return $this->client()->get('/conversations')->json() ?? [];
    }

    public function chatDetail(int $id): array
    {
        return $this->client()->get("/conversations/{$id}")->json() ?? [];
    }

    public function reply(int $id, string $message): array
    {
        return $this->client()->post("/conversations/{$id}/reply", [
            'message' => $message,
        ])->json() ?? [];
    }

    public function replyMedia(int $id, $file, ?string $caption = null): array
    {
        $req = $this->client()->attach(
            'file',
            file_get_contents($file->getRealPath()),
            $file->getClientOriginalName()
        );

        $payload = [];
        if ($caption !== null && $caption !== '') {
            $payload['caption'] = $caption;
        }

        return $req->post("/conversations/{$id}/reply-media", $payload)->json() ?? [];
    }

    // ─── Messaging (by phone) ─────────────────────────────
    public function sendText(string $phone, string $message): array
    {
        return $this->client()->post('/messages/send', [
            'phone' => $phone,
            'message' => $message,
        ])->json() ?? [];
    }

    public function checkExists(array $phones): array
    {
        return $this->client()->post('/messages/check-exists', [
            'phones' => $phones,
        ])->json() ?? [];
    }

    // ─── Contacts ─────────────────────────────────────────
    public function contacts(string $search = ''): array
    {
        return $this->client()->get('/contacts', ['search' => $search])->json() ?? [];
    }

    /**
     * Fetch a media file (auth-protected /storage path) and return its raw
     * bytes + content type, so Blade can proxy images/videos to the browser.
     */
    public function media(string $path): array
    {
        $res = Http::timeout(config('waproxy.timeout'))
            ->withBasicAuth(config('waproxy.username'), config('waproxy.password'))
            ->get($this->base.$path);

        return [
            'ok' => $res->successful(),
            'body' => $res->body(),
            'type' => $res->header('Content-Type') ?: 'application/octet-stream',
        ];
    }
}

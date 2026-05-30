<?php

namespace App\Http\Controllers;

use App\Services\WaProxy;
use Illuminate\Http\Request;

class ChatController extends Controller
{
    public function __construct(protected WaProxy $wa) {}

    /** Render the chat shell. */
    public function index()
    {
        return view('chat');
    }

    /** Device/connection status. */
    public function status()
    {
        return response()->json($this->wa->status());
    }

    /** List conversations (inbox). */
    public function conversations()
    {
        return response()->json($this->wa->conversations());
    }

    /** Conversation history. */
    public function detail(int $id)
    {
        return response()->json($this->wa->chatDetail($id));
    }

    /** Send a text reply within a conversation. */
    public function reply(Request $request, int $id)
    {
        $data = $request->validate(['message' => 'required|string']);

        return response()->json($this->wa->reply($id, $data['message']));
    }

    /** Send an image/document within a conversation. */
    public function replyMedia(Request $request, int $id)
    {
        $request->validate(['file' => 'required|file|max:51200']); // 50MB

        return response()->json(
            $this->wa->replyMedia($id, $request->file('file'), $request->input('caption'))
        );
    }

    /** Start a new chat by phone (sends a first text message). */
    public function sendNew(Request $request)
    {
        $data = $request->validate([
            'phone' => 'required|string',
            'message' => 'required|string',
        ]);

        return response()->json($this->wa->sendText($data['phone'], $data['message']));
    }

    /**
     * Proxy an auth-protected media file from wa-proxy to the browser so
     * <img>/<video> tags can render it without exposing credentials.
     */
    public function media(Request $request)
    {
        $path = $request->query('path', '');
        if (! str_starts_with($path, '/storage/')) {
            abort(400, 'invalid path');
        }
        $res = $this->wa->media($path);
        if (! $res['ok']) {
            abort(404);
        }

        return response($res['body'])->header('Content-Type', $res['type']);
    }
}

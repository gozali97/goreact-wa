<?php

use App\Http\Controllers\ChatController;
use Illuminate\Support\Facades\Route;

Route::get('/', [ChatController::class, 'index']);

// Chat UI backing endpoints (Laravel proxies these to wa-proxy).
Route::prefix('chat')->group(function () {
    Route::get('/status', [ChatController::class, 'status']);
    Route::get('/conversations', [ChatController::class, 'conversations']);
    Route::get('/conversations/{id}', [ChatController::class, 'detail'])->whereNumber('id');
    Route::post('/conversations/{id}/reply', [ChatController::class, 'reply'])->whereNumber('id');
    Route::post('/conversations/{id}/reply-media', [ChatController::class, 'replyMedia'])->whereNumber('id');
    Route::post('/send-new', [ChatController::class, 'sendNew']);
    Route::get('/media', [ChatController::class, 'media']);
});

<?php

return [
    // Base URL of the running wa-proxy server (no trailing slash).
    'url' => rtrim(env('WA_PROXY_URL', 'http://localhost:8080'), '/'),

    // Authentication: prefer the API key (X-Api-Key); Basic Auth as fallback.
    'api_key' => env('WA_PROXY_API_KEY', ''),
    'username' => env('WA_PROXY_USER', 'admin'),
    'password' => env('WA_PROXY_PASS', 'secret123'),

    // HTTP timeout (seconds).
    'timeout' => (int) env('WA_PROXY_TIMEOUT', 30),
];

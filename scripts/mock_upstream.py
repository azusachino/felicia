#!/usr/bin/env python3
"""Mock Dawarich + Immich upstreams for felicia's ingest workflow.

Stdlib only (matches scripts/test_api.py) — no external deps. Serves
fixture-backed, auth-checked, *paginated* responses so the real Go clients
(apps/providers/dawarich, apps/providers/immich) can be driven offline over HTTP.

Pagination is forced to one item per page so the clients must walk
X-Total-Pages (Dawarich) / nextPage (Immich).

Run:  make mock-up            (background)
  or: uv run python scripts/mock_upstream.py
Env:  MOCK_PORT (8099), MOCK_API_KEY (mock-key)
"""

import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

API_KEY = os.getenv("MOCK_API_KEY", "mock-key")
PAGE_SIZE = 1  # force multi-page so clients exercise pagination

# --- Dawarich GET /api/v1/tracks: GeoJSON FeatureCollection features ---
TRACKS = [
    {
        "type": "Feature",
        "geometry": {"type": "LineString", "coordinates": [[139.7671, 35.6812], [139.7003, 35.6895]]},
        "properties": {"id": 1, "start_at": "2026-03-20T09:00:00Z", "end_at": "2026-03-20T09:45:00Z",
                       "distance": 6300, "dominant_mode": "walking"},
    },
    {
        "type": "Feature",
        "geometry": {"type": "LineString", "coordinates": [[139.7003, 35.6895], [135.7583, 34.9859]]},
        "properties": {"id": 2, "start_at": "2026-03-20T12:00:00Z", "end_at": "2026-03-20T14:30:00Z",
                       "distance": 370000, "dominant_mode": "train"},
    },
]

# --- Dawarich GET /api/v1/visits: plain array ---
VISITS = [
    {"id": 7, "started_at": "2026-03-20T10:00:00Z", "ended_at": "2026-03-20T11:30:00Z",
     "name": "明治神宮", "status": "confirmed", "confidence": 0.92,
     "place": {"id": 3, "latitude": 35.6764, "longitude": 139.6993}},
    {"id": 8, "started_at": "2026-03-20T15:00:00Z", "ended_at": "2026-03-20T16:00:00Z",
     "name": "道頓堀", "status": "suggested", "confidence": 0.7,
     "place": {"id": 4, "latitude": 34.6687, "longitude": 135.5013}},
    {"id": 9, "started_at": "2026-03-20T17:00:00Z", "ended_at": "2026-03-20T17:20:00Z",
     "name": "Declined stop", "status": "declined", "confidence": 0.2,
     "place": {"id": 5, "latitude": 34.7, "longitude": 135.5}},
]

# --- Immich POST /api/search/metadata: AssetResponseDto items ---
ASSETS = [
    {"id": "asset-1", "checksum": "abc123", "type": "IMAGE",
     "fileCreatedAt": "2026-03-20T09:05:00.000Z", "localDateTime": "2026-03-20T18:05:00.000Z",
     "visibility": "timeline",
     "exifInfo": {"latitude": 35.6764, "longitude": 139.6993, "dateTimeOriginal": "2026-03-20T09:05:00.000Z",
                  "city": "Shibuya", "country": "Japan", "make": "Apple", "model": "iPhone 15",
                  "timeZone": "Asia/Tokyo"}},
    {"id": "asset-2", "checksum": "def456", "type": "IMAGE",
     "fileCreatedAt": "2026-03-20T12:00:00.000Z", "localDateTime": "2026-03-20T21:00:00.000Z",
     "visibility": "timeline",
     "exifInfo": {"latitude": None, "longitude": None, "dateTimeOriginal": None, "timeZone": None}},
]


def page_of(items, page):
    start = (page - 1) * PAGE_SIZE
    chunk = items[start:start + PAGE_SIZE]
    total_pages = max(1, (len(items) + PAGE_SIZE - 1) // PAGE_SIZE)
    return chunk, total_pages


class Handler(BaseHTTPRequestHandler):
    def _json(self, code, payload, headers=None):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        for k, v in (headers or {}).items():
            self.send_header(k, v)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _dawarich_authed(self, q):
        return q.get("api_key", [""])[0] == API_KEY or self.headers.get("Authorization") == f"Bearer {API_KEY}"

    def do_GET(self):
        u = urlparse(self.path)
        q = parse_qs(u.query)
        if u.path.startswith("/api/v1/") and not self._dawarich_authed(q):
            return self._json(401, {"error": "unauthorized"})
        page = int(q.get("page", ["1"])[0])
        if u.path == "/api/v1/tracks":
            chunk, tp = page_of(TRACKS, page)
            return self._json(200, {"type": "FeatureCollection", "features": chunk}, {"X-Total-Pages": str(tp)})
        if u.path == "/api/v1/visits":
            chunk, tp = page_of(VISITS, page)
            return self._json(200, chunk, {"X-Total-Pages": str(tp)})
        self._json(404, {"error": "not found"})

    def do_POST(self):
        u = urlparse(self.path)
        if u.path == "/api/search/metadata":
            if self.headers.get("x-api-key") != API_KEY:
                return self._json(401, {"error": "unauthorized"})
            length = int(self.headers.get("Content-Length", "0"))
            body = json.loads(self.rfile.read(length) or "{}")
            page = int(body.get("page", 1))
            chunk, tp = page_of(ASSETS, page)
            next_page = str(page + 1) if page < tp else None
            return self._json(200, {"assets": {"items": chunk, "total": len(ASSETS),
                                               "count": len(chunk), "nextPage": next_page}})
        self._json(404, {"error": "not found"})

    def log_message(self, *args):  # keep the console quiet
        pass


def main():
    port = int(os.getenv("MOCK_PORT", "8099"))
    print(f"mock upstream (Dawarich + Immich) on http://127.0.0.1:{port}  api_key={API_KEY}", flush=True)
    ThreadingHTTPServer(("127.0.0.1", port), Handler).serve_forever()


if __name__ == "__main__":
    main()

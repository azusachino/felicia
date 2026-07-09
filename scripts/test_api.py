#!/usr/bin/env python3
import json
import urllib.request
import urllib.error
import sys

BASE_URL = "http://localhost:8080"

def request(path, method="GET", data=None):
    url = f"{BASE_URL}{path}"
    headers = {"Content-Type": "application/json"}
    req_data = None
    if data is not None:
        req_data = json.dumps(data).encode("utf-8")
    
    req = urllib.request.Request(url, data=req_data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            body = resp.read().decode("utf-8")
            return resp.status, json.loads(body) if body else {}
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")
        try:
            res_body = json.loads(body) if body else {}
        except Exception:
            res_body = {"error": body}
        return e.code, res_body

def test_templates():
    print("Testing GET /api/admin/templates...")
    status, body = request("/api/admin/templates")
    assert status == 200, f"Expected 200, got {status}"
    assert "transit" in body, "Missing transit template"
    assert "live" in body, "Missing live template"
    assert "goods" in body, "Missing goods template"
    assert "receipt" in body, "Missing receipt template"
    assert "stamp" in body, "Missing stamp template"
    print("✓ Templates OK")

def test_journeys():
    print("Testing journeys endpoints...")
    # List
    status, body = request("/api/admin/journeys")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) > 0, "Expected at least 1 seeded journey"
    
    journey = body[0]
    journey_id = journey["id"]
    
    # Get by ID
    status, body = request(f"/api/admin/journeys/{journey_id}")
    assert status == 200, f"Expected 200, got {status}"
    assert body["slug"] == "japan-spring-2026", f"Unexpected slug: {body['slug']}"
    print("✓ Journeys GET OK")

def test_mementos_list():
    print("Testing GET /api/admin/journeys/{id}/mementos...")
    journey_id = "11111111-1111-1111-1111-111111111111"
    status, body = request(f"/api/admin/journeys/{journey_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == 5, f"Expected 5 seeded mementos, got {len(body)}"
    
    kinds = [m["kind"] for m in body]
    for k in ["transit", "receipt", "live", "goods", "stamp"]:
        assert k in kinds, f"Missing kind {k} in seeded mementos"
    print("✓ Mementos List OK")

def test_memento_validation():
    print("Testing memento validation endpoint...")
    # A. Test invalid memento POST (missing fields)
    payload_invalid = {
        "id": "99999999-9999-9999-9999-999999999999",
        "journey_id": "11111111-1111-1111-1111-111111111111",
        "kind": "live",
        "seq": 6,
        "occurred_at": "2026-03-20T10:00:00Z",
        "occurred_tz": "Asia/Tokyo",
        "title": "Invalid Live",
        "place": "Tokyo",
        "kind_data": {
            "artist": "羊文学" # missing required 'venue' and 'date'
        }
    }
    status, body = request("/api/admin/mementos", method="POST", data=payload_invalid)
    assert status == 400, f"Expected 400 Bad Request, got {status}"
    assert body.get("error") == "validation failed", f"Unexpected error msg: {body}"
    issues = [iss["Field"] for iss in body.get("issues", [])]
    assert "venue" in issues, "Expected validation issue for venue"
    assert "date" in issues, "Expected validation issue for date"
    print("✓ Memento validation rejected bad input correctly")

    # B. Test valid memento POST
    payload_valid = {
        "id": "88888888-8888-8888-8888-888888888888",
        "journey_id": "11111111-1111-1111-1111-111111111111",
        "kind": "live",
        "seq": 7,
        "occurred_at": "2026-03-20T10:00:00Z",
        "occurred_tz": "Asia/Tokyo",
        "geom": {
            "type": "Point",
            "coordinates": [139.7495, 35.6933]
        },
        "title": "Valid Live",
        "place": "Tokyo",
        "kind_data": {
            "artist": "羊文学",
            "venue": {
                "name": "日本武道館",
                "coords": [139.7495, 35.6933]
            },
            "date": "2026-03-22T18:30:00+09:00"
        }
    }
    status, body = request("/api/admin/mementos", method="POST", data=payload_valid)
    assert status == 200, f"Expected 200 OK, got {status} ({body})"
    assert body.get("status") == "ok", "Expected status ok response"
    print("✓ Memento validation accepted valid input correctly")

def test_translations():
    print("Testing translation endpoints...")
    memento_id = "22222222-2222-2222-2222-222222222222"
    # List
    status, body = request(f"/api/admin/mementos/{memento_id}/translations")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) > 0, "Expected at least 1 translation sidecar row"
    
    # Upsert
    new_trans = {
        "id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
        "owner_type": "memento",
        "owner_id": memento_id,
        "lang": "en",
        "field": "place",
        "value": "Tokyo Station (Translated)",
        "provenance": "authored"
    }
    status, body = request("/api/admin/translations", method="POST", data=new_trans)
    assert status == 200, f"Expected 200 OK, got {status}"
    print("✓ Translations OK")

def main():
    print("Starting API E2E test suite...")
    try:
        test_templates()
        test_journeys()
        test_mementos_list()
        test_memento_validation()
        test_translations()
        print("🎉 All API E2E tests passed successfully!")
    except AssertionError as e:
        print(f"❌ Test assertion failed: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"❌ Unexpected error running tests: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()

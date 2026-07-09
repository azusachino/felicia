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
    assert len(body) == 3, f"Expected 3 seeded journeys, got {len(body)}"
    
    slugs = [j["slug"] for j in body]
    assert "japan-spring-2026" in slugs, "Missing J1 slug"
    assert "hokkaido-winter-2025" in slugs, "Missing J2 slug"
    assert "tokyo-autumn-2024" in slugs, "Missing J3 slug"
    
    # Get by ID (J1)
    j1_id = "11111111-1111-1111-1111-111111111111"
    status, body = request(f"/api/admin/journeys/{j1_id}")
    assert status == 200, f"Expected 200, got {status}"
    assert body["slug"] == "japan-spring-2026", f"Unexpected slug: {body['slug']}"
    print("✓ Journeys GET OK")

def test_mementos_list():
    print("Testing GET /api/admin/journeys/{id}/mementos...")
    # J1: Should have 2 mementos
    j1_id = "11111111-1111-1111-1111-111111111111"
    status, body = request(f"/api/admin/journeys/{j1_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == 2, f"Expected 2 mementos for J1, got {len(body)}"
    assert body[0]["kind"] == "transit", "Expected J1 memento 1 to be transit"
    assert body[1]["kind"] == "receipt", "Expected J1 memento 2 to be receipt"

    # J2: Should have 2 mementos
    j2_id = "11111111-1111-1111-1111-222222222222"
    status, body = request(f"/api/admin/journeys/{j2_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == 2, f"Expected 2 mementos for J2, got {len(body)}"
    assert body[0]["kind"] == "live", "Expected J2 memento 1 to be live"
    assert body[1]["kind"] == "goods", "Expected J2 memento 2 to be goods"

    # J3: Should have 1 memento (stamp on Tokyo Station - overlapping location!)
    j3_id = "11111111-1111-1111-1111-333333333333"
    status, body = request(f"/api/admin/journeys/{j3_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == 1, f"Expected 1 memento for J3, got {len(body)}"
    assert body[0]["kind"] == "stamp", "Expected J3 memento 1 to be stamp"
    print("✓ Mementos distribution and overlapping locations OK")

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
            "artist": "羊文学"
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
    
    # Comic-book upsert style
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

def test_public_apis():
    print("Testing public read-only query APIs (Valkey cached)...")
    # 1. Global list
    status, body = request("/api/v1/journeys")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == 3, f"Expected 3 public journeys, got {len(body)}"
    
    # 2. Get details by slug
    status, body_slug = request("/api/v1/journeys/japan-spring-2026")
    assert status == 200, f"Expected 200, got {status}"
    assert body_slug["slug"] == "japan-spring-2026"
    assert "gps_route" in body_slug
    
    # 3. Get details by UUID (dual lookup)
    j1_id = "11111111-1111-1111-1111-111111111111"
    status, body_uuid = request(f"/api/v1/journeys/{j1_id}")
    assert status == 200, f"Expected 200, got {status}"
    assert body_uuid["slug"] == "japan-spring-2026"
    
    # 4. Get mementos by slug
    status, mementos_slug = request("/api/v1/journeys/japan-spring-2026/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(mementos_slug) == 3, f"Expected 3 mementos, got {len(mementos_slug)}"
    
    # 5. Get mementos by UUID (dual lookup)
    status, mementos_uuid = request(f"/api/v1/journeys/{j1_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(mementos_uuid) == 3
    
    print("✓ Public APIs (slug & UUID lookup) OK")

def main():
    print("Starting API E2E test suite...")
    try:
        test_templates()
        test_journeys()
        test_mementos_list()
        test_memento_validation()
        test_translations()
        test_public_apis()
        print("🎉 All API E2E tests passed successfully!")
    except AssertionError as e:
        print(f"❌ Test assertion failed: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"❌ Unexpected error running tests: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()

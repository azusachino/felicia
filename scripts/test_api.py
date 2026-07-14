#!/usr/bin/env python3
import json
import urllib.request
import urllib.error
import sys
from pathlib import Path

BASE_URL = "http://localhost:8080"
DATA = json.loads((Path(__file__).with_name("data.json")).read_text())
EXPECTED_JOURNEYS = len(DATA["journeys"])
EXPECTED_MEMENTOS = len(DATA["journeys"][0]["mementos"])
EXPECTED_ADDITIONAL_MEMENTOS = 3

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
    assert len(body) == EXPECTED_JOURNEYS, f"Expected {EXPECTED_JOURNEYS} seeded journeys, got {len(body)}"
    
    slugs = [j["slug"] for j in body]
    assert "golden-route" in slugs, "Missing J1 slug"
    assert "hokkaido" in slugs, "Missing J2 slug"
    assert "kyushu" in slugs, "Missing J3 slug"
    
    # Get by ID (J1)
    j1_id = "0190cbde-f300-7000-8000-111111111111"
    status, body = request(f"/api/admin/journeys/{j1_id}")
    assert status == 200, f"Expected 200, got {status}"
    assert body["slug"] == "golden-route", f"Unexpected slug: {body['slug']}"
    print("✓ Journeys GET OK")

def test_mementos_list():
    print("Testing GET /api/admin/journeys/{id}/mementos...")
    # J1: Should have the canonical memento count
    j1_id = "0190cbde-f300-7000-8000-111111111111"
    status, body = request(f"/api/admin/journeys/{j1_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == EXPECTED_MEMENTOS, f"Expected {EXPECTED_MEMENTOS} mementos for J1, got {len(body)}"
    assert body[0]["kind"] == "transit", "Expected J1 memento 1 to be transit"
    assert body[1]["kind"] == "stamp", "Expected J1 memento 2 to be stamp"
    assert body[2]["kind"] == "receipt", "Expected J1 memento 3 to be receipt"

    # J2: Should have the canonical memento count
    j2_id = "0190cbde-f300-7000-8000-222222222222"
    status, body = request(f"/api/admin/journeys/{j2_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == EXPECTED_MEMENTOS, f"Expected {EXPECTED_MEMENTOS} mementos for J2, got {len(body)}"
    assert body[0]["kind"] == "transit", "Expected J2 memento 1 to be transit"
    assert body[1]["kind"] == "stamp", "Expected J2 memento 2 to be stamp"

    # J3: Should have the canonical memento count
    j3_id = "0190cbde-f300-7000-8000-333333333333"
    status, body = request(f"/api/admin/journeys/{j3_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == EXPECTED_MEMENTOS, f"Expected {EXPECTED_MEMENTOS} mementos for J3, got {len(body)}"
    assert body[0]["kind"] == "transit", "Expected J3 memento 1 to be transit"
    assert body[1]["kind"] == "stamp", "Expected J3 memento 2 to be stamp"
    print("✓ Mementos distribution and overlapping locations OK")

def test_memento_validation():
    print("Testing memento validation endpoint...")
    # A. Test invalid memento POST (missing fields)
    payload_invalid = {
        "id": "0190cbde-f300-7000-8000-a99999999999",
        "journey_id": "0190cbde-f300-7000-8000-111111111111",
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
        "id": "0190cbde-f300-7000-8000-a88888888888",
        "journey_id": "0190cbde-f300-7000-8000-111111111111",
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

def test_memento_lifecycle():
    print("Testing memento lifecycle and optimistic revision...")
    memento_id = "0190cbde-f300-7000-8000-b99999999999"
    journey_id = "0190cbde-f300-7000-8000-111111111111"

    draft = {
        "id": memento_id,
        "journey_id": journey_id,
        "kind": "live",
        "seq": 8,
        "state": "draft",
        "kind_data": {"artist": "羊文学"},
    }
    status, body = request("/api/admin/mementos", method="POST", data=draft)
    assert status == 200, f"Expected incomplete draft to save, got {status} ({body})"

    status, body = request(f"/api/admin/mementos/{memento_id}")
    assert status == 200, f"Expected draft GET 200, got {status}"
    assert body["state"] == "draft", f"Expected draft state, got {body.get('state')}"
    assert body["revision"] == 1, f"Expected initial revision 1, got {body.get('revision')}"

    incomplete_publish = dict(draft)
    incomplete_publish["state"] = "published"
    incomplete_publish["expected_revision"] = 1
    status, body = request("/api/admin/mementos", method="POST", data=incomplete_publish)
    assert status == 400, f"Expected incomplete publish 400, got {status}"
    codes = [issue["Code"] for issue in body.get("issues", [])]
    assert "required_missing" in codes, f"Expected required_missing issue, got {body}"

    published = {
        **draft,
        "state": "published",
        "expected_revision": 1,
        "occurred_at": "2026-03-20T10:00:00Z",
        "occurred_tz": "Asia/Tokyo",
        "geom": {"type": "Point", "coordinates": [139.7495, 35.6933]},
        "kind_data": {
            "artist": "羊文学",
            "venue": {"name": "日本武道館", "coords": [139.7495, 35.6933]},
            "date": "2026-03-22T18:30:00+09:00",
        },
    }
    status, body = request("/api/admin/mementos", method="POST", data=published)
    assert status == 200, f"Expected complete publish 200, got {status} ({body})"

    status, body = request(f"/api/admin/mementos/{memento_id}")
    assert status == 200, f"Expected published GET 200, got {status}"
    assert body["state"] == "published", f"Expected published state, got {body.get('state')}"
    assert body["revision"] == 2, f"Expected revision 2, got {body.get('revision')}"
    print("✓ Memento lifecycle and optimistic revision OK")

def test_memento_revision_conflict():
    print("Testing optimistic concurrency and stale revision conflict...")
    memento_id = "0190cbde-f300-7000-8000-b99999999999"
    status, body = request(f"/api/admin/mementos/{memento_id}")
    assert status == 200, f"Expected lifecycle memento GET 200, got {status}"
    revision = body["revision"]

    update = {
        "id": memento_id,
        "journey_id": "0190cbde-f300-7000-8000-111111111111",
        "kind": "live",
        "seq": 8,
        "state": "published",
        "expected_revision": revision,
        "occurred_at": "2026-03-20T10:00:00Z",
        "occurred_tz": "Asia/Tokyo",
        "geom": {"type": "Point", "coordinates": [139.7495, 35.6933]},
        "title": "Live show, revised",
        "place": "Tokyo",
        "kind_data": {
            "artist": "羊文学",
            "venue": {"name": "日本武道館", "coords": [139.7495, 35.6933]},
            "date": "2026-03-22T18:30:00+09:00",
        },
    }
    status, body = request("/api/admin/mementos", method="POST", data=update)
    assert status == 200, f"Expected revision update 200, got {status} ({body})"

    status, body = request(f"/api/admin/mementos/{memento_id}")
    assert status == 200, f"Expected updated memento GET 200, got {status}"
    assert body["revision"] == revision + 1, f"Expected revision {revision + 1}, got {body.get('revision')}"

    status, body = request("/api/admin/mementos", method="POST", data=update)
    assert status == 409, f"Expected stale revision 409, got {status} ({body})"
    assert body.get("error") == "memento was modified; reload before saving", f"Unexpected conflict body: {body}"

    unlocked = dict(update)
    unlocked.pop("expected_revision")
    unlocked["title"] = "Live show, unlocked update"
    status, body = request("/api/admin/mementos", method="POST", data=unlocked)
    assert status == 200, f"Expected update without lock 200, got {status} ({body})"
    print("✓ Optimistic concurrency and stale revision conflict OK")

def test_memento_boundary_validation():
    print("Testing memento geometry, timezone, and draft boundaries...")
    journey_id = "0190cbde-f300-7000-8000-111111111111"
    live_data = {
        "artist": "羊文学",
        "venue": {"name": "日本武道館", "coords": [139.7495, 35.6933]},
        "date": "2026-03-22T18:30:00+09:00",
    }

    invalid_coordinate = {
        "id": "0190cbde-f300-7000-8000-c11111111111",
        "journey_id": journey_id,
        "kind": "live",
        "seq": 9,
        "state": "published",
        "occurred_at": "2026-03-20T10:00:00Z",
        "occurred_tz": "Asia/Tokyo",
        "geom": {"type": "Point", "coordinates": [181, 35]},
        "kind_data": live_data,
    }
    status, body = request("/api/admin/mementos", method="POST", data=invalid_coordinate)
    assert status == 400, f"Expected invalid coordinate 400, got {status} ({body})"
    assert "invalid_coordinate" in [issue["Code"] for issue in body.get("issues", [])]

    wrong_anchor = dict(invalid_coordinate)
    wrong_anchor["id"] = "0190cbde-f300-7000-8000-c22222222222"
    wrong_anchor["geom"] = {
        "type": "LineString",
        "coordinates": [[139.7, 35.6], [139.8, 35.7]],
    }
    status, body = request("/api/admin/mementos", method="POST", data=wrong_anchor)
    assert status == 400, f"Expected anchor mismatch 400, got {status} ({body})"
    assert "anchor_mismatch" in [issue["Code"] for issue in body.get("issues", [])]

    invalid_timezone = dict(invalid_coordinate)
    invalid_timezone["id"] = "0190cbde-f300-7000-8000-c33333333333"
    invalid_timezone["geom"] = {"type": "Point", "coordinates": [139.7495, 35.6933]}
    invalid_timezone["occurred_tz"] = "not/a-timezone"
    status, body = request("/api/admin/mementos", method="POST", data=invalid_timezone)
    assert status == 400, f"Expected invalid timezone 400, got {status} ({body})"
    assert "invalid_timezone" in [issue["Code"] for issue in body.get("issues", [])]

    draft = {
        "id": "0190cbde-f300-7000-8000-c44444444444",
        "journey_id": journey_id,
        "kind": "live",
        "seq": 9,
        "state": "draft",
        "kind_data": {},
    }
    status, body = request("/api/admin/mementos", method="POST", data=draft)
    assert status == 200, f"Expected incomplete draft 200, got {status} ({body})"
    print("✓ Memento boundary validation OK")

def test_public_apis():
    print("Testing public read-only query APIs (Valkey cached)...")
    # 1. Global list
    status, body = request("/api/v1/journeys")
    assert status == 200, f"Expected 200, got {status}"
    assert len(body) == EXPECTED_JOURNEYS, f"Expected {EXPECTED_JOURNEYS} public journeys, got {len(body)}"
    
    # 2. Get details by slug
    status, body_slug = request("/api/v1/journeys/golden-route")
    assert status == 200, f"Expected 200, got {status}"
    assert body_slug["slug"] == "golden-route"
    assert "gps_route" in body_slug
    
    # 3. Get details by UUID (dual lookup)
    j1_id = "0190cbde-f300-7000-8000-111111111111"
    status, body_uuid = request(f"/api/v1/journeys/{j1_id}")
    assert status == 200, f"Expected 200, got {status}"
    assert body_uuid["slug"] == "golden-route"
    
    # 4. Get mementos by slug
    status, mementos_slug = request("/api/v1/journeys/golden-route/mementos")
    assert status == 200, f"Expected 200, got {status}"
    expected_mementos = EXPECTED_MEMENTOS + EXPECTED_ADDITIONAL_MEMENTOS
    assert len(mementos_slug) == expected_mementos, f"Expected {expected_mementos} mementos, got {len(mementos_slug)}"
    
    # 5. Get mementos by UUID (dual lookup)
    status, mementos_uuid = request(f"/api/v1/journeys/{j1_id}/mementos")
    assert status == 200, f"Expected 200, got {status}"
    assert len(mementos_uuid) == expected_mementos
    
    print("✓ Public APIs (slug & UUID lookup) OK")

def main():
    print("Starting API E2E test suite...")
    try:
        test_templates()
        test_journeys()
        test_mementos_list()
        test_memento_validation()
        test_memento_lifecycle()
        test_memento_revision_conflict()
        test_memento_boundary_validation()
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

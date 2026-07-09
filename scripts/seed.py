#!/usr/bin/env python3
import os
import sys
import psycopg

def main():
    dsn = os.getenv("DATABASE_DSN")
    if not not dsn:
        pass
    else:
        print("Error: DATABASE_DSN environment variable is required.", file=sys.stderr)
        sys.exit(1)

    print(f"Connecting to database to seed multi-trip dev data...")
    try:
        with psycopg.connect(dsn) as conn:
            with conn.cursor() as cur:
                # 1. Clean up existing data (cascading deletes everything)
                print("Cleaning up old data...")
                cur.execute("TRUNCATE TABLE journal, translations CASCADE")

                # 2. Insert Root Journal (UUIDv7)
                journal_id = "0190cbde-f300-7000-8000-000000000000"
                print(f"Seeding Journal: {journal_id}")
                cur.execute(
                    "INSERT INTO journal (id, created_at) VALUES (%s, NOW())",
                    (journal_id,)
                )

                # 3. Seed Three Journeys across three years with overlapping and new locations (UUIDv7)
                # Journey 1: Japan Spring 2026 (Tokyo & Kyoto)
                journey_1_id = "0190cbde-f300-7000-8000-111111111111"
                print(f"Seeding Journey 1 (2026, Tokyo/Kyoto): {journey_1_id}")
                cur.execute(
                    """
                    INSERT INTO journeys (
                        id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('MULTILINESTRING((139.7671 35.6812, 139.6921 35.6896, 135.7583 34.9859))', 4326),
                        %s
                    )
                    """,
                    (
                        journey_1_id,
                        journal_id,
                        "japan-spring-2026",
                        "immich-album:japan2026",
                        "日本春旅 2026",
                        "東京 & 京都",
                        "JPN",
                        "関東・関西",
                        "2026-03-20",
                        "2026-04-05",
                        []
                    )
                )

                # Journey 2: Hokkaido Winter 2025 (Sapporo & Otaru)
                journey_2_id = "0190cbde-f300-7000-8000-222222222222"
                print(f"Seeding Journey 2 (2025, Hokkaido): {journey_2_id}")
                cur.execute(
                    """
                    INSERT INTO journeys (
                        id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('MULTILINESTRING((141.3545 43.0620, 140.9947 43.1907))', 4326),
                        %s
                    )
                    """,
                    (
                        journey_2_id,
                        journal_id,
                        "hokkaido-winter-2025",
                        "immich-album:hokkaido2025",
                        "北海道冬旅 2025",
                        "札幌 & 小樽",
                        "JPN",
                        "北海道",
                        "2025-01-10",
                        "2025-01-17",
                        []
                    )
                )

                # Journey 3: Tokyo Autumn 2024 (Tokyo - overlapping location with J1!)
                journey_3_id = "0190cbde-f300-7000-8000-333333333333"
                print(f"Seeding Journey 3 (2024, Tokyo): {journey_3_id}")
                cur.execute(
                    """
                    INSERT INTO journeys (
                        id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('MULTILINESTRING((139.7671 35.6812, 139.7003 35.6895))', 4326),
                        %s
                    )
                    """,
                    (
                        journey_3_id,
                        journal_id,
                        "tokyo-autumn-2024",
                        "immich-album:tokyo2024",
                        "東京秋旅 2024",
                        "東京",
                        "JPN",
                        "関東",
                        "2024-11-01",
                        "2024-11-05",
                        []
                    )
                )

                # 4. Seed Mementos distributed across journeys (UUIDv7)
                # Memento 1 (J1 - Transit, 2026): Tokyo -> Kyoto
                memento_1_id = "0190cbde-f300-7000-8000-a22222222222"
                print(f"Seeding Memento 1 (Transit, J1): {memento_1_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(139.7671 35.6812)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_1_id,
                        journey_1_id,
                        "transit",
                        1,
                        "2026-03-20 10:00:00+0900",
                        "Asia/Tokyo",
                        "JR 東京駅 乗車券",
                        "東京駅",
                        "JR East",
                        "東京から京都への旅立ち。",
                        14000,
                        "JPY",
                        '{"operator": "JR East", "line": "Tokaido Shinkansen", "from": {"name": "Tokyo", "coords": [139.7671, 35.6812]}, "to": {"name": "Kyoto", "coords": [135.7583, 34.9859]}}',
                        "immich-photo:ticket-pic",
                        []
                    )
                )

                # Memento 2 (J1 - Receipt, 2026): Kyoto Cafe
                memento_2_id = "0190cbde-f300-7000-8000-a33333333333"
                print(f"Seeding Memento 2 (Receipt, J1): {memento_2_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(135.7583 34.9859)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_2_id,
                        journey_1_id,
                        "receipt",
                        2,
                        "2026-03-21 15:30:00+0900",
                        "Asia/Tokyo",
                        "スマートコーヒー レシート",
                        "京都",
                        "Smart Coffee",
                        "フレンチトーストと自家焙煎珈琲をいただく。",
                        1200,
                        "JPY",
                        '{"shop": "Smart Coffee", "total": {"amount": 1200, "currency": "JPY"}, "items": "French Toast & Coffee"}',
                        "immich-photo:coffee-pic",
                        []
                    )
                )

                # Memento 3 (J2 - Live, 2025): Sapporo Concert
                memento_3_id = "0190cbde-f300-7000-8000-a44444444444"
                print(f"Seeding Memento 3 (Live, J2): {memento_3_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(141.3545 43.0620)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_3_id,
                        journey_2_id,
                        "live",
                        1,
                        "2025-01-12 18:30:00+0900",
                        "Asia/Tokyo",
                        "羊文学 札幌コンサート",
                        "札幌市",
                        "Sapporo Hall",
                        "札幌のホールでのライブ。素晴らしい演奏だった。",
                        7500,
                        "JPY",
                        '{"artist": "羊文学", "venue": {"name": "Sapporo Hall", "coords": [141.3545, 43.0620]}, "date": "2025-01-12T18:30:00+09:00", "seat": "Row B-2"}',
                        "immich-photo:sapporo-live",
                        []
                    )
                )

                # Memento 4 (J2 - Goods, 2025): Hokkaido Souvenirs
                memento_4_id = "0190cbde-f300-7000-8000-a55555555555"
                print(f"Seeding Memento 4 (Goods, J2): {memento_4_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(140.9947 43.1907)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_4_id,
                        journey_2_id,
                        "goods",
                        2,
                        "2025-01-14 14:00:00+0900",
                        "Asia/Tokyo",
                        "小樽ガラス工芸品",
                        "小樽",
                        "Otaru Glass Shop",
                        "美しいガラスペンを購入。",
                        5500,
                        "JPY",
                        '{"name": "Glass Pen", "shop": "Otaru Glass Shop", "price": {"amount": 5500, "currency": "JPY"}, "manufacturer": "Otaru Craft"}',
                        "immich-photo:glass-pen",
                        []
                    )
                )

                # Memento 5 (J3 - Stamp, 2024): Tokyo Station Stamp (overlapping location with J1!)
                memento_5_id = "0190cbde-f300-7000-8000-a66666666666"
                print(f"Seeding Memento 5 (Stamp, J3): {memento_5_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(139.7671 35.6812)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_5_id,
                        journey_3_id,
                        "stamp",
                        1,
                        "2024-11-02 11:00:00+0900",
                        "Asia/Tokyo",
                        "東京駅 記念スタンプ",
                        "東京駅",
                        "JR Tokyo Station",
                        "東京駅丸の内改札口で記念の駅スタンプを押した。",
                        0,
                        "JPY",
                        '{"name": "東京駅 記念スタンプ", "shrine_or_temple": "JR Tokyo Station", "deity": "Marunouchi Station Building"}',
                        "immich-photo:station-stamp",
                        []
                    )
                )

                # 5. Insert Photos for Memento 1 (UUIDv7)
                photo_id = "0190cbde-f300-7000-8000-f77777777777"
                print(f"Seeding Memento Photo: {photo_id}")
                cur.execute(
                    """
                    INSERT INTO memento_photos (
                        id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    (
                        photo_id,
                        memento_1_id,
                        "media/photos/tokyo_ticket.jpg",
                        "sha256:ticket-photo-hash-12345",
                        "新幹線の切符",
                        1,
                        "2026-03-20 09:55:00+0900",
                        "immich-photo:ticket-pic"
                    )
                )

                # 6. Insert Translation Sidecars (UUIDv7)
                print("Seeding Translations...")
                # English journey names
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        generate_uuid_v7(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("journey", journey_1_id, "en", "title", "Japan Spring Journey 2026", "machine")
                )
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        generate_uuid_v7(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("journey", journey_2_id, "en", "title", "Hokkaido Winter Trip 2025", "machine")
                )
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        generate_uuid_v7(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("journey", journey_3_id, "en", "title", "Tokyo Autumn Tour 2024", "machine")
                )
                # English memento names
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        generate_uuid_v7(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("memento", memento_1_id, "en", "title", "JR Tokyo Station Ticket", "machine")
                )
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        generate_uuid_v7(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("memento", memento_5_id, "en", "title", "Tokyo Station Commemorative Stamp", "machine")
                )

                # 6. Seed a transit leg on Journey 1 (HND -> KIX flight). Kept
                # separate from gps_route; the geodesic arc is built by
                # ST_Segmentize and composed into the display route at read time
                # (union-at-read). This gives the dev DB a journey with both a
                # Dawarich track AND an authored leg; Journeys 2 & 3 stay track-only.
                print("Seeding transit leg on Journey 1 (HND -> KIX)...")
                cur.execute(
                    """
                    INSERT INTO transit_legs (
                        id, journey_id, seq, origin_label, dest_label, geom
                    ) VALUES (
                        generate_uuid_v7(), %s, %s, %s, %s,
                        ST_Segmentize(
                            ST_MakeLine(
                                ST_SetSRID(ST_MakePoint(139.7798, 35.5494), 4326),
                                ST_SetSRID(ST_MakePoint(135.2381, 34.4342), 4326)
                            )::geography,
                            100000
                        )::geometry
                    )
                    """,
                    (journey_1_id, 0, "HND", "KIX")
                )

                conn.commit()
                print("Database successfully seeded with multi-trip UUIDv7 dataset!")
    except Exception as e:
        print(f"Error seeding database: {e}", file=sys.stderr)
        sys.exit(2)

if __name__ == "__main__":
    main()

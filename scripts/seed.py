#!/usr/bin/env python3
import os
import sys
import psycopg

def main():
    dsn = os.getenv("DATABASE_DSN")
    if not dsn:
        print("Error: DATABASE_DSN environment variable is required.", file=sys.stderr)
        sys.exit(1)

    print(f"Connecting to database to seed data...")
    try:
        with psycopg.connect(dsn) as conn:
            with conn.cursor() as cur:
                # 1. Clean up existing data (cascading deletes everything)
                print("Cleaning up old data...")
                cur.execute("TRUNCATE TABLE journal, translations CASCADE")

                # 2. Insert Root Journal
                journal_id = "00000000-0000-0000-0000-000000000000"
                print(f"Seeding Journal: {journal_id}")
                cur.execute(
                    "INSERT INTO journal (id, created_at) VALUES (%s, NOW())",
                    (journal_id,)
                )

                # 3. Insert Journey
                journey_id = "11111111-1111-1111-1111-111111111111"
                print(f"Seeding Journey: {journey_id}")
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
                        journey_id,
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

                # 4. Insert Mementos
                # Memento 1: Transit Ticket
                memento_1_id = "22222222-2222-2222-2222-222222222222"
                print(f"Seeding Memento 1 (Transit): {memento_1_id}")
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
                        journey_id,
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

                # Memento 2: Cafe Receipt
                memento_2_id = "33333333-3333-3333-3333-333333333333"
                print(f"Seeding Memento 2 (Receipt): {memento_2_id}")
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
                        journey_id,
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

                # Memento 3: Live Event Ticket
                memento_3_id = "44444444-4444-4444-4444-444444444444"
                print(f"Seeding Memento 3 (Live): {memento_3_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(139.7495 35.6933)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_3_id,
                        journey_id,
                        "live",
                        3,
                        "2026-03-22 18:30:00+0900",
                        "Asia/Tokyo",
                        "羊文学 日本武道館ライブ",
                        "日本武道館",
                        "Budokan",
                        "日本武道館でのライブ。素晴らしい演奏だった。",
                        7500,
                        "JPY",
                        '{"artist": "羊文学", "venue": {"name": "日本武道館", "coords": [139.7495, 35.6933]}, "date": "2026-03-22T18:30:00+09:00", "seat": "Arena A-10"}',
                        "immich-photo:live-pic",
                        []
                    )
                )

                # Memento 4: Goods / Souvenirs
                memento_4_id = "55555555-5555-5555-5555-555555555555"
                print(f"Seeding Memento 4 (Goods): {memento_4_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(139.7003 35.6895)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_4_id,
                        journey_id,
                        "goods",
                        4,
                        "2026-03-25 14:00:00+0900",
                        "Asia/Tokyo",
                        "アニメイト キャラクターグッズ",
                        "新宿",
                        "Animate",
                        "お気に入りのアニメのグッズを購入。",
                        3500,
                        "JPY",
                        '{"name": "Fuwamiku Plush", "shop": "Animate Shinjuku", "price": {"amount": 3500, "currency": "JPY"}, "manufacturer": "Good Smile Company"}',
                        "immich-photo:goods-pic",
                        []
                    )
                )

                # Memento 5: Goshuin Stamp
                memento_5_id = "66666666-6666-6666-6666-666666666666"
                print(f"Seeding Memento 5 (Stamp): {memento_5_id}")
                cur.execute(
                    """
                    INSERT INTO mementos (
                        id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        ST_GeomFromText('POINT(135.7839 34.9948)', 4326),
                        %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    """,
                    (
                        memento_5_id,
                        journey_id,
                        "stamp",
                        5,
                        "2026-03-28 11:00:00+0900",
                        "Asia/Tokyo",
                        "清水寺 御朱印",
                        "京都 清水寺",
                        "清水寺",
                        "京都清水寺参拝の際に御朱印をいただく。",
                        500,
                        "JPY",
                        '{"name": "清水寺 御朱印", "shrine_or_temple": "清水寺", "deity": "十一面千手観世音菩薩"}',
                        "immich-photo:stamp-pic",
                        []
                    )
                )

                # 5. Insert Photos
                photo_id = "77777777-7777-7777-7777-777777777777"
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

                # 6. Insert Translation Sidecars
                print("Seeding Translations...")
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        gen_random_uuid(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("journey", journey_id, "en", "title", "Japan Spring Journey 2026", "machine")
                )
                cur.execute(
                    """
                    INSERT INTO translations (
                        id, owner_type, owner_id, lang, field, value, provenance, updated_at
                    ) VALUES (
                        gen_random_uuid(), %s, %s, %s, %s, %s, %s, NOW()
                    )
                    """,
                    ("memento", memento_1_id, "en", "title", "JR Tokyo Station Ticket", "machine")
                )

                conn.commit()
                print("Database seeded successfully with all 5 kinds!")
    except Exception as e:
        print(f"Error seeding database: {e}", file=sys.stderr)
        sys.exit(2)

if __name__ == "__main__":
    main()

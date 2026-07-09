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
                        [] # empty authored fields
                    )
                )

                # 4. Insert Mementos
                memento_1_id = "22222222-2222-2222-2222-222222222222"
                print(f"Seeding Memento 1 (Ticket): {memento_1_id}")
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
                        "ticket",
                        1,
                        "2026-03-20 10:00:00+0900",
                        "Asia/Tokyo",
                        "JR 東京駅 乗車券",
                        "東京駅",
                        "JR East",
                        "東京から京都への旅立ち。新幹線の切符を購入し、のぞみ号に乗車した。",
                        14000,
                        "JPY",
                        '{"operator": "JR East", "line": "Tokaido Shinkansen", "from": "Tokyo", "to": "Kyoto"}',
                        "immich-photo:ticket-pic",
                        []
                    )
                )

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
                        "スマートコーヒー 喫茶レシート",
                        "京都",
                        "Smart Coffee",
                        "京都寺町通りの老舗喫茶店。フレンチトーストと自家焙煎珈琲をいただく。最高の時間だった。",
                        1200,
                        "JPY",
                        '{"shop": "Smart Coffee", "items": ["French Toast", "Coffee"]}',
                        "immich-photo:coffee-pic",
                        []
                    )
                )

                # 5. Insert Photos
                photo_id = "44444444-4444-4444-4444-444444444444"
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
                print("Database seeded successfully!")
    except Exception as e:
        print(f"Error seeding database: {e}", file=sys.stderr)
        sys.exit(2)

if __name__ == "__main__":
    main()

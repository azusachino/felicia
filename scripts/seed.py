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

                # 3. Seed Three Journeys across three years (UUIDv7)
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
                # Journey 1 Mementos (Transit, Receipt, Souvenir)
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

                memento_3_id = "0190cbde-f300-7000-8000-aaaaa8888888"
                print(f"Seeding Memento 3 (Souvenir, J1): {memento_3_id}")
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
                        memento_3_id,
                        journey_1_id,
                        "souvenir",
                        3,
                        "2026-03-22 10:00:00+0900",
                        "Asia/Tokyo",
                        "清水寺のお守り",
                        "京都",
                        "清水寺",
                        "清水寺を参拝し、開運の木札お守りを授かった。静かな朝の境内の匂いがする。",
                        800,
                        "JPY",
                        '{"shrine_or_temple": "Kiyomizu-dera", "item": "Amulet"}',
                        "immich-photo:kiyomizu-amulet",
                        []
                    )
                )

                # Journey 2 Mementos (Live, Goods, Receipt)
                memento_4_id = "0190cbde-f300-7000-8000-a44444444444"
                print(f"Seeding Memento 4 (Live, J2): {memento_4_id}")
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
                        memento_4_id,
                        journey_2_id,
                        "live",
                        1,
                        "2025-01-12 18:30:00+0900",
                        "Asia/Tokyo",
                        "羊文学 札幌コンサート",
                        "札幌市",
                        "Sapporo Hall",
                        "札幌のホールでのライブ。吹雪の夜だったが、熱気あふれる素晴らしい演奏だった。",
                        7500,
                        "JPY",
                        '{"artist": "羊文学", "venue": {"name": "Sapporo Hall", "coords": [141.3545, 43.0620]}, "date": "2025-01-12T18:30:00+09:00", "seat": "Row B-2"}',
                        "immich-photo:sapporo-live",
                        []
                    )
                )

                memento_5_id = "0190cbde-f300-7000-8000-a55555555555"
                print(f"Seeding Memento 5 (Goods, J2): {memento_5_id}")
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
                        memento_5_id,
                        journey_2_id,
                        "goods",
                        2,
                        "2025-01-14 14:00:00+0900",
                        "Asia/Tokyo",
                        "小樽ガラス工芸品",
                        "小樽",
                        "Otaru Glass Shop",
                        "運河沿いの美しい店で手作りのガラスペンを購入。",
                        5500,
                        "JPY",
                        '{"name": "Glass Pen", "shop": "Otaru Glass Shop", "price": {"amount": 5500, "currency": "JPY"}, "manufacturer": "Otaru Craft"}',
                        "immich-photo:glass-pen",
                        []
                    )
                )

                memento_6_id = "0190cbde-f300-7000-8000-a99999999999"
                print(f"Seeding Memento 6 (Receipt, J2): {memento_6_id}")
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
                        memento_6_id,
                        journey_2_id,
                        "receipt",
                        3,
                        "2025-01-15 19:30:00+0900",
                        "Asia/Tokyo",
                        "サッポロビール園 レシート",
                        "札幌市",
                        "サッポロビール園",
                        "冷えたビールと焼き立てのジンギスカンを囲む夕食。",
                        4500,
                        "JPY",
                        '{"shop": "Sapporo Beer Garden", "total": {"amount": 4500, "currency": "JPY"}, "items": "Genghis Khan & Draft Beer"}',
                        "immich-photo:beer-garden",
                        []
                    )
                )

                # Journey 3 Mementos (Stamp, Goods)
                memento_7_id = "0190cbde-f300-7000-8000-a66666666666"
                print(f"Seeding Memento 7 (Stamp, J3): {memento_7_id}")
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
                        memento_7_id,
                        journey_3_id,
                        "stamp",
                        1,
                        "2024-11-02 11:00:00+0900",
                        "Asia/Tokyo",
                        "東京駅 記念スタンプ",
                        "東京駅",
                        "JR Tokyo Station",
                        "東京駅丸の内改札口で記念の駅スタンプを押した。インクのにじみが良い味を出している。",
                        0,
                        "JPY",
                        '{"name": "東京駅 記念スタンプ", "shrine_or_temple": "JR Tokyo Station", "deity": "Marunouchi Station Building"}',
                        "immich-photo:station-stamp",
                        []
                    )
                )

                memento_8_id = "0190cbde-f300-7000-8000-aaaaa7777777"
                print(f"Seeding Memento 8 (Goods, J3): {memento_8_id}")
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
                        memento_8_id,
                        journey_3_id,
                        "goods",
                        2,
                        "2024-11-03 14:00:00+0900",
                        "Asia/Tokyo",
                        "秋葉原 レトロゲーム",
                        "東京",
                        "スーパーポテト",
                        "懐かしのファミリーコンピュータカセットを購入。箱のデザインが秀逸。",
                        3200,
                        "JPY",
                        '{"item": "Famicom Cartridge", "shop": "Super Potato", "price": {"amount": 3200, "currency": "JPY"}}',
                        "immich-photo:retro-game",
                        []
                    )
                )

                # 5. Insert Photos for Mementos (Multi-media support!)
                # Memento 1 Photos (2 photos)
                photo_ids = [
                    ("0190cbde-f300-7000-8000-f77777777777", memento_1_id, "media/photos/tokyo_ticket.jpg", "新幹線の切符", 1, "2026-03-20 09:55:00+0900", "immich-photo:ticket-pic"),
                    ("0190cbde-f300-7000-8000-f77777777778", memento_1_id, "media/photos/tokyo_station_building.jpg", "東京駅丸の内駅舎", 2, "2026-03-20 09:56:00+0900", "immich-photo:station-building-pic")
                ]
                # Memento 2 Photos (2 photos)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-f88888888881", memento_2_id, "media/photos/coffee_cup.jpg", "自家焙煎のスマートコーヒー", 1, "2026-03-21 15:35:00+0900", "immich-photo:coffee-cup-pic"),
                    ("0190cbde-f300-7000-8000-f88888888882", memento_2_id, "media/photos/french_toast.jpg", "名物のふっくらフレンチトースト", 2, "2026-03-21 15:40:00+0900", "immich-photo:toast-pic")
                ]
                # Memento 3 Photos (2 photos)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-f99999999991", memento_3_id, "media/photos/omamori_front.jpg", "授かった開運の木札お守り", 1, "2026-03-22 10:15:00+0900", "immich-photo:omamori-pic"),
                    ("0190cbde-f300-7000-8000-f99999999992", memento_3_id, "media/photos/kiyomizudera_stage.jpg", "清々しい朝の清水の舞台", 2, "2026-03-22 10:20:00+0900", "immich-photo:stage-pic")
                ]
                # Memento 4 Photos (1 photo)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-faaaaaaa1111", memento_4_id, "media/photos/sapporo_hall.jpg", "ライブが行われた札幌コンサートホール外観", 1, "2025-01-12 18:25:00+0900", "immich-photo:hall-pic")
                ]
                # Memento 5 Photos (2 photos)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-faaaaaaa2221", memento_5_id, "media/photos/glass_pen.jpg", "手作りのガラスペン", 1, "2025-01-14 14:05:00+0900", "immich-photo:glass-pen-pic"),
                    ("0190cbde-f300-7000-8000-faaaaaaa2222", memento_5_id, "media/photos/otaru_canal.jpg", "雪化粧が施された美しい小樽運河", 2, "2025-01-14 14:10:00+0900", "immich-photo:canal-pic")
                ]
                # Memento 6 Photos (1 photo)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-fbbbbbbb1111", memento_6_id, "media/photos/beer_and_lamb.jpg", "ジンギスカンとジョッキビール", 1, "2025-01-15 19:35:00+0900", "immich-photo:beer-garden-pic")
                ]
                # Memento 7 Photos (1 photo)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-fccccccc1111", memento_7_id, "media/photos/tokyo_station_stamp.jpg", "スタンプ帳に押した丸の内改札スタンプ", 1, "2024-11-02 11:05:00+0900", "immich-photo:stamp-card-pic")
                ]
                # Memento 8 Photos (2 photos)
                photo_ids += [
                    ("0190cbde-f300-7000-8000-fddddddd1111", memento_8_id, "media/photos/retro_game_box.jpg", "レトロなパッケージアートのソフト", 1, "2024-11-03 14:10:00+0900", "immich-photo:retro-box-pic"),
                    ("0190cbde-f300-7000-8000-fddddddd1112", memento_8_id, "media/photos/akihabara_neon.jpg", "電気街の賑やかなネオンサイン", 2, "2024-11-03 14:15:00+0900", "immich-photo:akiba-neon-pic")
                ]

                for p_id, m_id, o_key, caption, seq, taken_at, s_ref in photo_ids:
                    print(f"Seeding Photo: {p_id} for Memento {m_id}")
                    cur.execute(
                        """
                        INSERT INTO memento_photos (
                            id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at
                        ) VALUES (
                            %s, %s, %s, %s, %s, %s, %s, %s, NOW()
                        )
                        """,
                        (p_id, m_id, o_key, f"sha256:{o_key}-hash", caption, seq, taken_at, s_ref)
                    )

                # 6. Insert Translation Sidecars (UUIDv7)
                print("Seeding Translations...")
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
                    ("memento", memento_7_id, "en", "title", "Tokyo Station Commemorative Stamp", "machine")
                )

                # 7. Seed transit legs on Journey 1 (HND -> KIX flight)
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
                print("Database successfully seeded with expanded multi-trip UUIDv7 dataset!")
    except Exception as e:
        print(f"Error seeding database: {e}", file=sys.stderr)
        sys.exit(2)

if __name__ == "__main__":
    main()

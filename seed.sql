-- ข้อมูลตั้งต้น 10 รายการ จาก src/data/data.js ของหน้าเว็บ (ราคาและคะแนนเป็นค่าตัวอย่าง)
-- รันหลัง schema.sql  รันซ้ำได้ (อัปเดตแถวเดิมตาม id)
-- สร้างใหม่จาก data.js ได้ด้วย: node scripts/make-seed.mjs path/to/data.js > seed.sql

BEGIN;

INSERT INTO equipment (id, title, brand, category_id, release_year, rating, price, emoji, detail, poster_url) VALUES
    (1, 'GeForce RTX 4060', 'NVIDIA', (SELECT id FROM categories WHERE name_th = 'การ์ดจอ'), 2023, 8, 10900, '🎮', 'การ์ดจอระดับกลางที่เน้นเล่นเกมความละเอียด 1080p ได้ลื่น รองรับ DLSS และ Ray Tracing กินไฟต่ำ เหมาะกับคนที่อยากประกอบเครื่องเล่นเกมในงบไม่สูงมาก', NULL),
    (2, 'Ryzen 5 7600', 'AMD', (SELECT id FROM categories WHERE name_th = 'ซีพียู'), 2022, 8.3, 7500, '🧠', 'ซีพียู 6 คอร์ 12 เธรด สถาปัตยกรรม Zen 4 ใช้ซ็อกเก็ต AM5 ประสิทธิภาพดีทั้งเล่นเกมและทำงานทั่วไป และมีกราฟิกในตัวสำหรับใช้งานเบื้องต้น', NULL),
    (3, 'Core i5-13400F', 'Intel', (SELECT id FROM categories WHERE name_th = 'ซีพียู'), 2023, 8, 6500, '🧠', 'ซีพียู 10 คอร์ (6P+4E) คุ้มราคา เหมาะกับการเล่นเกมและงานทั่วไป รุ่น F ไม่มีกราฟิกในตัว ต้องใช้คู่กับการ์ดจอแยก', NULL),
    (4, 'G102 LIGHTSYNC', 'Logitech', (SELECT id FROM categories WHERE name_th = 'เมาส์'), 2018, 7.8, 450, '🖱️', 'เมาส์เกมมิ่งราคาประหยัด เซนเซอร์แม่นยำ น้ำหนักเบา ปรับไฟ RGB ได้ เหมาะกับมือใหม่และงบจำกัด', NULL),
    (5, 'MX Master 3S', 'Logitech', (SELECT id FROM categories WHERE name_th = 'เมาส์'), 2022, 9, 3500, '🖱️', 'เมาส์สำหรับงานออฟฟิศและครีเอทีฟ จับถนัดมือ ล้อเลื่อนเงียบ เชื่อมต่อได้หลายอุปกรณ์ เหมาะกับคนที่ใช้คอมทั้งวัน', NULL),
    (6, 'K2', 'Keychron', (SELECT id FROM categories WHERE name_th = 'คีย์บอร์ด'), 2020, 8.4, 3200, '⌨️', 'คีย์บอร์ดแมคคานิคอลไร้สายขนาดกะทัดรัด 75% เชื่อมต่อบลูทูธได้หลายเครื่อง ใช้ได้ทั้ง Windows และ Mac เหมาะกับคนที่ชอบพิมพ์และพกพา', NULL),
    (7, 'UltraGear 27GP850', 'LG', (SELECT id FROM categories WHERE name_th = 'จอมอนิเตอร์'), 2021, 8.6, 13900, '🖥️', 'จอเกมมิ่ง 27 นิ้ว ความละเอียด 1440p แผง Nano IPS รีเฟรชเรต 165Hz สีสวยและตอบสนองเร็ว เหมาะกับทั้งเล่นเกมและงานกราฟิก', NULL),
    (8, 'Cloud II', 'HyperX', (SELECT id FROM categories WHERE name_th = 'หูฟัง'), 2015, 8.2, 3500, '🎧', 'หูฟังเกมมิ่งแบบครอบหู ใส่สบายแม้ใช้นาน เสียงชัด ไมค์ถอดได้ เป็นรุ่นที่ได้รับความนิยมมายาวนาน', NULL),
    (9, '970 EVO Plus 1TB', 'Samsung', (SELECT id FROM categories WHERE name_th = 'SSD'), 2019, 8.7, 2900, '💾', 'SSD NVMe ความเร็วสูง ความจุ 1TB เปิดเครื่องและโหลดเกมไว เหมาะใช้เป็นไดรฟ์หลัก มีชื่อเสียงด้านความทนทาน', NULL),
    (10, 'Fury Beast DDR5 16GB', 'Kingston', (SELECT id FROM categories WHERE name_th = 'แรม'), 2022, 8, 1900, '📟', 'แรม DDR5 สำหรับเมนบอร์ดรุ่นใหม่ ติดตั้งง่าย รองรับ XMP/EXPO เหมาะกับการอัปเกรดเครื่องเล่นเกมและทำงาน', NULL)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title, brand = EXCLUDED.brand, category_id = EXCLUDED.category_id,
    release_year = EXCLUDED.release_year, rating = EXCLUDED.rating, price = EXCLUDED.price,
    emoji = EXCLUDED.emoji, detail = EXCLUDED.detail, poster_url = EXCLUDED.poster_url,
    updated_at = now();

-- ให้ลำดับของ id ถัดไปต่อจากเลขที่ใส่ไว้ (ไม่งั้น INSERT จากหลังบ้านจะชน id เดิม)
SELECT setval(pg_get_serial_sequence('equipment', 'id'), (SELECT max(id) FROM equipment));

COMMIT;

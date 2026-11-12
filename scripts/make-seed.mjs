// ใช้: node scripts/make-seed.mjs ../Computer_equipment/src/data/data.js > seed.sql
import { readFileSync, writeFileSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const input = process.argv[2];
if (!input) { console.error('ใส่ path ของ data.js ด้วย'); process.exit(1); }
// data.js เป็น ES module แต่นามสกุล .js ใน CRA ไม่มี "type":"module" จึงก็อปเป็น .mjs ชั่วคราวก่อน import
const dir = mkdtempSync(join(tmpdir(), 'seed-'));
const tmp = join(dir, 'data.mjs');
writeFileSync(tmp, readFileSync(input, 'utf8'));
const { movies } = await import(pathToFileURL(tmp).href);
rmSync(dir, { recursive: true });
const q = (s) => (s == null || s === '' ? 'NULL' : `'${String(s).replace(/'/g, "''")}'`);
const n = (x) => (x == null ? 'NULL' : String(x));
const rows = movies.map(m =>
  `    (${m.id}, ${q(m.title)}, ${q(m.titleTh)}, (SELECT id FROM categories WHERE name_th = ${q(m.genre)}), ${n(m.year)}, ${n(m.rating)}, ${n(m.price)}, ${q(m.emoji)}, ${q(m.detail)}, ${q(m.poster)})`);
const out = `-- ข้อมูลตั้งต้น ${movies.length} รายการ จาก src/data/data.js ของหน้าเว็บ (ราคาและคะแนนเป็นค่าตัวอย่าง)
-- รันหลัง schema.sql  รันซ้ำได้ (อัปเดตแถวเดิมตาม id)
-- สร้างใหม่จาก data.js ได้ด้วย: node scripts/make-seed.mjs path/to/data.js > seed.sql

BEGIN;

INSERT INTO equipment (id, title, brand, category_id, release_year, rating, price, emoji, detail, poster_url) VALUES
${rows.join(',\n')}
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title, brand = EXCLUDED.brand, category_id = EXCLUDED.category_id,
    release_year = EXCLUDED.release_year, rating = EXCLUDED.rating, price = EXCLUDED.price,
    emoji = EXCLUDED.emoji, detail = EXCLUDED.detail, poster_url = EXCLUDED.poster_url,
    updated_at = now();

-- ให้ลำดับของ id ถัดไปต่อจากเลขที่ใส่ไว้ (ไม่งั้น INSERT จากหลังบ้านจะชน id เดิม)
SELECT setval(pg_get_serial_sequence('equipment', 'id'), (SELECT max(id) FROM equipment));

COMMIT;
`;
process.stdout.write(out);

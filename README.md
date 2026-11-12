# TechReview API

Backend (Go + PostgreSQL) ของเว็บรีวิวอุปกรณ์คอมพิวเตอร์ TechReview โครงสร้างเดียวกับ MovieHub:
`main.go` ต่อสาย → `mw/` ด่านตรวจตั๋ว → `handler/` รับ-ตอบ HTTP → `store/` SQL ทั้งหมด, ตั๋วเป็น JWT (HS256, อายุ 1 ชั่วโมง), รหัสผ่านเป็น bcrypt

## เริ่มใช้งาน

```bash
# 1) สร้างตารางและข้อมูลตั้งต้น (ฐานข้อมูลเปล่า, PostgreSQL 14+)
psql "$DATABASE_URL" -f schema.sql
psql "$DATABASE_URL" -f seed.sql        # 10 รายการจาก src/data/data.js รันซ้ำได้

# 2) ตั้งค่า
cp .env.example .env                    # แล้วใส่ DATABASE_URL และ JWT_SECRET (openssl rand -base64 48)

# 3) รัน
go mod tidy                             # ครั้งแรก
go test ./...
go run .                                # http://localhost:8080
```

ต่อกับหน้าเว็บ: ตั้ง `REACT_APP_API_URL=http://localhost:8080` ใน `.env` ของ React (ตอน deploy ใส่ URL ของ Render)
และใส่ URL ของหน้าเว็บบน Vercel ใน `ALLOWED_ORIGINS` ของ backend

### รันด้วย Docker (ไม่ต้องลง Go / Postgres)

```bash
cp .env.example .env        # ตั้ง POSTGRES_PASSWORD (ตัวอักษร/ตัวเลขล้วน), JWT_SECRET, ALLOWED_ORIGINS
docker compose up --build   # API ที่ http://localhost:8080 · Postgres ที่ 127.0.0.1:5432
```

- ครั้งแรก Postgres รัน `schema.sql` + `seed.sql` ให้เองอัตโนมัติ (รันเฉพาะตอน volume ว่าง) ข้อมูลเก็บใน volume `pgdata`
- แก้ `schema.sql` แล้วอยากเริ่มใหม่: `docker compose down -v` (ลบข้อมูลทั้งหมด)
- สร้าง admin: `docker compose exec db psql -U techreview -c "UPDATE members SET role='admin' WHERE lower(email)='you@example.com';"`
- ใน container ไม่มีไฟล์ `.env` (อยู่ใน `.dockerignore`) ค่าทั้งหมดมาจาก compose · `Dockerfile` ใช้ build บน Render ได้เหมือนกัน

### สร้าง admin คนแรก

สมัครสมาชิกผ่านหน้าเว็บก่อน แล้วเลื่อนบทบาทด้วย SQL (หลังจากนั้น admin เลื่อนคนอื่นผ่านหลังบ้านได้):

```sql
UPDATE members SET role = 'admin' WHERE lower(email) = 'you@example.com';
```

ต้อง login ใหม่หลังเลื่อน เพราะบทบาทถูกฝังอยู่ในตั๋วที่ออกไปแล้ว

## Endpoints

🔒 = ต้องแนบ `Authorization: Bearer <token>` · 👑 = ต้องเป็น admin (ไม่ใช่ → 403)
ตอบ error เป็น `{"error": {"code": "...", "message": "..."}}` เสมอ

| Method | Path | หมายเหตุ |
|---|---|---|
| POST | `/api/auth/register` | `{email, password, displayName}` → 201 · 409 อีเมลซ้ำ |
| POST | `/api/auth/login` | → `{token, member}` |
| GET | `/api/me` 🔒 | |
| GET | `/api/equipment?q=&category=` | ค้นจากชื่อรุ่นหรือยี่ห้อ, `category` = เลขหมวด |
| GET | `/api/equipment/{id}` | แนบตั๋วมาด้วยจะได้ `myVote`, `inWishlist` · มี `memberRating`, `memberVoteCount` |
| GET | `/api/categories` | `[{id, nameEn, nameTh}]` ใช้ทำ dropdown |
| GET | `/api/equipment/{id}/reviews` | |
| POST | `/api/equipment/{id}/reviews` 🔒 | `{text}` 10–2000 ตัวอักษร · 409 รีวิวซ้ำ (1 คนต่อ 1 รุ่น) |
| DELETE | `/api/reviews/{reviewId}` 🔒 | ลบของตัวเอง (admin ลบของใครก็ได้) |
| PUT / DELETE | `/api/equipment/{id}/vote` 🔒 | `{score}` จำนวนเต็ม 1–10 |
| GET | `/api/me/wishlist` 🔒 | |
| PUT / DELETE | `/api/me/wishlist/{equipmentId}` 🔒 | PUT ส่ง `{note}` หรือไม่ส่ง body ก็ได้ |
| GET | `/api/admin/stats` 👑 | |
| GET | `/api/admin/members` 👑 | |
| PUT | `/api/admin/members/{id}/role` 👑 | `{role: "member" \| "admin"}` · เปลี่ยนของตัวเองไม่ได้ |
| POST | `/api/admin/equipment` 👑 | ดูช่องด้านล่าง |
| GET / PUT / DELETE | `/api/admin/equipment/{id}` 👑 | GET ตอบ `impact` = จำนวนรีวิว/โหวต/wishlist ที่จะหายถ้าลบ |
| GET | `/healthz` | สำหรับ Render |

ช่องของอุปกรณ์ (ตรงกับ `src/data/data.js`): `title` ชื่อรุ่น · `titleTh` ยี่ห้อ · `genre` ชื่อหมวด (ตอนอ่าน) / `categoryId` (ตอนเขียน) ·
`year` · `rating` คะแนนทีมงาน 0–10 · `price` บาท · `emoji` · `detail` · `poster` URL เต็ม (`http(s)://`)

**ชื่อเดิมของ MovieHub:** หน้าเว็บยังเรียก `/api/movies`, `/api/genres`, `/api/admin/movies` จึงเปิดไว้เป็นชื่อเล่น (alias) ชี้ไปที่ handler เดียวกัน
ใช้ได้ทันทีโดยไม่ต้องแก้ `backend.js` ลบได้เมื่อหน้าเว็บเปลี่ยนเป็น `/api/equipment` แล้ว (ลูปใน `routes()` ของ `main.go`)

## ต่างจาก MovieHub อย่างไร

- ตัด TMDB ออก (ไม่มี `movie_lists`, poster path, sync) · `movies` → `equipment`, `genres` → `categories`, เพิ่ม `price`, `emoji`
- เพิ่ม endpoint ที่หน้าเว็บเรียกแต่ MovieHub ยังไม่มี: เขียน/ลบรีวิว, โหวต, wishlist (`handler/reviews.go` ของ MovieHub ว่างเปล่า)
- `GET /api/equipment/{id}` ใช้ `mw.OptionalLogin` — ไม่มีตั๋วก็เปิดได้ มีตั๋วก็เห็น `myVote`/`inWishlist`
- จำกัดขนาด body 1 MB, ตั้ง timeout ของ HTTP server, ค้นหาแบบ escape `%` `_` ที่ผู้ใช้พิมพ์
- Login ใช้เวลาใกล้เคียงกันไม่ว่าจะมีอีเมลนั้นหรือไม่ (กันเดาอีเมลจากเวลาตอบ)
- ตรวจรูปแบบอีเมลด้วย `net/mail` แทนการเช็กแค่ `@`
- ไม่ใช้ `godotenv` (ใช้ `loadEnv` ของตัวเองอย่างเดียว) · ไม่มีไฟล์ `.env` จริงมาด้วย

## ข้อควรรู้

- บทบาทถูกฝังในตั๋ว (1 ชั่วโมง) — ลดสิทธิ์ admin แล้วจะมีผลเมื่อตั๋วเดิมหมดอายุ
- ลบอุปกรณ์ = รีวิว/โหวต/wishlist ของรุ่นนั้นหายตามด้วย (`ON DELETE CASCADE`) หน้าแอดมินควรโชว์ `impact` ให้ยืนยันก่อนลบ
- `go.sum` ก็อปมาจาก MovieHub (เวอร์ชันไลบรารีชุดเดียวกัน) · ถ้า `go build` ฟ้องว่าขาดอะไร รัน `go mod tidy`
- ถ้าใช้ Supabase pooler แบบ transaction (พอร์ต 6543) pgx อาจมีปัญหา prepared statement ให้ใช้พอร์ต 5432 เหมือน MovieHub

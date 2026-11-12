-- TechReview database schema (PostgreSQL 14+)
-- โครงเดียวกับ MovieHub: ข้อมูลหลัก (equipment) + ข้อมูลของเว็บเรา (สมาชิก รีวิว คะแนน wishlist)
-- รันครั้งเดียวกับฐานข้อมูลเปล่า:  psql "$DATABASE_URL" -f schema.sql

BEGIN;

-- ---------------------------------------------------------------------------
-- categories: หมวดอุปกรณ์ (การ์ดจอ ซีพียู ...) เก็บชื่อ 2 ภาษา ใช้ทำ dropdown ในฟอร์มหลังบ้าน
-- ---------------------------------------------------------------------------
CREATE TABLE categories (
    id       SERIAL       PRIMARY KEY,
    name_en  VARCHAR(50)  NOT NULL UNIQUE,
    name_th  VARCHAR(50)  NOT NULL UNIQUE
);

-- ---------------------------------------------------------------------------
-- equipment: อุปกรณ์ 1 แถวต่อ 1 รุ่น
-- ชื่อ field ฝั่ง JSON ตาม src/data/data.js ของหน้าเว็บ: title = ชื่อรุ่น, titleTh = ยี่ห้อ (brand),
-- genre = ชื่อหมวด, year = ปีเปิดตัว
-- ---------------------------------------------------------------------------
CREATE TABLE equipment (
    id            BIGSERIAL     PRIMARY KEY,
    title         VARCHAR(255)  NOT NULL CHECK (char_length(btrim(title)) >= 1),   -- ชื่อรุ่น
    brand         VARCHAR(100),                                                    -- ยี่ห้อ
    category_id   INTEGER       REFERENCES categories (id),                        -- หมวด (ลบหมวดที่ยังมีอุปกรณ์อยู่ไม่ได้)
    release_year  SMALLINT      CHECK (release_year BETWEEN 1990 AND 2100),
    rating        NUMERIC(3,1)  CHECK (rating BETWEEN 0 AND 10),                   -- คะแนนรีวิวของทีมงาน (NULL = ยังไม่ให้)
    price         INTEGER       CHECK (price >= 0),                                -- ราคาโดยประมาณ (บาท)
    emoji         VARCHAR(16),                                                     -- ไอคอนที่แสดงเมื่อไม่มีรูป
    detail        TEXT,
    poster_url    VARCHAR(500),                                                    -- URL เต็มของรูป (http/https)
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_equipment_category   ON equipment (category_id);
CREATE INDEX idx_equipment_title_lower ON equipment (lower(title));

-- ===========================================================================
-- ข้อมูลของเว็บเราเอง
-- ===========================================================================

-- ---------------------------------------------------------------------------
-- members: สมาชิก  อีเมลห้ามซ้ำโดยไม่สนตัวพิมพ์เล็กใหญ่ (Go lower() ก่อนบันทึกและก่อนค้นเสมอ)
-- ---------------------------------------------------------------------------
CREATE TABLE members (
    id             BIGSERIAL     PRIMARY KEY,
    email          VARCHAR(255)  NOT NULL,
    password_hash  VARCHAR(255)  NOT NULL,          -- bcrypt ห้ามเก็บรหัสผ่านตรง ๆ
    display_name   VARCHAR(50)   NOT NULL CHECK (char_length(btrim(display_name)) >= 1),
    role           VARCHAR(10)   NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'admin')),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_members_email ON members (lower(email));

-- ---------------------------------------------------------------------------
-- reviews: สมาชิก 1 คนรีวิวอุปกรณ์ 1 รุ่นได้ 1 ครั้ง
-- ชื่อ constraint ตั้งเอง เพราะ Go ใช้ชื่อแยกว่า FK ที่พังคือสมาชิกหรืออุปกรณ์
-- ---------------------------------------------------------------------------
CREATE TABLE reviews (
    id            BIGSERIAL    PRIMARY KEY,
    member_id     BIGINT       NOT NULL,
    equipment_id  BIGINT       NOT NULL,
    text          TEXT         NOT NULL CHECK (char_length(btrim(text)) BETWEEN 10 AND 2000),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT reviews_member_fk    FOREIGN KEY (member_id)    REFERENCES members (id)   ON DELETE CASCADE,
    CONSTRAINT reviews_equipment_fk FOREIGN KEY (equipment_id) REFERENCES equipment (id) ON DELETE CASCADE,
    CONSTRAINT reviews_one_per_member UNIQUE (member_id, equipment_id)
);

CREATE INDEX idx_reviews_equipment_created ON reviews (equipment_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- votes: คะแนน 1–10 จากสมาชิก 1 คน 1 รุ่น 1 คะแนน (โหวตใหม่ = แก้คะแนนเดิม)
-- ---------------------------------------------------------------------------
CREATE TABLE votes (
    member_id     BIGINT       NOT NULL,
    equipment_id  BIGINT       NOT NULL,
    score         SMALLINT     NOT NULL CHECK (score BETWEEN 1 AND 10),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, equipment_id),
    CONSTRAINT votes_member_fk    FOREIGN KEY (member_id)    REFERENCES members (id)   ON DELETE CASCADE,
    CONSTRAINT votes_equipment_fk FOREIGN KEY (equipment_id) REFERENCES equipment (id) ON DELETE CASCADE
);

CREATE INDEX idx_votes_equipment ON votes (equipment_id);

-- คะแนนเฉลี่ยจากสมาชิกเว็บเรา (แยกจาก equipment.rating ที่เป็นคะแนนของทีมงาน)
CREATE VIEW equipment_vote_stats AS
SELECT equipment_id,
       count(*)              AS vote_count,
       round(avg(score), 1)  AS vote_average
FROM votes
GROUP BY equipment_id;

-- ---------------------------------------------------------------------------
-- wishlist_items: อุปกรณ์ที่สมาชิกอยากได้ รุ่นเดิมใส่ซ้ำไม่ได้
-- ---------------------------------------------------------------------------
CREATE TABLE wishlist_items (
    member_id     BIGINT        NOT NULL,
    equipment_id  BIGINT        NOT NULL,
    note          VARCHAR(200),
    added_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, equipment_id),
    CONSTRAINT wishlist_member_fk    FOREIGN KEY (member_id)    REFERENCES members (id)   ON DELETE CASCADE,
    CONSTRAINT wishlist_equipment_fk FOREIGN KEY (equipment_id) REFERENCES equipment (id) ON DELETE CASCADE
);

CREATE INDEX idx_wishlist_member_added ON wishlist_items (member_id, added_at DESC);

-- ---------------------------------------------------------------------------
-- seed: หมวดอุปกรณ์ (ตรงกับหมวดใน src/data/data.js)
-- ---------------------------------------------------------------------------
INSERT INTO categories (name_en, name_th) VALUES
    ('Graphics Card', 'การ์ดจอ'),
    ('CPU',           'ซีพียู'),
    ('Mouse',         'เมาส์'),
    ('Keyboard',      'คีย์บอร์ด'),
    ('Monitor',       'จอมอนิเตอร์'),
    ('Headset',       'หูฟัง'),
    ('SSD',           'SSD'),
    ('RAM',           'แรม');

COMMIT;

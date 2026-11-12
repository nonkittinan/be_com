package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------- Dashboard ----------

type TopEquipment struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	TitleTh     *string `json:"titleTh"`
	Poster      *string `json:"poster"`
	VoteCount   int64   `json:"voteCount"`
	VoteAverage float64 `json:"voteAverage"`
}

type Stats struct {
	Equipment     int64          `json:"equipment"`
	Members       int64          `json:"members"`
	Admins        int64          `json:"admins"`
	Reviews       int64          `json:"reviews"`
	Votes         int64          `json:"votes"`
	WishlistItems int64          `json:"wishlistItems"`
	NewestMembers []Member       `json:"newestMembers"`
	TopVoted      []TopEquipment `json:"topVoted"`
}

// Stats รวมตัวเลขของทั้งเว็บในคำขอเดียว ให้หน้า Dashboard ยิงครั้งเดียวจบ
func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.Pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM equipment),
                (SELECT count(*) FROM members),
                (SELECT count(*) FROM members WHERE role = 'admin'),
                (SELECT count(*) FROM reviews),
                (SELECT count(*) FROM votes),
                (SELECT count(*) FROM wishlist_items)`).
		Scan(&st.Equipment, &st.Members, &st.Admins, &st.Reviews, &st.Votes, &st.WishlistItems)
	if err != nil {
		return st, err
	}

	st.NewestMembers = []Member{}
	rows, err := s.Pool.Query(ctx,
		`SELECT id, email, display_name, role, created_at
         FROM members ORDER BY created_at DESC, id DESC LIMIT 5`)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.DisplayName, &m.Role, &m.CreatedAt); err != nil {
			rows.Close()
			return st, err
		}
		st.NewestMembers = append(st.NewestMembers, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return st, err
	}

	// ใช้ view equipment_vote_stats: คะแนนเฉลี่ยจากสมาชิกเว็บเรา
	st.TopVoted = []TopEquipment{}
	rows, err = s.Pool.Query(ctx,
		`SELECT e.id, e.title, e.brand, e.poster_url, vs.vote_count, vs.vote_average::float8
         FROM equipment_vote_stats vs
         JOIN equipment e ON e.id = vs.equipment_id
         ORDER BY vs.vote_average DESC, vs.vote_count DESC, e.title
         LIMIT 5`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var t TopEquipment
		if err := rows.Scan(&t.ID, &t.Title, &t.TitleTh, &t.Poster, &t.VoteCount, &t.VoteAverage); err != nil {
			return st, err
		}
		st.TopVoted = append(st.TopVoted, t)
	}
	return st, rows.Err()
}

// ---------- หมวดอุปกรณ์ (สาธารณะ ใช้ทำ dropdown และตัวกรอง) ----------

type Category struct {
	ID     int    `json:"id"`
	NameEn string `json:"nameEn"`
	NameTh string `json:"nameTh"`
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name_en, name_th FROM categories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.NameEn, &c.NameTh); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ---------- อุปกรณ์: มุมมองของ admin ----------

// EquipmentInput คือช่องที่ admin แก้ได้ (JSON ชื่อเดียวกับที่ GET ตอบ ยกเว้น categoryId แทน genre)
type EquipmentInput struct {
	Title      string   `json:"title"`
	TitleTh    *string  `json:"titleTh"`    // ยี่ห้อ
	CategoryID *int     `json:"categoryId"` // หมวด หรือ null
	Year       *int     `json:"year"`
	Rating     *float64 `json:"rating"` // คะแนนทีมงาน 0–10
	Price      *int     `json:"price"`  // บาท
	Emoji      *string  `json:"emoji"`
	Detail     *string  `json:"detail"`
	Poster     *string  `json:"poster"` // URL เต็ม
}

// Impact บอกว่าถ้าลบอุปกรณ์รุ่นนี้ ข้อมูลของสมาชิกจะหายตามไปเท่าไร (ON DELETE CASCADE)
type Impact struct {
	Reviews  int64 `json:"reviews"`
	Votes    int64 `json:"votes"`
	Wishlist int64 `json:"wishlist"`
}

type AdminEquipment struct {
	ID int64 `json:"id"`
	EquipmentInput
	UpdatedAt time.Time `json:"updatedAt"`
	Impact    Impact    `json:"impact"`
}

// AdminGetEquipment อ่านทุกช่องที่ฟอร์มต้องใช้ พร้อมจำนวนข้อมูลที่ผูกกับรุ่นนี้
func (s *Store) AdminGetEquipment(ctx context.Context, id int64) (AdminEquipment, error) {
	var a AdminEquipment
	err := s.Pool.QueryRow(ctx,
		`SELECT e.id, e.title, e.brand, e.category_id, e.release_year, e.rating::float8,
                e.price, e.emoji, e.detail, e.poster_url, e.updated_at,
                (SELECT count(*) FROM reviews        WHERE equipment_id = e.id),
                (SELECT count(*) FROM votes          WHERE equipment_id = e.id),
                (SELECT count(*) FROM wishlist_items WHERE equipment_id = e.id)
         FROM equipment e WHERE e.id = $1`, id).
		Scan(&a.ID, &a.Title, &a.TitleTh, &a.CategoryID, &a.Year, &a.Rating,
			&a.Price, &a.Emoji, &a.Detail, &a.Poster, &a.UpdatedAt,
			&a.Impact.Reviews, &a.Impact.Votes, &a.Impact.Wishlist)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

func badCategory(err error) error {
	if pgErr := pgError(err); pgErr != nil && pgErr.Code == "23503" {
		return ErrBadCategory
	}
	return err
}

// CreateEquipment เพิ่มอุปกรณ์ใหม่ id ออกให้โดยฐานข้อมูล (BIGSERIAL)
func (s *Store) CreateEquipment(ctx context.Context, in EquipmentInput) (AdminEquipment, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO equipment (title, brand, category_id, release_year, rating, price, emoji, detail, poster_url)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
         RETURNING id`,
		in.Title, in.TitleTh, in.CategoryID, in.Year, in.Rating, in.Price, in.Emoji, in.Detail, in.Poster).
		Scan(&id)
	if err != nil {
		return AdminEquipment{}, badCategory(err)
	}
	return s.AdminGetEquipment(ctx, id)
}

// UpdateEquipment แก้ทุกช่องที่ admin ดูแล (ส่งมาครบทุกช่อง ช่องที่เป็น null จะถูกล้าง)
func (s *Store) UpdateEquipment(ctx context.Context, id int64, in EquipmentInput) (AdminEquipment, error) {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE equipment
         SET title = $2, brand = $3, category_id = $4, release_year = $5, rating = $6,
             price = $7, emoji = $8, detail = $9, poster_url = $10, updated_at = now()
         WHERE id = $1`,
		id, in.Title, in.TitleTh, in.CategoryID, in.Year, in.Rating, in.Price, in.Emoji, in.Detail, in.Poster)
	if err != nil {
		return AdminEquipment{}, badCategory(err)
	}
	if tag.RowsAffected() == 0 {
		return AdminEquipment{}, ErrNotFound
	}
	return s.AdminGetEquipment(ctx, id)
}

// DeleteEquipment ลบอุปกรณ์ รีวิว โหวต และ wishlist ของรุ่นนั้นหายตามด้วย ON DELETE CASCADE ใน schema
func (s *Store) DeleteEquipment(ctx context.Context, id int64) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM equipment WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

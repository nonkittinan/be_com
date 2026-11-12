package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type ReviewAuthor struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
}

type Review struct {
	ID          int64        `json:"id"`
	MemberID    int64        `json:"memberId"`
	EquipmentID int64        `json:"equipmentId"`
	Text        string       `json:"text"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Member      ReviewAuthor `json:"member"`
	Score       *int16       `json:"score"` // คะแนนที่คนเขียนรีวิวให้ไว้ (ถ้ามี)
}

const reviewCols = `r.id, r.member_id, r.equipment_id, r.text, r.created_at, r.updated_at, m.display_name, v.score`

const reviewFrom = `FROM reviews r
         JOIN members m ON m.id = r.member_id
         LEFT JOIN votes v ON v.member_id = r.member_id AND v.equipment_id = r.equipment_id`

func scanReview(row pgx.Row, r *Review) error {
	if err := row.Scan(&r.ID, &r.MemberID, &r.EquipmentID, &r.Text,
		&r.CreatedAt, &r.UpdatedAt, &r.Member.DisplayName, &r.Score); err != nil {
		return err
	}
	r.Member.ID = r.MemberID
	return nil
}

func (s *Store) ListReviews(ctx context.Context, equipmentID int64) ([]Review, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+reviewCols+` `+reviewFrom+`
         WHERE r.equipment_id = $1
         ORDER BY r.created_at DESC, r.id DESC`, equipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Review{}
	for rows.Next() {
		var r Review
		if err := scanReview(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateReview เพิ่มรีวิว สมาชิก 1 คนต่ออุปกรณ์ 1 รุ่นได้ 1 รีวิว (ซ้ำ = ErrAlreadyReviewed)
// ไม่มีอุปกรณ์รุ่นนั้น = ErrNotFound  memberID ต้องมาจากตั๋วเท่านั้น ห้ามรับจาก body
func (s *Store) CreateReview(ctx context.Context, memberID, equipmentID int64, text string) (Review, error) {
	var r Review
	err := scanReview(s.Pool.QueryRow(ctx,
		`WITH ins AS (
             INSERT INTO reviews (member_id, equipment_id, text)
             VALUES ($1, $2, $3)
             RETURNING id, member_id, equipment_id, text, created_at, updated_at
         )
         SELECT r.id, r.member_id, r.equipment_id, r.text, r.created_at, r.updated_at, m.display_name, v.score
         FROM ins r
         JOIN members m ON m.id = r.member_id
         LEFT JOIN votes v ON v.member_id = r.member_id AND v.equipment_id = r.equipment_id`,
		memberID, equipmentID, text), &r)
	if pgErr := pgError(err); pgErr != nil && pgErr.Code == "23505" {
		return Review{}, ErrAlreadyReviewed
	}
	if err != nil {
		return Review{}, mapFK(err)
	}
	return r, nil
}

// DeleteReview ลบรีวิวของตัวเอง (admin ลบของใครก็ได้) ไม่พบหรือไม่ใช่ของตัวเอง ตอบ ErrNotFound เหมือนกัน
// เพื่อไม่บอกคนอื่นว่ารีวิวเลขนั้นมีอยู่จริง
func (s *Store) DeleteReview(ctx context.Context, reviewID, memberID int64, isAdmin bool) error {
	tag, err := s.Pool.Exec(ctx,
		`DELETE FROM reviews WHERE id = $1 AND (member_id = $2 OR $3)`, reviewID, memberID, isAdmin)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

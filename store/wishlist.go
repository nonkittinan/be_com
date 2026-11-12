package store

import (
	"context"
	"time"
)

// WishlistItem คืออุปกรณ์รูปร่างเดียวกับ GET /api/equipment บวกโน้ตและเวลาที่เพิ่ม
// หน้าเว็บจึงใช้ MovieCard / MovieGrid เดิมได้เลย
type WishlistItem struct {
	Equipment
	Note    *string   `json:"note"`
	AddedAt time.Time `json:"addedAt"`
}

func (s *Store) ListWishlist(ctx context.Context, memberID int64) ([]WishlistItem, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+equipmentCols+`, w.note, w.added_at
         FROM wishlist_items w
         JOIN equipment e ON e.id = w.equipment_id
         LEFT JOIN categories c ON c.id = e.category_id
         WHERE w.member_id = $1
         ORDER BY w.added_at DESC, e.id DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WishlistItem{}
	for rows.Next() {
		var w WishlistItem
		if err := rows.Scan(&w.ID, &w.Title, &w.TitleTh, &w.Genre, &w.Year, &w.Rating,
			&w.Price, &w.Emoji, &w.Poster, &w.Detail, &w.Note, &w.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// PutWishlist เพิ่มเข้า wishlist (หรือแก้โน้ตถ้ามีอยู่แล้ว) ใช้ PUT เพราะกดซ้ำกี่ครั้งผลก็เหมือนเดิม
func (s *Store) PutWishlist(ctx context.Context, memberID, equipmentID int64, note *string) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO wishlist_items (member_id, equipment_id, note) VALUES ($1, $2, $3)
         ON CONFLICT (member_id, equipment_id) DO UPDATE SET note = EXCLUDED.note`,
		memberID, equipmentID, note)
	return mapFK(err)
}

func (s *Store) DeleteWishlist(ctx context.Context, memberID, equipmentID int64) error {
	_, err := s.Pool.Exec(ctx,
		`DELETE FROM wishlist_items WHERE member_id = $1 AND equipment_id = $2`, memberID, equipmentID)
	return err
}

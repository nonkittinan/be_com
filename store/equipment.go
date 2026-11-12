package store

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Equipment คืออุปกรณ์ 1 รุ่น รูปร่างตาม src/data/data.js ของหน้าเว็บ
// title = ชื่อรุ่น, titleTh = ยี่ห้อ, genre = ชื่อหมวด (ไทย), year = ปีเปิดตัว, rating = คะแนนทีมงาน
type Equipment struct {
	ID      int64    `json:"id"`
	Title   string   `json:"title"`
	TitleTh *string  `json:"titleTh"`
	Genre   *string  `json:"genre"`
	Year    *int     `json:"year"`
	Rating  *float64 `json:"rating"`
	Price   *int     `json:"price"`
	Emoji   *string  `json:"emoji"`
	Poster  *string  `json:"poster"`
	Detail  *string  `json:"detail"`
}

// EquipmentDetail คืออุปกรณ์พร้อมคะแนนจากสมาชิก และสถานะของคนที่ login อยู่ (ไม่ login = null / false)
type EquipmentDetail struct {
	Equipment
	MemberRating    *float64 `json:"memberRating"`
	MemberVoteCount int64    `json:"memberVoteCount"`
	MyVote          *int16   `json:"myVote"`
	InWishlist      bool     `json:"inWishlist"`
}

const equipmentCols = `e.id, e.title, e.brand, c.name_th, e.release_year, e.rating::float8,
                e.price, e.emoji, e.poster_url, e.detail`

const equipmentFrom = `FROM equipment e LEFT JOIN categories c ON c.id = e.category_id`

func scanEquipment(row pgx.Row, e *Equipment) error {
	return row.Scan(&e.ID, &e.Title, &e.TitleTh, &e.Genre, &e.Year, &e.Rating,
		&e.Price, &e.Emoji, &e.Poster, &e.Detail)
}

// likeEscaper ทำให้ % _ \ ที่ผู้ใช้พิมพ์มาเป็นตัวอักษรธรรมดา ไม่ใช่ wildcard ของ LIKE
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string { return likeEscaper.Replace(s) }

// maxList กันตอบรายการยาวเกินไป (ตอนนี้แคตตาล็อกเล็ก ถ้าโตค่อยทำ paging)
const maxList = 500

// SearchEquipment ค้นจากชื่อรุ่นหรือยี่ห้อ (q ว่าง = ทั้งหมด) และกรองตามหมวดได้ (categoryID 0 = ทุกหมวด)
// ค่าจากผู้ใช้เดินทางผ่าน $1 $2 แยกจากคำสั่งเสมอ
func (s *Store) SearchEquipment(ctx context.Context, q string, categoryID int) ([]Equipment, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+equipmentCols+` `+equipmentFrom+`
         WHERE ($1 = '' OR e.title ILIKE '%' || $1 || '%' ESCAPE '\'
                        OR e.brand ILIKE '%' || $1 || '%' ESCAPE '\')
           AND ($2::int = 0 OR e.category_id = $2::int)
         ORDER BY e.id
         LIMIT `+strconv.Itoa(maxList), escapeLike(strings.TrimSpace(q)), categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Equipment{}
	for rows.Next() {
		var e Equipment
		if err := scanEquipment(rows, &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetEquipment หาอุปกรณ์จากเลข memberID = 0 แปลว่าผู้เยี่ยมชมที่ยังไม่ login ไม่พบตอบ ErrNotFound
func (s *Store) GetEquipment(ctx context.Context, id, memberID int64) (EquipmentDetail, error) {
	var d EquipmentDetail
	err := s.Pool.QueryRow(ctx,
		`SELECT `+equipmentCols+`,
                vs.vote_average::float8, coalesce(vs.vote_count, 0),
                (SELECT score FROM votes WHERE member_id = $2 AND equipment_id = e.id),
                EXISTS (SELECT 1 FROM wishlist_items WHERE member_id = $2 AND equipment_id = e.id)
         `+equipmentFrom+`
         LEFT JOIN equipment_vote_stats vs ON vs.equipment_id = e.id
         WHERE e.id = $1`, id, memberID).
		Scan(&d.ID, &d.Title, &d.TitleTh, &d.Genre, &d.Year, &d.Rating,
			&d.Price, &d.Emoji, &d.Poster, &d.Detail,
			&d.MemberRating, &d.MemberVoteCount, &d.MyVote, &d.InWishlist)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}

package store

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrEmailTaken      = errors.New("email taken")
	ErrBadCategory     = errors.New("bad category")
	ErrAlreadyReviewed = errors.New("already reviewed")
	ErrMemberGone      = errors.New("member gone") // ตั๋วยังไม่หมดอายุ แต่บัญชีถูกลบไปแล้ว
)

type Store struct {
	Pool *pgxpool.Pool
}

// pgError คืน error ของ PostgreSQL ถ้าใช่ (ดูรหัสที่ .Code) ของ PostgreSQL (เช่น 23505 = unique ชน, 23503 = foreign key ไม่มีต้นทาง)
func pgError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}
	return nil
}

// mapFK แปลง foreign key พังเป็น error ของเรา: ชื่อ constraint ที่ลงท้าย member_fk = บัญชีหาย ที่เหลือ = อุปกรณ์ไม่มี
// (ชื่อ constraint ตั้งเองไว้ใน schema.sql)
func mapFK(err error) error {
	if pgErr := pgError(err); pgErr != nil && pgErr.Code == "23503" {
		if strings.HasSuffix(pgErr.ConstraintName, "member_fk") {
			return ErrMemberGone
		}
		return ErrNotFound
	}
	return err
}

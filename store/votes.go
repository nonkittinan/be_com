package store

import "context"

// PutVote ให้คะแนน 1–10 (โหวตใหม่ = แก้คะแนนเดิม) กดซ้ำกี่ครั้งผลเหมือนเดิม
// score ตรวจช่วงใน handler แล้ว และฐานข้อมูลมี CHECK ซ้ำอีกชั้น
func (s *Store) PutVote(ctx context.Context, memberID, equipmentID int64, score int) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO votes (member_id, equipment_id, score) VALUES ($1, $2, $3)
         ON CONFLICT (member_id, equipment_id)
         DO UPDATE SET score = EXCLUDED.score, updated_at = now()`,
		memberID, equipmentID, score)
	return mapFK(err)
}

// DeleteVote ถอนคะแนนของตัวเอง ไม่เคยโหวตก็ไม่ error (ผลลัพธ์สุดท้ายเหมือนกัน)
func (s *Store) DeleteVote(ctx context.Context, memberID, equipmentID int64) error {
	_, err := s.Pool.Exec(ctx,
		`DELETE FROM votes WHERE member_id = $1 AND equipment_id = $2`, memberID, equipmentID)
	return err
}

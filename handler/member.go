package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"techreview/mw"
	"techreview/store"
)

// memberAndEquipment อ่านเลขอุปกรณ์จาก URL (ชื่อ path value = name) และเลขสมาชิกจากตั๋ว
// ถ้าอ่านไม่ได้ จะตอบ error ให้เองแล้วคืน ok = false
func memberAndEquipment(w http.ResponseWriter, r *http.Request, name string) (memberID, equipmentID int64, ok bool) {
	equipmentID, ok = pathID(r, name)
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return 0, 0, false
	}
	memberID, ok = mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return 0, 0, false
	}
	return memberID, equipmentID, true
}

// storeFailed ตอบ error ที่พบบ่อยของ store (อุปกรณ์ไม่มี / บัญชีหาย / อื่น ๆ) คืน true ถ้ามี error
func storeFailed(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์รุ่นนี้")
	case errors.Is(err, store.ErrMemberGone):
		writeError(w, http.StatusUnauthorized, "MEMBER_GONE", "ไม่พบบัญชีนี้แล้ว กรุณาเข้าสู่ระบบใหม่")
	default:
		internal(w, err)
	}
	return true
}

// ---------- คะแนน ----------

type voteReq struct {
	Score *int `json:"score"`
}

// validScore: จำนวนเต็ม 1–10 เท่านั้น (ไม่ส่งมา = ไม่ผ่าน)
func validScore(s *int) bool { return s != nil && *s >= 1 && *s <= 10 }

// PutVote PUT /api/equipment/{id}/vote 🔒 body {"score": 1..10}
func (h *Handler) PutVote(w http.ResponseWriter, r *http.Request) {
	memberID, equipmentID, ok := memberAndEquipment(w, r, "id")
	if !ok {
		return
	}
	var req voteReq
	if err := decodeJSON(w, r, &req); err != nil || !validScore(req.Score) {
		writeError(w, http.StatusBadRequest, "BAD_SCORE", "score ต้องเป็นจำนวนเต็ม 1 ถึง 10")
		return
	}
	if storeFailed(w, h.Store.PutVote(r.Context(), memberID, equipmentID, *req.Score)) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"equipmentId": equipmentID, "score": *req.Score})
}

// DeleteVote DELETE /api/equipment/{id}/vote 🔒
func (h *Handler) DeleteVote(w http.ResponseWriter, r *http.Request) {
	memberID, equipmentID, ok := memberAndEquipment(w, r, "id")
	if !ok {
		return
	}
	if storeFailed(w, h.Store.DeleteVote(r.Context(), memberID, equipmentID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Wishlist ----------

type wishReq struct {
	Note *string `json:"note"`
}

// cleanNote ตัดช่องว่าง ว่างเปล่า = ไม่มีโน้ต (NULL) ยาวเกิน 200 ตัวอักษรไม่ผ่าน
func cleanNote(p *string) (note *string, ok bool) {
	if p == nil {
		return nil, true
	}
	s := strings.TrimSpace(*p)
	if utf8.RuneCountInString(s) > 200 {
		return nil, false
	}
	if s == "" {
		return nil, true
	}
	return &s, true
}

// GetWishlist GET /api/me/wishlist 🔒
func (h *Handler) GetWishlist(w http.ResponseWriter, r *http.Request) {
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	items, err := h.Store.ListWishlist(r.Context(), memberID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// PutWishlist PUT /api/me/wishlist/{equipmentId} 🔒 body {"note": "..."} จะไม่ส่ง body เลยก็ได้
func (h *Handler) PutWishlist(w http.ResponseWriter, r *http.Request) {
	memberID, equipmentID, ok := memberAndEquipment(w, r, "equipmentId")
	if !ok {
		return
	}
	var req wishReq
	if err := decodeJSON(w, r, &req); err != nil && !errors.Is(err, io.EOF) { // EOF = ไม่มี body
		badBody(w, err)
		return
	}
	note, ok := cleanNote(req.Note)
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_NOTE", "โน้ตยาวไม่เกิน 200 ตัวอักษร")
		return
	}
	if storeFailed(w, h.Store.PutWishlist(r.Context(), memberID, equipmentID, note)) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"equipmentId": equipmentID, "note": note})
}

// DeleteWishlist DELETE /api/me/wishlist/{equipmentId} 🔒
func (h *Handler) DeleteWishlist(w http.ResponseWriter, r *http.Request) {
	memberID, equipmentID, ok := memberAndEquipment(w, r, "equipmentId")
	if !ok {
		return
	}
	if storeFailed(w, h.Store.DeleteWishlist(r.Context(), memberID, equipmentID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

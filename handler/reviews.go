package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"techreview/mw"
	"techreview/store"
)

type reviewReq struct {
	Text string `json:"text"`
}

// validateReviewText ตัดช่องว่างหัวท้าย แล้วต้องยาว 10–2000 ตัวอักษร (กติกาเดียวกับ CHECK ใน schema)
func validateReviewText(raw string) (text, code, message string) {
	text = strings.TrimSpace(raw)
	n := utf8.RuneCountInString(text)
	if n < 10 || n > 2000 {
		return text, "BAD_TEXT", "รีวิวต้องยาว 10–2000 ตัวอักษร"
	}
	return text, "", ""
}

// ListReviews GET /api/equipment/{id}/reviews (สาธารณะ)
func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	equipmentID, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	items, err := h.Store.ListReviews(r.Context(), equipmentID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// CreateReview POST /api/equipment/{id}/reviews 🔒  ใครเป็นคนเขียนอ่านจากตั๋ว ไม่รับจาก body
func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	equipmentID, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	var req reviewReq
	if err := decodeJSON(w, r, &req); err != nil {
		badBody(w, err)
		return
	}
	text, code, msg := validateReviewText(req.Text)
	if code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}
	rev, err := h.Store.CreateReview(r.Context(), memberID, equipmentID, text)
	switch {
	case errors.Is(err, store.ErrAlreadyReviewed):
		writeError(w, http.StatusConflict, "ALREADY_REVIEWED", "คุณรีวิวอุปกรณ์รุ่นนี้ไปแล้ว")
		return
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์รุ่นนี้")
		return
	case errors.Is(err, store.ErrMemberGone):
		writeError(w, http.StatusUnauthorized, "MEMBER_GONE", "ไม่พบบัญชีนี้แล้ว กรุณาเข้าสู่ระบบใหม่")
		return
	case err != nil:
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rev)
}

// DeleteReview DELETE /api/reviews/{reviewId} 🔒  ลบได้เฉพาะรีวิวของตัวเอง (admin ลบของใครก็ได้)
func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewID, ok := pathID(r, "reviewId")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขรีวิวต้องเป็นตัวเลข")
		return
	}
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	err := h.Store.DeleteReview(r.Context(), reviewID, memberID, mw.Role(r.Context()) == "admin")
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "REVIEW_NOT_FOUND", "ไม่พบรีวิวนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

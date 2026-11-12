package handler

import (
	"errors"
	"net/http"
	"strconv"

	"techreview/mw"
	"techreview/store"
)

// SearchEquipment GET /api/equipment?q=คำค้น&category=หมวด(เลข)
func (h *Handler) SearchEquipment(w http.ResponseWriter, r *http.Request) {
	categoryID := 0
	if raw := r.URL.Query().Get("category"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "BAD_CATEGORY", "category ต้องเป็นเลขหมวด")
			return
		}
		categoryID = n
	}
	items, err := h.Store.SearchEquipment(r.Context(), r.URL.Query().Get("q"), categoryID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// GetEquipment GET /api/equipment/{id} ใครก็เปิดได้ ถ้าแนบตั๋วมาด้วยจะได้ myVote / inWishlist ของตัวเอง
func (h *Handler) GetEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	memberID, _ := mw.MemberID(r.Context()) // ไม่ login = 0
	e, err := h.Store.GetEquipment(r.Context(), id, memberID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์รุ่นนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// Categories GET /api/categories (สาธารณะ ใช้ทำ dropdown / ตัวกรอง)
func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.ListCategories(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

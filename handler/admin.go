package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"techreview/mw"
	"techreview/store"
)

// ทุก handler ในไฟล์นี้ถูกครอบด้วย RequireLogin + RequireRole("admin") ใน main.go แล้ว
// จึงไม่ต้องเช็กบทบาทซ้ำในนี้ แต่ "ใครเป็นคนกด" ยังอ่านจากตั๋วผ่าน mw.MemberID เสมอ

// ---------- Dashboard ----------

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	st, err := h.Store.Stats(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// ---------- สมาชิก ----------

func (h *Handler) AdminMembers(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.ListMembers(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type roleReq struct {
	Role string `json:"role"`
}

// AdminSetRole เลื่อนหรือลดบทบาท ห้ามแก้ของตัวเอง (กัน admin คนสุดท้ายลดตัวเองจนไม่มีใครเข้าหลังบ้านได้)
func (h *Handler) AdminSetRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขสมาชิกต้องเป็นตัวเลข")
		return
	}
	var req roleReq
	if err := decodeJSON(w, r, &req); err != nil {
		badBody(w, err)
		return
	}
	if req.Role != "member" && req.Role != "admin" {
		writeError(w, http.StatusBadRequest, "BAD_ROLE", "role ต้องเป็น member หรือ admin")
		return
	}
	if me, _ := mw.MemberID(r.Context()); me == id {
		writeError(w, http.StatusBadRequest, "SELF_ROLE", "เปลี่ยนบทบาทของตัวเองไม่ได้ ให้ admin คนอื่นทำแทน")
		return
	}
	m, err := h.Store.UpdateMemberRole(r.Context(), id, req.Role)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "ไม่พบสมาชิกคนนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// ---------- อุปกรณ์ ----------

// trimPtr ตัดช่องว่างหัวท้าย และทำให้ช่องว่างเปล่ากลายเป็น NULL ในฐานข้อมูล
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// validateEquipment ตรวจฝั่ง server ซ้ำอีกชั้น แม้ฟอร์ม React จะเช็กไปแล้ว เพราะ curl ไม่ผ่านฟอร์ม
// และทำความสะอาดค่า (ตัดช่องว่าง ว่าง = null) ในที่เดียวกัน
func validateEquipment(in *store.EquipmentInput) (code, message string) {
	in.Title = strings.TrimSpace(in.Title)
	in.TitleTh = trimPtr(in.TitleTh)
	in.Emoji = trimPtr(in.Emoji)
	in.Detail = trimPtr(in.Detail)
	in.Poster = trimPtr(in.Poster)

	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 255:
		return "BAD_TITLE", "ต้องมีชื่อรุ่น และยาวไม่เกิน 255 ตัวอักษร"
	case in.TitleTh != nil && utf8.RuneCountInString(*in.TitleTh) > 100:
		return "BAD_BRAND", "ยี่ห้อยาวไม่เกิน 100 ตัวอักษร"
	case in.CategoryID != nil && *in.CategoryID <= 0:
		return "BAD_CATEGORY", "รหัสหมวดต้องเป็นเลขบวก"
	case in.Year != nil && (*in.Year < 1990 || *in.Year > 2100):
		return "BAD_YEAR", "ปีเปิดตัวต้องอยู่ระหว่าง 1990 ถึง 2100"
	case in.Rating != nil && (*in.Rating < 0 || *in.Rating > 10):
		return "BAD_RATING", "คะแนนต้องอยู่ระหว่าง 0 ถึง 10"
	case in.Price != nil && *in.Price < 0:
		return "BAD_PRICE", "ราคาต้องไม่ติดลบ"
	case in.Emoji != nil && utf8.RuneCountInString(*in.Emoji) > 8:
		return "BAD_EMOJI", "ไอคอนยาวไม่เกิน 8 ตัวอักษร"
	case in.Detail != nil && utf8.RuneCountInString(*in.Detail) > 5000:
		return "BAD_DETAIL", "รายละเอียดยาวไม่เกิน 5000 ตัวอักษร"
	case in.Poster != nil && (len(*in.Poster) > 500 ||
		(!strings.HasPrefix(*in.Poster, "https://") && !strings.HasPrefix(*in.Poster, "http://"))):
		return "BAD_POSTER", "รูปต้องเป็น URL เต็มที่ขึ้นต้นด้วย https:// หรือ http:// และยาวไม่เกิน 500 ตัวอักษร"
	}
	return "", ""
}

func (h *Handler) AdminGetEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	a, err := h.Store.AdminGetEquipment(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์รุ่นนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) AdminCreateEquipment(w http.ResponseWriter, r *http.Request) {
	var in store.EquipmentInput
	if err := decodeJSON(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	if code, msg := validateEquipment(&in); code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}
	a, err := h.Store.CreateEquipment(r.Context(), in)
	if errors.Is(err, store.ErrBadCategory) {
		writeError(w, http.StatusBadRequest, "BAD_CATEGORY", "ไม่มีหมวดรหัสนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handler) AdminUpdateEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	var in store.EquipmentInput
	if err := decodeJSON(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	if code, msg := validateEquipment(&in); code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}
	a, err := h.Store.UpdateEquipment(r.Context(), id, in)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์รุ่นนี้")
		return
	case errors.Is(err, store.ErrBadCategory):
		writeError(w, http.StatusBadRequest, "BAD_CATEGORY", "ไม่มีหมวดรหัสนี้")
		return
	case err != nil:
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) AdminDeleteEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	err := h.Store.DeleteEquipment(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์รุ่นนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // ลบแล้ว ไม่มีอะไรให้ส่งกลับ
}

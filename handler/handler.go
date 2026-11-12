package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"techreview/auth"
	"techreview/store"
)

type Handler struct {
	Store   *store.Store
	Tickets *auth.Issuer
}

// maxBody จำกัดขนาด JSON ที่รับ (1 MB) กันคนยิงข้อมูลก้อนใหญ่เข้ามากินหน่วยความจำ
const maxBody = 1 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

// internal จดรายละเอียดไว้ใน log ฝั่งเรา แต่บอกผู้ใช้แค่ว่าระบบขัดข้อง
func internal(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL", "ระบบขัดข้อง ลองใหม่อีกครั้ง")
}

// pathID อ่านเลขจาก URL เช่น /api/equipment/{id} ต้องเป็นจำนวนเต็มบวก
func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

// decodeJSON อ่าน body เป็น JSON โดยจำกัดขนาด
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	return json.NewDecoder(r.Body).Decode(v)
}

// badBody ตอบ error ของ decodeJSON: ใหญ่เกิน = 413 นอกนั้น = 400
func badBody(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		writeError(w, http.StatusRequestEntityTooLarge, "TOO_LARGE", "ข้อมูลที่ส่งมาใหญ่เกินไป")
		return
	}
	writeError(w, http.StatusBadRequest, "BAD_JSON", "อ่านกล่อง JSON ไม่ออก")
}

// Health ใช้ให้ Render / load balancer เช็กว่าโปรเซสยังอยู่ (ไม่แตะฐานข้อมูล)
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

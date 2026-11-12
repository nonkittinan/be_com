package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"techreview/auth"
	"techreview/mw"
	"techreview/store"
)

type registerReq struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// dummyHash ใช้เทียบรหัสแก้ขัดตอนไม่พบอีเมล เพื่อให้ใช้เวลาใกล้เคียงกับกรณีพบอีเมล
// (ไม่งั้นวัดเวลาตอบแล้วรู้ได้ว่าอีเมลไหนสมัครไว้แล้ว)
var dummyHash, _ = auth.HashPassword("timing-equalizer-not-a-real-password")

// validEmail ยอมรับเฉพาะรูปแบบ "ชื่อ@โดเมน" ล้วน ๆ (ไม่รับ "Name <a@b.c>")
func validEmail(email string) bool {
	if len(email) > 255 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

// validateRegister ตรวจข้อมูลสมัครสมาชิก คืน code/message ถ้าไม่ผ่าน (email ต้อง lower + trim มาแล้ว)
func validateRegister(email, password, name string) (code, message string) {
	switch {
	case !validEmail(email):
		return "BAD_EMAIL", "รูปแบบอีเมลไม่ถูกต้อง"
	case utf8.RuneCountInString(password) < 8 || len(password) > 72:
		return "BAD_PASSWORD", "รหัสผ่านต้องยาว 8 ตัวอักษรขึ้นไป และไม่เกิน 72 ไบต์"
	case name == "" || utf8.RuneCountInString(name) > 50:
		return "BAD_NAME", "ต้องมีชื่อที่แสดง และยาวไม่เกิน 50 ตัวอักษร"
	}
	return "", ""
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := decodeJSON(w, r, &req); err != nil {
		badBody(w, err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.DisplayName)
	if code, msg := validateRegister(email, req.Password, name); code != "" {
		writeError(w, http.StatusBadRequest, code, msg)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		internal(w, err)
		return
	}
	m, err := h.Store.CreateMember(r.Context(), email, hash, name)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "EMAIL_TAKEN", "อีเมลนี้มีบัญชีอยู่แล้ว")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := decodeJSON(w, r, &req); err != nil {
		badBody(w, err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	m, err := h.Store.MemberByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		auth.CheckPassword(dummyHash, req.Password) // กินเวลาเท่ากัน ผลไม่ใช้
		writeError(w, http.StatusUnauthorized, "BAD_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if !auth.CheckPassword(m.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "BAD_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		return
	}
	token, err := h.Tickets.Issue(m.ID, m.Role)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "member": m})
}

// Me ตอบข้อมูลของคนที่ถือตั๋ว เลขสมาชิกมาจากตั๋วที่ RequireLogin ตรวจแล้วเท่านั้น
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	m, err := h.Store.MemberByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "MEMBER_GONE", "ไม่พบบัญชีนี้แล้ว กรุณาเข้าสู่ระบบใหม่")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

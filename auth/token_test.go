package auth

import (
	"strings"
	"testing"
	"time"
)

func newIssuer(ttl time.Duration) *Issuer {
	return &Issuer{Secret: []byte("test-secret-that-is-long-enough-123456"), TTL: ttl}
}

func TestIssueThenParseRoundTrip(t *testing.T) {
	iss := newIssuer(time.Hour)
	tok, err := iss.Issue(42, "admin")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	id, role, err := iss.Parse(tok)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if id != 42 || role != "admin" {
		t.Fatalf("got id=%d role=%q, want 42 admin", id, role)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	iss := newIssuer(-time.Minute)
	tok, _ := iss.Issue(1, "member")
	if _, _, err := iss.Parse(tok); err == nil {
		t.Fatal("ตั๋วหมดอายุต้องไม่ผ่าน")
	}
}

func TestParseRejectsTokenSignedWithOtherSecret(t *testing.T) {
	tok, _ := newIssuer(time.Hour).Issue(1, "admin")
	other := &Issuer{Secret: []byte("another-secret-that-is-long-enough-99999"), TTL: time.Hour}
	if _, _, err := other.Parse(tok); err == nil {
		t.Fatal("ตั๋วที่เซ็นด้วยกุญแจอื่นต้องไม่ผ่าน")
	}
}

func TestParseRejectsTamperedPayload(t *testing.T) {
	iss := newIssuer(time.Hour)
	tok, _ := iss.Issue(1, "member")
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token ควรมี 3 ส่วน ได้ %d", len(parts))
	}
	// เปลี่ยน payload แต่คงลายเซ็นเดิม
	tampered := parts[0] + "." + parts[1] + "x." + parts[2]
	if _, _, err := iss.Parse(tampered); err == nil {
		t.Fatal("ตั๋วที่ถูกแก้ต้องไม่ผ่าน")
	}
}

func TestParseRejectsAlgNone(t *testing.T) {
	iss := newIssuer(time.Hour)
	// header {"alg":"none","typ":"JWT"} payload {"sub":"1","role":"admin","exp":99999999999}
	none := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxIiwicm9sZSI6ImFkbWluIiwiZXhwIjo5OTk5OTk5OTk5OX0."
	if _, _, err := iss.Parse(none); err == nil {
		t.Fatal("alg none ต้องไม่ผ่าน")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, _, err := newIssuer(time.Hour).Parse("not-a-token"); err == nil {
		t.Fatal("ข้อความมั่วต้องไม่ผ่าน")
	}
}

func TestPasswordHashAndCheck(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h == "correct horse battery" {
		t.Fatal("ห้ามเก็บรหัสผ่านตรง ๆ")
	}
	if !CheckPassword(h, "correct horse battery") {
		t.Fatal("รหัสถูกต้องควรผ่าน")
	}
	if CheckPassword(h, "wrong password") {
		t.Fatal("รหัสผิดต้องไม่ผ่าน")
	}
}

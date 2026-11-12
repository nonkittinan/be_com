package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"techreview/auth"
	"techreview/handler"
)

func TestLoadEnvLoadsKeyValuePairsFromDotEnv(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := "# comment\nJWT_SECRET=abcdefghijklmnopqrstuvwxyz123456\nDATABASE_URL='postgres://localhost:5432/test'\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("JWT_SECRET", "")
	os.Unsetenv("JWT_SECRET") // t.Setenv จะคืนค่าเดิมให้ตอนจบเทสต์
	t.Setenv("DATABASE_URL", "")
	os.Unsetenv("DATABASE_URL")

	if err := loadEnv(envPath); err != nil {
		t.Fatalf("load env: %v", err)
	}
	if got := os.Getenv("JWT_SECRET"); got != "abcdefghijklmnopqrstuvwxyz123456" {
		t.Fatalf("JWT_SECRET = %q", got)
	}
	if got := os.Getenv("DATABASE_URL"); got != "postgres://localhost:5432/test" {
		t.Fatalf("DATABASE_URL = %q (ต้องตัดเครื่องหมายคำพูดออก)", got)
	}
}

func TestLoadEnvDoesNotOverrideExistingVariable(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(envPath, []byte("PORT=1111\n"), 0o600)
	t.Setenv("PORT", "2222")
	if err := loadEnv(envPath); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("PORT"); got != "2222" {
		t.Fatalf("PORT = %q ค่าที่ตั้งไว้แล้วต้องชนะไฟล์ .env", got)
	}
}

func TestLoadEnvMissingFileIsNotAnError(t *testing.T) {
	if err := loadEnv(filepath.Join(t.TempDir(), "nope.env")); err != nil {
		t.Fatalf("ไม่มีไฟล์ .env ต้องไม่ error (บน Render ไม่มีไฟล์): %v", err)
	}
}

// ด่านสิทธิ์ทำงานก่อนแตะฐานข้อมูล จึงเทสต์ได้ด้วย Store ว่าง: ถ้าด่านหลุด handler จะ panic ทันที
func TestAdminAndMemberRoutesAreGuarded(t *testing.T) {
	tickets := &auth.Issuer{Secret: []byte("test-secret-that-is-long-enough-123456"), TTL: time.Hour}
	mux := routes(&handler.Handler{Tickets: tickets}, tickets)

	memberTok, _ := tickets.Issue(7, "member")

	adminRoutes := []struct{ method, path string }{
		{"GET", "/api/admin/stats"},
		{"GET", "/api/admin/members"},
		{"PUT", "/api/admin/members/1/role"},
		{"POST", "/api/admin/equipment"},
		{"GET", "/api/admin/equipment/1"},
		{"PUT", "/api/admin/equipment/1"},
		{"DELETE", "/api/admin/equipment/1"},
		{"POST", "/api/admin/movies"}, // ชื่อเดิมต้องถูกป้องกันเท่ากัน
		{"DELETE", "/api/admin/movies/1"},
	}
	for _, rt := range adminRoutes {
		// ไม่มีตั๋ว = 401
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(rt.method, rt.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s ไม่มีตั๋ว: ได้ %d ต้องเป็น 401", rt.method, rt.path, rec.Code)
		}
		// ตั๋ว member = 403
		req := httptest.NewRequest(rt.method, rt.path, nil)
		req.Header.Set("Authorization", "Bearer "+memberTok)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s ตั๋ว member: ได้ %d ต้องเป็น 403", rt.method, rt.path, rec.Code)
		}
	}

	loginRoutes := []struct{ method, path string }{
		{"GET", "/api/me"},
		{"GET", "/api/me/wishlist"},
		{"PUT", "/api/me/wishlist/1"},
		{"DELETE", "/api/me/wishlist/1"},
		{"POST", "/api/equipment/1/reviews"},
		{"PUT", "/api/equipment/1/vote"},
		{"DELETE", "/api/equipment/1/vote"},
		{"DELETE", "/api/reviews/1"},
		{"POST", "/api/movies/1/reviews"},
	}
	for _, rt := range loginRoutes {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(rt.method, rt.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s ไม่มีตั๋ว: ได้ %d ต้องเป็น 401", rt.method, rt.path, rec.Code)
		}
	}
}

func TestHealthz(t *testing.T) {
	tickets := &auth.Issuer{Secret: []byte("test-secret-that-is-long-enough-123456"), TTL: time.Hour}
	mux := routes(&handler.Handler{Tickets: tickets}, tickets)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz = %d", rec.Code)
	}
}

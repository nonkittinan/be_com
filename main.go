package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"techreview/auth"
	"techreview/handler"
	"techreview/mw"
	"techreview/store"
)

// loadEnv อ่านไฟล์ .env (ถ้ามี) ใส่เป็น environment variable โดยไม่ทับค่าที่ตั้งไว้แล้ว
// บน Render ไม่มีไฟล์ ก็เงียบ แล้วใช้ค่าที่ตั้งในหน้า Environment แทน
func loadEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// routes ประกอบเส้นทางทั้งหมด แยกออกมาเพื่อให้เทสต์สร้างได้โดยไม่ต้องต่อฐานข้อมูล
func routes(h *handler.Handler, tickets *auth.Issuer) *http.ServeMux {
	// login = ต้องมีตั๋ว, admin = ต้องมีตั๋วและบทบาทเป็น admin (ตรวจตั๋วก่อนเสมอ ค่อยดูบทบาท)
	login := func(next http.HandlerFunc) http.HandlerFunc { return mw.RequireLogin(tickets, next) }
	admin := func(next http.HandlerFunc) http.HandlerFunc { return login(mw.RequireRole("admin", next)) }
	maybe := func(next http.HandlerFunc) http.HandlerFunc { return mw.OptionalLogin(tickets, next) }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Health)

	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/me", login(h.Me))

	// Wishlist ของตัวเอง
	mux.HandleFunc("GET /api/me/wishlist", login(h.GetWishlist))
	mux.HandleFunc("PUT /api/me/wishlist/{equipmentId}", login(h.PutWishlist))
	mux.HandleFunc("DELETE /api/me/wishlist/{equipmentId}", login(h.DeleteWishlist))

	mux.HandleFunc("DELETE /api/reviews/{reviewId}", login(h.DeleteReview))

	// หน้าเว็บ (ฟรอนต์เอนด์) ยังเรียกชื่อเดิมของโปรเจกต์ MovieHub (/api/movies, /api/genres)
	// จึงเปิดไว้ทั้ง 2 ชื่อ ให้ใช้ได้ทันทีโดยไม่ต้องแก้ฟรอนต์ ชื่อเดิมลบทิ้งได้เมื่อฟรอนต์เปลี่ยนแล้ว
	for _, p := range []string{"/api/equipment", "/api/movies"} {
		mux.HandleFunc("GET "+p, h.SearchEquipment)
		mux.HandleFunc("GET "+p+"/{id}", maybe(h.GetEquipment))
		mux.HandleFunc("GET "+p+"/{id}/reviews", h.ListReviews)
		mux.HandleFunc("POST "+p+"/{id}/reviews", login(h.CreateReview))
		mux.HandleFunc("PUT "+p+"/{id}/vote", login(h.PutVote))
		mux.HandleFunc("DELETE "+p+"/{id}/vote", login(h.DeleteVote))
	}
	for _, p := range []string{"/api/categories", "/api/genres"} {
		mux.HandleFunc("GET "+p, h.Categories)
	}

	// หลังบ้าน: ทุกเส้นทางใต้ /api/admin ผ่าน admin(...) ไม่มีข้อยกเว้น
	mux.HandleFunc("GET /api/admin/stats", admin(h.AdminStats))
	mux.HandleFunc("GET /api/admin/members", admin(h.AdminMembers))
	mux.HandleFunc("PUT /api/admin/members/{id}/role", admin(h.AdminSetRole))
	for _, p := range []string{"/api/admin/equipment", "/api/admin/movies"} {
		mux.HandleFunc("POST "+p, admin(h.AdminCreateEquipment))
		mux.HandleFunc("GET "+p+"/{id}", admin(h.AdminGetEquipment))
		mux.HandleFunc("PUT "+p+"/{id}", admin(h.AdminUpdateEquipment))
		mux.HandleFunc("DELETE "+p+"/{id}", admin(h.AdminDeleteEquipment))
	}
	return mux
}

func main() {
	if err := loadEnv(".env"); err != nil {
		log.Fatal("ไม่สามารถอ่านไฟล์ .env ได้: ", err)
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL ยังไม่ได้ตั้งค่า")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("ต่อฐานข้อมูลไม่ติด ", err)
	}

	tickets := &auth.Issuer{Secret: []byte(secret), TTL: time.Hour}
	h := &handler.Handler{Store: &store.Store{Pool: pool}, Tickets: tickets}

	origins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	if strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")) == "" {
		origins = []string{"http://localhost:3000"}
	}

	port := os.Getenv("PORT") // Render ส่งมาให้ ในเครื่องไม่มีก็ใช้ 8080
	if port == "" {
		port = "8080"
	}

	// ตั้ง timeout ทุกขา กันคนเปิดคอนเนกชันค้างไว้เฉย ๆ จนเซิร์ฟเวอร์หมดแรง
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mw.Logging(mw.CORS(origins, routes(h, tickets))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("TechReview API ฟังอยู่ที่ :"+port, "ต้อนรับ", origins)
	log.Fatal(srv.ListenAndServe())
}

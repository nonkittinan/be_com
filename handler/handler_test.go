package handler

import (
	"strings"
	"testing"

	"techreview/store"
)

func ptr[T any](v T) *T { return &v }

func TestValidateRegister(t *testing.T) {
	cases := []struct {
		name, email, pw, display, wantCode string
	}{
		{"ปกติ", "a@b.co", "12345678", "สมชาย", ""},
		{"อีเมลไม่มี @", "abc", "12345678", "x", "BAD_EMAIL"},
		{"อีเมลมีชื่อนำหน้า", "Bob <a@b.co>", "12345678", "x", "BAD_EMAIL"},
		{"อีเมลยาวเกิน 255", strings.Repeat("a", 251) + "@b.co", "12345678", "x", "BAD_EMAIL"},
		{"อีเมลยาว 255 พอดี", strings.Repeat("a", 250) + "@b.co", "12345678", "x", ""},
		{"รหัสสั้น", "a@b.co", "1234567", "x", "BAD_PASSWORD"},
		{"รหัสเกิน 72 ไบต์", "a@b.co", strings.Repeat("a", 73), "x", "BAD_PASSWORD"},
		{"ไม่มีชื่อ", "a@b.co", "12345678", "", "BAD_NAME"},
		{"ชื่อยาวเกิน 50 ตัวอักษร", "a@b.co", "12345678", strings.Repeat("ก", 51), "BAD_NAME"},
		{"ชื่อไทย 50 ตัวพอดี", "a@b.co", "12345678", strings.Repeat("ก", 50), ""},
	}
	for _, c := range cases {
		if code, _ := validateRegister(c.email, c.pw, c.display); code != c.wantCode {
			t.Errorf("%s: code=%q want %q", c.name, code, c.wantCode)
		}
	}
}

func TestValidateReviewText(t *testing.T) {
	if _, code, _ := validateReviewText("สั้นไป"); code != "BAD_TEXT" {
		t.Error("รีวิวสั้นกว่า 10 ตัวอักษรต้องไม่ผ่าน")
	}
	if _, code, _ := validateReviewText("          x          "); code != "BAD_TEXT" {
		t.Error("ช่องว่างล้วนต้องไม่ถูกนับเป็นความยาว")
	}
	if text, code, _ := validateReviewText("  การ์ดจอตัวนี้เล่นเกมลื่นมาก  "); code != "" || text != "การ์ดจอตัวนี้เล่นเกมลื่นมาก" {
		t.Errorf("รีวิวปกติต้องผ่านและถูก trim: %q %q", text, code)
	}
	if _, code, _ := validateReviewText(strings.Repeat("ก", 2001)); code != "BAD_TEXT" {
		t.Error("รีวิวเกิน 2000 ต้องไม่ผ่าน")
	}
	if _, code, _ := validateReviewText(strings.Repeat("ก", 2000)); code != "" {
		t.Error("รีวิว 2000 ตัวพอดีต้องผ่าน")
	}
}

func TestValidScore(t *testing.T) {
	for _, s := range []int{1, 5, 10} {
		if !validScore(&s) {
			t.Errorf("score %d ต้องผ่าน", s)
		}
	}
	for _, s := range []int{0, -1, 11, 100} {
		if validScore(&s) {
			t.Errorf("score %d ต้องไม่ผ่าน", s)
		}
	}
	if validScore(nil) {
		t.Error("ไม่ส่ง score ต้องไม่ผ่าน")
	}
}

func TestCleanNote(t *testing.T) {
	if n, ok := cleanNote(nil); n != nil || !ok {
		t.Error("nil ต้องได้ nil, ok")
	}
	if n, ok := cleanNote(ptr("   ")); n != nil || !ok {
		t.Error("ช่องว่างล้วน = ไม่มีโน้ต")
	}
	if n, ok := cleanNote(ptr("  ซื้อเดือนหน้า ")); !ok || n == nil || *n != "ซื้อเดือนหน้า" {
		t.Error("ต้อง trim")
	}
	if _, ok := cleanNote(ptr(strings.Repeat("ก", 201))); ok {
		t.Error("โน้ตเกิน 200 ต้องไม่ผ่าน")
	}
}

func validInput() store.EquipmentInput {
	return store.EquipmentInput{Title: "RTX 4060", TitleTh: ptr("NVIDIA"), CategoryID: ptr(1),
		Year: ptr(2023), Rating: ptr(8.0), Price: ptr(10900), Emoji: ptr("🎮"), Detail: ptr("ดี"),
		Poster: ptr("https://example.com/a.jpg")}
}

func TestValidateEquipment(t *testing.T) {
	in := validInput()
	if code, msg := validateEquipment(&in); code != "" {
		t.Fatalf("ข้อมูลปกติต้องผ่าน: %s %s", code, msg)
	}

	cases := []struct {
		name, want string
		mutate     func(*store.EquipmentInput)
	}{
		{"ไม่มีชื่อ", "BAD_TITLE", func(i *store.EquipmentInput) { i.Title = "   " }},
		{"ชื่อยาวเกิน", "BAD_TITLE", func(i *store.EquipmentInput) { i.Title = strings.Repeat("a", 256) }},
		{"ยี่ห้อยาวเกิน", "BAD_BRAND", func(i *store.EquipmentInput) { i.TitleTh = ptr(strings.Repeat("a", 101)) }},
		{"หมวดติดลบ", "BAD_CATEGORY", func(i *store.EquipmentInput) { i.CategoryID = ptr(-1) }},
		{"ปีน้อยไป", "BAD_YEAR", func(i *store.EquipmentInput) { i.Year = ptr(1980) }},
		{"ปีมากไป", "BAD_YEAR", func(i *store.EquipmentInput) { i.Year = ptr(2101) }},
		{"คะแนนเกิน 10", "BAD_RATING", func(i *store.EquipmentInput) { i.Rating = ptr(10.1) }},
		{"คะแนนติดลบ", "BAD_RATING", func(i *store.EquipmentInput) { i.Rating = ptr(-0.1) }},
		{"ราคาติดลบ", "BAD_PRICE", func(i *store.EquipmentInput) { i.Price = ptr(-1) }},
		{"ไอคอนยาวเกิน", "BAD_EMOJI", func(i *store.EquipmentInput) { i.Emoji = ptr(strings.Repeat("a", 9)) }},
		{"รูปไม่ใช่ URL", "BAD_POSTER", func(i *store.EquipmentInput) { i.Poster = ptr("/abc.jpg") }},
		{"รูปเป็น javascript:", "BAD_POSTER", func(i *store.EquipmentInput) { i.Poster = ptr("javascript:alert(1)") }},
	}
	for _, c := range cases {
		in := validInput()
		c.mutate(&in)
		if code, _ := validateEquipment(&in); code != c.want {
			t.Errorf("%s: code=%q want %q", c.name, code, c.want)
		}
	}
}

func TestValidateEquipmentCleansBlankFieldsToNil(t *testing.T) {
	in := store.EquipmentInput{Title: "  Cloud II  ", TitleTh: ptr("  "), Detail: ptr(""), Poster: ptr("   "), Emoji: ptr(" ")}
	if code, msg := validateEquipment(&in); code != "" {
		t.Fatalf("%s %s", code, msg)
	}
	if in.Title != "Cloud II" {
		t.Errorf("title ต้อง trim: %q", in.Title)
	}
	if in.TitleTh != nil || in.Detail != nil || in.Poster != nil || in.Emoji != nil {
		t.Error("ช่องว่างเปล่าต้องกลายเป็น nil (NULL ในฐานข้อมูล)")
	}
}

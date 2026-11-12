# build แบบ 2 ชั้น: ชั้นแรกคอมไพล์ ชั้นสองมีแค่ไฟล์ที่รันได้ (เล็ก และไม่มีคอมไพเลอร์/ซอร์สติดไป)
# go.mod ระบุ go 1.27.1 ถ้า build ฟ้องว่า image เก่าไป ให้เปลี่ยน GO_VERSION ให้ตรงกับที่ใช้ในเครื่อง
ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# ก็อป go.mod/go.sum ก่อน เพื่อให้ชั้นดาวน์โหลดไลบรารีถูก cache ไว้ ถ้าแก้แค่โค้ดไม่ต้องโหลดใหม่
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/techreview .

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=build /out/techreview /app/techreview

# ไม่มีไฟล์ .env ใน image (ดู .dockerignore) ค่าทั้งหมดมาจาก environment ที่ docker-compose / Render ส่งให้
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:${PORT}/healthz >/dev/null || exit 1

CMD ["/app/techreview"]

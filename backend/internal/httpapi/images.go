package httpapi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxImageBytes = 8 << 20

var uploadNamePattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

func registerImageRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("POST /api/v1/uploads", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+1024*1024)
		if r.ParseMultipartForm(maxImageBytes) != nil {
			writeJSON(w, 400, map[string]string{"error": "图片超过 8 MB 或表单无效"})
			return
		}
		defer r.MultipartForm.RemoveAll()
		f, header, err := r.FormFile("image")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "请选择一张图片"})
			return
		}
		defer f.Close()
		if header.Size > maxImageBytes {
			writeJSON(w, 400, map[string]string{"error": "图片不能超过 8 MB"})
			return
		}
		raw, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
		if err != nil || len(raw) > maxImageBytes {
			writeJSON(w, 400, map[string]string{"error": "图片不能超过 8 MB"})
			return
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 6000 || cfg.Height > 6000 || int64(cfg.Width)*int64(cfg.Height) > 25_000_000 {
			writeJSON(w, 400, map[string]string{"error": "仅支持不超过 6000×6000 的 JPEG 或 PNG 图片"})
			return
		}
		decoded, format, err := image.Decode(bytes.NewReader(raw))
		if err != nil || (format != "jpeg" && format != "png") {
			writeJSON(w, 400, map[string]string{"error": "图片文件无效"})
			return
		}
		id := make([]byte, 16)
		if _, err = rand.Read(id); err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法保存图片"})
			return
		}
		name := hex.EncodeToString(id) + ".jpg"
		dir := uploadDir()
		if err = os.MkdirAll(dir, 0700); err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法保存图片"})
			return
		}
		path := filepath.Join(dir, name)
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法保存图片"})
			return
		}
		canvas := image.NewRGBA(decoded.Bounds())
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(canvas, canvas.Bounds(), decoded, decoded.Bounds().Min, draw.Over)
		encodeErr := jpeg.Encode(out, canvas, &jpeg.Options{Quality: 88})
		closeErr := out.Close()
		if encodeErr != nil || closeErr != nil {
			_ = os.Remove(path)
			writeJSON(w, 500, map[string]string{"error": "无法处理图片"})
			return
		}
		writeJSON(w, 201, map[string]string{"image_url": "/uploads/" + name, "privacy_reminder": "发布前请遮挡图片中的姓名、学号和聊天对象等个人信息。"})
	}))))
	mux.Handle("GET /uploads/{name}", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !uploadNamePattern.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(filepath.Join(uploadDir(), name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = io.Copy(w, file)
	}))))
}

func uploadDir() string {
	if d := os.Getenv("UPLOAD_DIR"); d != "" {
		return d
	}
	return "./uploads"
}

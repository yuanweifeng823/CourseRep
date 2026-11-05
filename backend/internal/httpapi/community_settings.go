package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

type communitySettings struct {
	ContentReviewSLAHours *int   `json:"content_review_sla_hours"`
	CommunityGuidelines   string `json:"community_guidelines"`
}

func registerCommunitySettingsRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/community-guidelines", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var settings communitySettings
		if err := pool.QueryRow(r.Context(), `SELECT content_review_sla_hours, community_guidelines FROM community_settings WHERE id=1`).Scan(&settings.ContentReviewSLAHours, &settings.CommunityGuidelines); err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取社区信息失败"})
			return
		}
		writeJSON(w, 200, settings)
	}))))
	mux.Handle("PUT /api/v1/admin/community-settings", authenticate(pool, requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in communitySettings
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 12*1024)).Decode(&in) != nil || utf8.RuneCountInString(in.CommunityGuidelines) > 10000 || (in.ContentReviewSLAHours != nil && (*in.ContentReviewSLAHours < 1 || *in.ContentReviewSLAHours > 720)) {
			writeJSON(w, 400, map[string]string{"error": "处理时限需为 1 至 720 小时，社区规范最多 10,000 字"})
			return
		}
		in.CommunityGuidelines = strings.TrimSpace(in.CommunityGuidelines)
		u := userFromContext(r.Context())
		_, err := pool.Exec(r.Context(), `UPDATE community_settings SET content_review_sla_hours=$1, community_guidelines=$2, updated_by=$3, updated_at=now() WHERE id=1`, in.ContentReviewSLAHours, in.CommunityGuidelines, u.ID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "保存社区设置失败"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "updated"})
	}))))
}

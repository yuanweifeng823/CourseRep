package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerModerationRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("POST /api/v1/reports", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ReviewID *int64 `json:"review_id"`
			PostID   *int64 `json:"post_id"`
			Category string `json:"category"`
			Detail   string `json:"detail"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil || ((in.ReviewID == nil) == (in.PostID == nil)) || len([]rune(strings.TrimSpace(in.Detail))) > 900 {
			writeJSON(w, 400, map[string]string{"error": "举报内容无效"})
			return
		}
		allowed := map[string]bool{"spam": true, "academic_integrity": true, "copyright": true, "personal_attack": true, "teacher_claim": true, "other": true}
		if !allowed[in.Category] {
			writeJSON(w, 400, map[string]string{"error": "请选择有效举报原因"})
			return
		}
		u := userFromContext(r.Context())
		reason := in.Category
		if strings.TrimSpace(in.Detail) != "" {
			reason += ": " + strings.TrimSpace(in.Detail)
		}
		var id int64
		var err error
		if in.ReviewID != nil {
			err = pool.QueryRow(r.Context(), `INSERT INTO reports(reporter_id,review_id,reason) SELECT $1,id,$2 FROM course_reviews WHERE id=$3 AND status='published' RETURNING id`, u.ID, reason, *in.ReviewID).Scan(&id)
		} else {
			err = pool.QueryRow(r.Context(), `INSERT INTO reports(reporter_id,post_id,reason) SELECT $1,id,$2 FROM course_posts WHERE id=$3 AND status='published' RETURNING id`, u.ID, reason, *in.PostID).Scan(&id)
		}
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "内容不存在或暂不可举报"})
			return
		}
		writeJSON(w, 201, map[string]any{"report_id": id, "status": "open"})
	}))))
	mux.Handle("GET /api/v1/admin/reports", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, e := queryInt(r, "page", 1, 1, 1_000_000)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "page 无效"})
			return
		}
		size, e := queryInt(r, "page_size", 20, 1, 100)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "page_size 无效"})
			return
		}
		state := r.URL.Query().Get("status")
		if state == "" {
			state = "open"
		}
		if state != "open" && state != "reviewing" && state != "resolved" && state != "dismissed" {
			writeJSON(w, 400, map[string]string{"error": "状态无效"})
			return
		}
		rows, e := pool.Query(r.Context(), `SELECT r.id,r.reason,r.status,r.created_at,CASE WHEN r.review_id IS NOT NULL THEN 'review' ELSE 'post' END,coalesce(r.review_id,r.post_id),coalesce(rv.body,p.body),CASE WHEN r.review_id IS NOT NULL OR p.display_identity='anonymous' THEN '同校已验证' ELSE u.nickname END FROM reports r LEFT JOIN course_reviews rv ON rv.id=r.review_id LEFT JOIN course_posts p ON p.id=r.post_id JOIN users u ON u.id=coalesce(rv.author_id,p.author_id) WHERE r.status=$1 ORDER BY r.created_at,r.id LIMIT $2 OFFSET $3`, state, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取举报队列失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID         int64  `json:"id"`
			Reason     string `json:"reason"`
			Status     string `json:"status"`
			Created    any    `json:"created_at"`
			TargetType string `json:"target_type"`
			TargetID   int64  `json:"target_id"`
			Body       string `json:"body"`
			Author     string `json:"author"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.Reason, &x.Status, &x.Created, &x.TargetType, &x.TargetID, &x.Body, &x.Author) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取举报队列失败"})
				return
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	}))))
	mux.Handle("POST /api/v1/admin/reports/{reportID}/decision", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reportID, e := strconv.ParseInt(r.PathValue("reportID"), 10, 64)
		if e != nil || reportID < 1 {
			writeJSON(w, 400, map[string]string{"error": "举报编号无效"})
			return
		}
		var in struct {
			Decision string `json:"decision"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil || (in.Decision != "resolve" && in.Decision != "dismiss") || (in.Action != "none" && in.Action != "remove_content" && in.Action != "disable_author") || (in.Decision == "dismiss" && in.Action != "none") || len([]rune(strings.TrimSpace(in.Reason))) < 2 || len([]rune(in.Reason)) > 500 {
			writeJSON(w, 400, map[string]string{"error": "处理决定、动作或原因无效"})
			return
		}
		u := userFromContext(r.Context())
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法处理举报"})
			return
		}
		defer tx.Rollback(r.Context())
		var reporter int64
		var reviewID, postID *int64
		err = tx.QueryRow(r.Context(), `UPDATE reports SET status=CASE WHEN $2='resolve' THEN 'resolved' ELSE 'dismissed' END WHERE id=$1 AND status IN ('open','reviewing') RETURNING reporter_id,review_id,post_id`, reportID, in.Decision).Scan(&reporter, &reviewID, &postID)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "待处理举报不存在"})
			return
		}
		action := in.Action
		if action == "remove_content" {
			var owner int64
			if reviewID != nil {
				err = tx.QueryRow(r.Context(), `SELECT author_id FROM course_reviews WHERE id=$1`, *reviewID).Scan(&owner)
				var result pgconn.CommandTag
				if err == nil {
					result, err = tx.Exec(r.Context(), `UPDATE course_reviews SET status='removed',updated_at=now() WHERE id=$1 AND status='published'`, *reviewID)
				}
				if err == nil && result.RowsAffected() > 0 {
					_, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'review_moderation',jsonb_build_object('review_id',$2::bigint,'status','removed'))`, owner, *reviewID)
				}
			} else {
				err = tx.QueryRow(r.Context(), `SELECT author_id FROM course_posts WHERE id=$1`, *postID).Scan(&owner)
				var result pgconn.CommandTag
				if err == nil {
					result, err = tx.Exec(r.Context(), `UPDATE course_posts SET status='removed',updated_at=now() WHERE id=$1 AND status='published'`, *postID)
				}
				if err == nil && result.RowsAffected() > 0 {
					_, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'post_moderation',jsonb_build_object('post_id',$2::bigint,'status','removed'))`, owner, *postID)
				}
			}
		}
		var owner int64
		if err == nil && in.Action == "disable_author" {
			if reviewID != nil {
				err = tx.QueryRow(r.Context(), `SELECT author_id FROM course_reviews WHERE id=$1`, *reviewID).Scan(&owner)
			} else {
				err = tx.QueryRow(r.Context(), `SELECT author_id FROM course_posts WHERE id=$1`, *postID).Scan(&owner)
			}
			if err == nil {
				_, err = tx.Exec(r.Context(), `UPDATE users SET status='disabled',updated_at=now() WHERE id=$1`, owner)
			}
			if err == nil {
				_, err = tx.Exec(r.Context(), `UPDATE refresh_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, owner)
			}
		}
		var rID, pID any
		if reviewID != nil {
			rID = *reviewID
		}
		if postID != nil {
			pID = *postID
		}
		if err == nil && in.Action != "none" {
			_, err = tx.Exec(r.Context(), `INSERT INTO moderation_actions(moderator_id,report_id,review_id,post_id,action,reason) VALUES($1,$2,$3,$4,$5,$6)`, u.ID, reportID, rID, pID, action, strings.TrimSpace(in.Reason))
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'report_result',jsonb_build_object('report_id',$2::bigint,'status',$3::text))`, reporter, reportID, in.Decision)
		}
		if err != nil || tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "举报处置事务失败"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": in.Decision, "action": action})
	}))))
	registerTagRoutes(mux, pool)
}

func registerTagRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/admin/topic-tags", authenticate(pool, requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `SELECT id,name,active FROM topic_tags ORDER BY active DESC,name`)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取话题标签失败"})
			return
		}
		defer rows.Close()
		type tag struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			Active bool   `json:"active"`
		}
		items := make([]tag, 0)
		for rows.Next() {
			var x tag
			if rows.Scan(&x.ID, &x.Name, &x.Active) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取话题标签失败"})
				return
			}
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取话题标签失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))))
	mux.Handle("GET /api/v1/topic-tags", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rows, e := pool.Query(r.Context(), `SELECT id,name FROM topic_tags WHERE active ORDER BY name`)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取话题标签失败"})
			return
		}
		defer rows.Close()
		type tag struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		}
		items := make([]tag, 0)
		for rows.Next() {
			var x tag
			if rows.Scan(&x.ID, &x.Name) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取话题标签失败"})
				return
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))))
	mux.Handle("POST /api/v1/admin/topic-tags", authenticate(pool, requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name string `json:"name"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "标签名称无效"})
			return
		}
		in.Name = strings.TrimSpace(in.Name)
		if len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 40 {
			writeJSON(w, 400, map[string]string{"error": "标签名称需为 1 到 40 个字符"})
			return
		}
		var id int64
		e := pool.QueryRow(r.Context(), `INSERT INTO topic_tags(name) VALUES($1) RETURNING id`, in.Name).Scan(&id)
		if e != nil {
			writeJSON(w, 409, map[string]string{"error": "标签已存在或无法创建"})
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "name": in.Name, "active": true})
	}))))
	mux.Handle("PATCH /api/v1/admin/topic-tags/{tagID}", authenticate(pool, requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, e := strconv.ParseInt(r.PathValue("tagID"), 10, 64)
		if e != nil || id < 1 {
			writeJSON(w, 400, map[string]string{"error": "标签编号无效"})
			return
		}
		var in struct {
			Name   *string `json:"name"`
			Active *bool   `json:"active"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in) != nil || (in.Name == nil && in.Active == nil) {
			writeJSON(w, 400, map[string]string{"error": "请提供标签名称或启用状态"})
			return
		}
		name := ""
		if in.Name != nil {
			name = strings.TrimSpace(*in.Name)
			if len([]rune(name)) < 1 || len([]rune(name)) > 40 {
				writeJSON(w, 400, map[string]string{"error": "标签名称无效"})
				return
			}
		}
		var finalName string
		var active bool
		e = pool.QueryRow(r.Context(), `UPDATE topic_tags SET name=coalesce(nullif($2,''),name),active=coalesce($3,active) WHERE id=$1 RETURNING name,active`, id, name, in.Active).Scan(&finalName, &active)
		if e != nil {
			writeJSON(w, 404, map[string]string{"error": "标签不存在或名称冲突"})
			return
		}
		writeJSON(w, 200, map[string]any{"id": id, "name": finalName, "active": active})
	}))))
}

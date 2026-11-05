package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postInput struct {
	Title           string   `json:"title"`
	Body            string   `json:"body"`
	ImageURLs       []string `json:"image_urls"`
	Tags            []string `json:"topic_tags"`
	AttendanceYear  *int     `json:"attendance_year"`
	DisplayIdentity string   `json:"display_identity"`
}

func registerPostRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/admin/posts", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		rows, e := pool.Query(r.Context(), `SELECT p.id,p.course_id,c.course_code,c.course_name,p.title,p.body,p.image_urls,p.topic_tags,p.attendance_year,p.display_identity,p.status,p.created_at,CASE WHEN p.display_identity='anonymous' THEN '同校已验证' ELSE u.nickname END FROM course_posts p JOIN courses c ON c.id=p.course_id JOIN users u ON u.id=p.author_id WHERE p.status IN ('submitted','under_review') ORDER BY p.created_at,p.id LIMIT $1 OFFSET $2`, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取待审经验帖失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID                  int64     `json:"id"`
			CourseID            int64     `json:"course_id"`
			CourseCode          string    `json:"course_code"`
			CourseName          string    `json:"course_name"`
			Title               string    `json:"title"`
			Body                string    `json:"body"`
			ImageURLs           []string  `json:"image_urls"`
			TopicTags           []string  `json:"topic_tags"`
			AttendanceYear      *int      `json:"attendance_year"`
			AttendanceYearLabel string    `json:"attendance_year_label"`
			DisplayIdentity     string    `json:"display_identity"`
			DisplayName         string    `json:"display_name"`
			Status              string    `json:"status"`
			CreatedAt           time.Time `json:"created_at"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.CourseID, &x.CourseCode, &x.CourseName, &x.Title, &x.Body, &x.ImageURLs, &x.TopicTags, &x.AttendanceYear, &x.DisplayIdentity, &x.Status, &x.CreatedAt, &x.DisplayName) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取待审经验帖失败"})
				return
			}
			x.AttendanceYearLabel = "年份未注明"
			if x.AttendanceYear != nil {
				x.AttendanceYearLabel = strconv.Itoa(*x.AttendanceYear)
			}
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取待审经验帖失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	}))))
	mux.Handle("GET /api/v1/courses/{courseID}/posts", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		courseID, ok := pathID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "课程编号无效"})
			return
		}
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
		year := strings.TrimSpace(r.URL.Query().Get("year"))
		tag := strings.TrimSpace(r.URL.Query().Get("tag"))
		sortBy := r.URL.Query().Get("sort")
		if sortBy == "" {
			sortBy = "newest"
		}
		if year != "" && year != "unspecified" {
			n, er := strconv.Atoi(year)
			if er != nil || n < 1990 || n > currentYear() {
				writeJSON(w, 400, map[string]string{"error": "年份筛选无效"})
				return
			}
		}
		if sortBy != "newest" && sortBy != "popular" {
			writeJSON(w, 400, map[string]string{"error": "排序方式无效"})
			return
		}
		order := "p.created_at DESC,p.id DESC"
		if sortBy == "popular" {
			order = "like_count DESC,p.created_at DESC,p.id DESC"
		}
		viewer := userFromContext(r.Context())
		query := `SELECT p.id,p.title,p.body,p.image_urls,p.topic_tags,p.attendance_year,p.display_identity,p.created_at,CASE WHEN p.display_identity='anonymous' THEN '同校已验证' ELSE u.nickname END,(SELECT count(*) FROM post_likes l WHERE l.post_id=p.id) AS like_count,EXISTS(SELECT 1 FROM post_likes l WHERE l.post_id=p.id AND l.user_id=$4),EXISTS(SELECT 1 FROM post_bookmarks b WHERE b.post_id=p.id AND b.user_id=$4) FROM course_posts p JOIN users u ON u.id=p.author_id WHERE p.course_id=$1 AND p.status='published' AND ($2='' OR ($2='unspecified' AND p.attendance_year IS NULL) OR p.attendance_year::text=$2) AND ($3='' OR $3=ANY(p.topic_tags)) ORDER BY ` + order + ` LIMIT $5 OFFSET $6`
		rows, e := pool.Query(r.Context(), query, courseID, year, tag, viewer.ID, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取课程经验失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID          int64    `json:"id"`
			Title       string   `json:"title"`
			Body        string   `json:"body"`
			Images      []string `json:"image_urls"`
			Tags        []string `json:"topic_tags"`
			Year        *int     `json:"attendance_year"`
			YearLabel   string   `json:"attendance_year_label"`
			Identity    string   `json:"display_identity"`
			Created     any      `json:"created_at"`
			DisplayName string   `json:"display_name"`
			Likes       int      `json:"like_count"`
			LikedByMe   bool     `json:"liked_by_me"`
			Bookmarked  bool     `json:"bookmarked_by_me"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.Title, &x.Body, &x.Images, &x.Tags, &x.Year, &x.Identity, &x.Created, &x.DisplayName, &x.Likes, &x.LikedByMe, &x.Bookmarked) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取课程经验失败"})
				return
			}
			if x.Year == nil {
				x.YearLabel = "年份未注明"
			} else {
				x.YearLabel = strconv.Itoa(*x.Year)
			}
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取课程经验失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	}))))
	mux.Handle("POST /api/v1/courses/{courseID}/posts", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		courseID, ok := pathID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "课程编号无效"})
			return
		}
		var in postInput
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&in) != nil || len([]rune(strings.TrimSpace(in.Title))) < 1 || len([]rune(in.Title)) > 120 || len([]rune(strings.TrimSpace(in.Body))) < 1 || len([]rune(in.Body)) > 2000 {
			writeJSON(w, 400, map[string]string{"error": "标题或正文不符合长度要求"})
			return
		}
		if len(in.ImageURLs) > 9 {
			writeJSON(w, 400, map[string]string{"error": "图片最多 9 张"})
			return
		}
		for _, raw := range in.ImageURLs {
			u, e := url.Parse(raw)
			if e != nil || u.IsAbs() || !strings.HasPrefix(u.Path, "/uploads/") || !uploadNamePattern.MatchString(strings.TrimPrefix(u.Path, "/uploads/")) || u.RawQuery != "" {
				writeJSON(w, 400, map[string]string{"error": "图片必须通过 CourseRep 上传接口添加"})
				return
			}
		}
		if in.DisplayIdentity == "" {
			in.DisplayIdentity = "named"
		}
		if in.DisplayIdentity != "named" && in.DisplayIdentity != "anonymous" {
			writeJSON(w, 400, map[string]string{"error": "署名选项无效"})
			return
		}
		if in.AttendanceYear != nil && (*in.AttendanceYear < 1990 || *in.AttendanceYear > currentYear()) {
			writeJSON(w, 400, map[string]string{"error": "上课年份不在有效范围内"})
			return
		}
		if len(in.Tags) > 10 {
			writeJSON(w, 400, map[string]string{"error": "话题最多 10 个"})
			return
		}
		seen := map[string]bool{}
		for _, t := range in.Tags {
			if t == "" || seen[t] {
				writeJSON(w, 400, map[string]string{"error": "话题标签无效或重复"})
				return
			}
			seen[t] = true
		}
		if len(in.Tags) > 0 {
			var n int
			if pool.QueryRow(r.Context(), `SELECT count(*) FROM topic_tags WHERE active AND name=ANY($1)`, in.Tags).Scan(&n) != nil || n != len(in.Tags) {
				writeJSON(w, 400, map[string]string{"error": "包含未启用的话题标签"})
				return
			}
		}
		var exists bool
		if pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM courses WHERE id=$1)`, courseID).Scan(&exists) != nil || !exists {
			writeJSON(w, 404, map[string]string{"error": "课程不存在"})
			return
		}
		u := userFromContext(r.Context())
		var id int64
		e := pool.QueryRow(r.Context(), `INSERT INTO course_posts(course_id,author_id,title,body,image_urls,topic_tags,attendance_year,display_identity,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'submitted') RETURNING id`, courseID, u.ID, strings.TrimSpace(in.Title), strings.TrimSpace(in.Body), in.ImageURLs, in.Tags, in.AttendanceYear, in.DisplayIdentity).Scan(&id)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "经验帖无法提交"})
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "status": "submitted", "message": "经验帖已提交，审核通过后公开"})
	}))))
	mux.Handle("PUT /api/v1/posts/{postID}", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := postID(r)
		var in postInput
		if !ok || json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&in) != nil || len([]rune(strings.TrimSpace(in.Title))) < 1 || len([]rune(in.Title)) > 120 || len([]rune(strings.TrimSpace(in.Body))) < 1 || len([]rune(in.Body)) > 2000 {
			writeJSON(w, 400, map[string]string{"error": "标题或正文不符合长度要求"})
			return
		}
		if len(in.ImageURLs) > 9 || in.AttendanceYear != nil && (*in.AttendanceYear < 1990 || *in.AttendanceYear > currentYear()) {
			writeJSON(w, 400, map[string]string{"error": "图片数量或上课年份无效"})
			return
		}
		for _, raw := range in.ImageURLs {
			u, e := url.Parse(raw)
			if e != nil || u.IsAbs() || !uploadNamePattern.MatchString(strings.TrimPrefix(u.Path, "/uploads/")) || !strings.HasPrefix(u.Path, "/uploads/") || u.RawQuery != "" {
				writeJSON(w, 400, map[string]string{"error": "图片地址无效"})
				return
			}
		}
		if len(in.Tags) > 10 || !validTags(in.Tags) {
			writeJSON(w, 400, map[string]string{"error": "话题标签无效"})
			return
		}
		if len(in.Tags) > 0 {
			var n int
			if pool.QueryRow(r.Context(), `SELECT count(*) FROM topic_tags WHERE active AND name=ANY($1)`, in.Tags).Scan(&n) != nil || n != len(in.Tags) {
				writeJSON(w, 400, map[string]string{"error": "包含未启用的话题标签"})
				return
			}
		}
		if in.DisplayIdentity == "" {
			in.DisplayIdentity = "named"
		}
		if in.DisplayIdentity != "named" && in.DisplayIdentity != "anonymous" {
			writeJSON(w, 400, map[string]string{"error": "署名选项无效"})
			return
		}
		u := userFromContext(r.Context())
		tag, e := pool.Exec(r.Context(), `UPDATE course_posts SET title=$1,body=$2,image_urls=$3,topic_tags=$4,attendance_year=$5,display_identity=$6,status='submitted',updated_at=now() WHERE id=$7 AND author_id=$8 AND status IN ('published','rejected')`, strings.TrimSpace(in.Title), strings.TrimSpace(in.Body), in.ImageURLs, in.Tags, in.AttendanceYear, in.DisplayIdentity, id, u.ID)
		if e != nil {
			writeJSON(w, 409, map[string]string{"error": "经验帖无法编辑"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, 404, map[string]string{"error": "经验帖不存在或当前不可编辑"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "submitted", "message": "修改已提交审核"})
	}))))
	mux.Handle("GET /api/v1/posts/{postID}", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := postID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "帖子编号无效"})
			return
		}
		var x struct {
			ID       int64    `json:"id"`
			CourseID int64    `json:"course_id"`
			Title    string   `json:"title"`
			Body     string   `json:"body"`
			Images   []string `json:"image_urls"`
			Tags     []string `json:"topic_tags"`
			Year     *int     `json:"attendance_year"`
			Identity string   `json:"display_identity"`
			Created  any      `json:"created_at"`
			Name     string   `json:"display_name"`
		}
		e := pool.QueryRow(r.Context(), `SELECT p.id,p.course_id,p.title,p.body,p.image_urls,p.topic_tags,p.attendance_year,p.display_identity,p.created_at,CASE WHEN p.display_identity='anonymous' THEN '同校已验证' ELSE u.nickname END FROM course_posts p JOIN users u ON u.id=p.author_id WHERE p.id=$1 AND p.status='published'`, id).Scan(&x.ID, &x.CourseID, &x.Title, &x.Body, &x.Images, &x.Tags, &x.Year, &x.Identity, &x.Created, &x.Name)
		if e != nil {
			writeJSON(w, 404, map[string]string{"error": "帖子不存在"})
			return
		}
		label := "年份未注明"
		if x.Year != nil {
			label = strconv.Itoa(*x.Year)
		}
		writeJSON(w, 200, map[string]any{"post": map[string]any{"id": x.ID, "course_id": x.CourseID, "title": x.Title, "body": x.Body, "image_urls": x.Images, "topic_tags": x.Tags, "attendance_year": x.Year, "attendance_year_label": label, "display_identity": x.Identity, "display_name": x.Name, "created_at": x.Created}})
	}))))
	mux.Handle("POST /api/v1/admin/posts/{postID}/decision", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("postID"), 10, 64)
		if err != nil || id < 1 {
			writeJSON(w, 400, map[string]string{"error": "帖子编号无效"})
			return
		}
		var req struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || (req.Decision != "publish" && req.Decision != "reject" && req.Decision != "remove") || len([]rune(strings.TrimSpace(req.Reason))) < 2 || len([]rune(req.Reason)) > 500 {
			writeJSON(w, 400, map[string]string{"error": "审核决定或处理原因无效"})
			return
		}
		status := map[string]string{"publish": "published", "reject": "rejected", "remove": "removed"}[req.Decision]
		u := userFromContext(r.Context())
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法处理审核"})
			return
		}
		defer tx.Rollback(r.Context())
		var owner int64
		var published any
		err = tx.QueryRow(r.Context(), `UPDATE course_posts SET status=$1,updated_at=now(),published_at=CASE WHEN $1='published' THEN now() ELSE published_at END WHERE id=$2 AND (($3='remove' AND status='published') OR ($3<>'remove' AND status IN ('submitted','under_review'))) RETURNING author_id,published_at`, status, id, req.Decision).Scan(&owner, &published)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "待处理帖子不存在"})
			return
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO moderation_actions(moderator_id,post_id,action,reason) VALUES($1,$2,$3,$4)`, u.ID, id, req.Decision, strings.TrimSpace(req.Reason))
		if err == nil {
			_, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'post_moderation',jsonb_build_object('post_id',$2,'status',$3))`, owner, id, status)
		}
		if err != nil || tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "无法完成审核"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": status})
	}))))
	registerInteractionRoutes(mux, pool)
}

func registerInteractionRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/notifications", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
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
		rows, e := pool.Query(r.Context(), `SELECT id,kind,payload,read_at,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, u.ID, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取通知失败"})
			return
		}
		defer rows.Close()
		type notification struct {
			ID      int64           `json:"id"`
			Kind    string          `json:"kind"`
			Payload json.RawMessage `json:"payload"`
			ReadAt  any             `json:"read_at"`
			Created any             `json:"created_at"`
		}
		items := make([]notification, 0)
		for rows.Next() {
			var x notification
			if rows.Scan(&x.ID, &x.Kind, &x.Payload, &x.ReadAt, &x.Created) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取通知失败"})
				return
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	})))
	mux.Handle("POST /api/v1/notifications/{notificationID}/read", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, e := strconv.ParseInt(r.PathValue("notificationID"), 10, 64)
		if e != nil || id < 1 {
			writeJSON(w, 400, map[string]string{"error": "通知编号无效"})
			return
		}
		u := userFromContext(r.Context())
		tag, e := pool.Exec(r.Context(), `UPDATE notifications SET read_at=coalesce(read_at,now()) WHERE id=$1 AND user_id=$2`, id, u.ID)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "无法更新通知"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, 404, map[string]string{"error": "通知不存在"})
			return
		}
		w.WriteHeader(204)
	})))
	mux.Handle("GET /api/v1/posts/{postID}/comments", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := postID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "帖子编号无效"})
			return
		}
		rows, e := pool.Query(r.Context(), `SELECT c.id,c.body,c.created_at,CASE WHEN p.display_identity='anonymous' AND c.author_id=p.author_id THEN '同校已验证' ELSE u.nickname END,(c.author_id=p.author_id) FROM post_comments c JOIN users u ON u.id=c.author_id JOIN course_posts p ON p.id=c.post_id WHERE c.post_id=$1 AND p.status='published' ORDER BY c.created_at,c.id`, id)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取评论失败"})
			return
		}
		defer rows.Close()
		type comment struct {
			ID           int64  `json:"id"`
			Body         string `json:"body"`
			Created      any    `json:"created_at"`
			Author       string `json:"author"`
			IsPostAuthor bool   `json:"is_post_author"`
		}
		items := make([]comment, 0)
		for rows.Next() {
			var x comment
			if rows.Scan(&x.ID, &x.Body, &x.Created, &x.Author, &x.IsPostAuthor) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取评论失败"})
				return
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))))
	mux.Handle("POST /api/v1/posts/{postID}/comments", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := postID(r)
		var in struct {
			Body string `json:"body"`
		}
		if !ok || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil || len([]rune(strings.TrimSpace(in.Body))) < 1 || len([]rune(in.Body)) > 1000 {
			writeJSON(w, 400, map[string]string{"error": "评论内容无效"})
			return
		}
		u := userFromContext(r.Context())
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法发表评论"})
			return
		}
		defer tx.Rollback(r.Context())
		var owner int64
		err = tx.QueryRow(r.Context(), `INSERT INTO post_comments(post_id,author_id,body) SELECT id,$2,$3 FROM course_posts WHERE id=$1 AND status='published' RETURNING (SELECT author_id FROM course_posts WHERE id=$1)`, id, u.ID, strings.TrimSpace(in.Body)).Scan(&owner)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "帖子不存在"})
			return
		}
		if owner != u.ID {
			_, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'post_comment',jsonb_build_object('post_id',$2))`, owner, id)
		}
		if err != nil || tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "无法发表评论"})
			return
		}
		writeJSON(w, 201, map[string]string{"status": "created"})
	}))))
	for _, verb := range []string{"like", "bookmark"} {
		v := verb
		mux.Handle("PUT /api/v1/posts/{postID}/"+v, authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := postID(r)
			if !ok {
				writeJSON(w, 400, map[string]string{"error": "帖子编号无效"})
				return
			}
			u := userFromContext(r.Context())
			table := "post_likes"
			if v == "bookmark" {
				table = "post_bookmarks"
			}
			var owner int64
			err := pool.QueryRow(r.Context(), `SELECT author_id FROM course_posts WHERE id=$1 AND status='published'`, id).Scan(&owner)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": "帖子不存在"})
				return
			}
			tx, err := pool.Begin(r.Context())
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "操作失败"})
				return
			}
			defer tx.Rollback(r.Context())
			inserted, err := tx.Exec(r.Context(), `INSERT INTO `+table+`(post_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, u.ID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "操作失败"})
				return
			}
			if v == "like" && owner != u.ID && inserted.RowsAffected() == 1 {
				if _, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'post_like',jsonb_build_object('post_id',$2))`, owner, id); err != nil {
					writeJSON(w, 500, map[string]string{"error": "操作失败"})
					return
				}
			}
			if err = tx.Commit(r.Context()); err != nil {
				writeJSON(w, 500, map[string]string{"error": "操作失败"})
				return
			}
			writeJSON(w, 200, map[string]string{"status": "saved"})
		}))))
	}
	for _, verb := range []string{"like", "bookmark"} {
		v := verb
		route := "/api/v1/posts/{postID}/" + v
		if v == "like" {
			route += ""
		}
		mux.Handle("DELETE "+route, authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := postID(r)
			if !ok {
				writeJSON(w, 400, map[string]string{"error": "帖子编号无效"})
				return
			}
			table := "post_likes"
			if v == "bookmark" {
				table = "post_bookmarks"
			}
			u := userFromContext(r.Context())
			_, e := pool.Exec(r.Context(), `DELETE FROM `+table+` WHERE post_id=$1 AND user_id=$2`, id, u.ID)
			if e != nil {
				writeJSON(w, 500, map[string]string{"error": "操作失败"})
				return
			}
			w.WriteHeader(204)
		}))))
	}
}

func postID(r *http.Request) (int64, bool) {
	n, e := strconv.ParseInt(r.PathValue("postID"), 10, 64)
	return n, e == nil && n > 0
}
func currentYear() int { return time.Now().Year() }

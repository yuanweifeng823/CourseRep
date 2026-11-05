package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type contractCourse struct {
	ID          int64   `json:"id"`
	CourseCode  string  `json:"course_code"`
	CourseName  string  `json:"course_name"`
	Professor   string  `json:"professor"`
	Semester    string  `json:"semester"`
	Department  string  `json:"department"`
	Credits     float64 `json:"credits"`
	Rating      float64 `json:"rating"`
	ReviewCount int     `json:"review_count"`
	PostCount   int     `json:"post_count"`
}
type contractListOutput struct {
	Body struct {
		Items    []contractCourse `json:"items"`
		Page     int              `json:"page"`
		PageSize int              `json:"page_size"`
	}
}
type contractMessageOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}
type contractVerificationOutput struct {
	Body struct {
		UserID  int64  `json:"user_id"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
}
type contractIDStatusMessageOutput struct {
	Body struct {
		ID      int64  `json:"id"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
}
type contractReportCreatedOutput struct {
	Body struct {
		ReportID int64  `json:"report_id"`
		Status   string `json:"status"`
	}
}
type contractStatusMessageOutput struct {
	Body struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}
}
type contractStatusOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}
type contractReportDecisionOutput struct {
	Body struct {
		Status string `json:"status"`
		Action string `json:"action"`
	}
}
type contractCurrentUserOutput struct {
	Body struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Role   string `json:"role"`
	}
}
type contractTopicTagOutput struct {
	Body struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}
}
type contractImageUploadOutput struct {
	Body struct {
		ImageURL        string `json:"image_url"`
		PrivacyReminder string `json:"privacy_reminder"`
	}
}
type contractImportPreviewOutput struct {
	Body struct {
		BatchID   string                   `json:"batch_id"`
		Rows      int                      `json:"rows"`
		ValidRows int                      `json:"valid_rows"`
		Errors    []contractImportIssue    `json:"errors"`
		Preview   []contractImportedCourse `json:"preview"`
	}
}
type contractImportIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}
type contractImportedCourse struct {
	CourseCode string  `json:"course_code"`
	CourseName string  `json:"course_name"`
	Professor  string  `json:"professor"`
	Semester   string  `json:"semester"`
	Department string  `json:"department"`
	Credits    float64 `json:"credits"`
}
type contractImportConfirmOutput struct {
	Body struct {
		Imported int    `json:"imported"`
		Status   string `json:"status"`
	}
}
type contractSessionOutput struct {
	Body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		User         struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
			Role   string `json:"role"`
		} `json:"user"`
	}
}
type contractCourseDetailOutput struct {
	Body struct {
		Course struct {
			ID         int64   `json:"id"`
			CourseCode string  `json:"course_code"`
			CourseName string  `json:"course_name"`
			Professor  string  `json:"professor"`
			Semester   string  `json:"semester"`
			Department string  `json:"department"`
			Credits    float64 `json:"credits"`
		} `json:"course"`
		ReviewCount int `json:"review_count"`
		Ratings     struct {
			Grading    float64 `json:"grading"`
			Workload   float64 `json:"workload"`
			Assessment float64 `json:"assessment"`
			Usefulness float64 `json:"usefulness"`
		} `json:"ratings"`
	}
}
type contractReviewItem struct {
	ID          int64    `json:"id"`
	Semester    string   `json:"semester"`
	Grading     int      `json:"grading"`
	Workload    int      `json:"workload"`
	Assessment  int      `json:"assessment"`
	Usefulness  int      `json:"usefulness"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
	CreatedAt   string   `json:"created_at"`
	DisplayName string   `json:"display_name"`
}
type contractReviewListOutput struct {
	Body struct {
		Items    []contractReviewItem `json:"items"`
		Page     int                  `json:"page"`
		PageSize int                  `json:"page_size"`
	}
}
type contractPostItem struct {
	ID                  int64    `json:"id"`
	Title               string   `json:"title"`
	Body                string   `json:"body"`
	ImageURLs           []string `json:"image_urls"`
	TopicTags           []string `json:"topic_tags"`
	AttendanceYear      *int     `json:"attendance_year"`
	AttendanceYearLabel string   `json:"attendance_year_label"`
	DisplayIdentity     string   `json:"display_identity"`
	DisplayName         string   `json:"display_name"`
	CreatedAt           string   `json:"created_at"`
	LikeCount           int      `json:"like_count"`
	LikedByMe           bool     `json:"liked_by_me"`
	BookmarkedByMe      bool     `json:"bookmarked_by_me"`
}
type contractPostListOutput struct {
	Body struct {
		Items    []contractPostItem `json:"items"`
		Page     int                `json:"page"`
		PageSize int                `json:"page_size"`
	}
}
type contractPostDetailOutput struct {
	Body struct {
		Post struct {
			ID                  int64    `json:"id"`
			CourseID            int64    `json:"course_id"`
			Title               string   `json:"title"`
			Body                string   `json:"body"`
			ImageURLs           []string `json:"image_urls"`
			TopicTags           []string `json:"topic_tags"`
			AttendanceYear      *int     `json:"attendance_year"`
			AttendanceYearLabel string   `json:"attendance_year_label"`
			DisplayIdentity     string   `json:"display_identity"`
			DisplayName         string   `json:"display_name"`
			CreatedAt           string   `json:"created_at"`
		} `json:"post"`
	}
}
type contractNotificationItem struct {
	ID        int64          `json:"id"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
	ReadAt    *string        `json:"read_at"`
	CreatedAt string         `json:"created_at"`
}
type contractNotificationListOutput struct {
	Body struct {
		Items    []contractNotificationItem `json:"items"`
		Page     int                        `json:"page"`
		PageSize int                        `json:"page_size"`
	}
}
type contractProfileOutput struct {
	Body struct {
		ID         int64  `json:"id"`
		Nickname   string `json:"nickname"`
		Status     string `json:"status"`
		Role       string `json:"role"`
		Department string `json:"department"`
	}
}
type profileUpdateInput struct {
	Body struct {
		Department string `json:"department" minLength:"1" maxLength:"100"`
	}
}
type contractHomePost struct {
	ID          int64          `json:"id"`
	CourseID    int64          `json:"course_id"`
	Title       string         `json:"title"`
	Body        string         `json:"body"`
	Year        *int           `json:"attendance_year"`
	YearLabel   string         `json:"attendance_year_label"`
	Identity    string         `json:"display_identity"`
	DisplayName string         `json:"display_name"`
	LikeCount   int            `json:"like_count"`
	CommentCnt  int            `json:"comment_count"`
	ViewCount   int            `json:"view_count"`
	Created     any            `json:"created_at"`
	Course      contractCourse `json:"course"`
}
type contractHomeOutput struct {
	Body struct {
		Department      string             `json:"department"`
		NeedsDepartment bool               `json:"needs_department"`
		TopCourses      []contractCourse   `json:"top_courses"`
		TrendingPosts   []contractHomePost `json:"trending_posts"`
	}
}
type contractFeedPost struct {
	ID          int64          `json:"id"`
	CourseID    int64          `json:"course_id"`
	Title       string         `json:"title"`
	Body        string         `json:"body"`
	ImageURLs   []string       `json:"image_urls"`
	TopicTags   []string       `json:"topic_tags"`
	Year        *int           `json:"attendance_year"`
	YearLabel   string         `json:"attendance_year_label"`
	Identity    string         `json:"display_identity"`
	DisplayName string         `json:"display_name"`
	LikeCount   int            `json:"like_count"`
	CommentCnt  int            `json:"comment_count"`
	ViewCount   int            `json:"view_count"`
	LikedByMe   bool           `json:"liked_by_me"`
	Bookmarked  bool           `json:"bookmarked_by_me"`
	Created     any            `json:"created_at"`
	Course      contractCourse `json:"course"`
}
type contractFeedOutput struct {
	Body struct {
		Items    []contractFeedPost `json:"items"`
		Page     int                `json:"page"`
		PageSize int                `json:"page_size"`
		HasMore  bool               `json:"has_more"`
	}
}
type contractMetricsOutput struct {
	Body struct {
		ValidReviews             int     `json:"valid_reviews"`
		ValidPosts               int     `json:"valid_posts"`
		ActiveCourseViewers30d   int     `json:"active_course_viewers_30d"`
		Reports30d               int     `json:"reports_30d"`
		ReportRatePercent        float64 `json:"report_rate_percent"`
		OpenTeacherComplaints30d int     `json:"open_teacher_complaints_30d"`
		ReportRateDefinition     string  `json:"report_rate_definition"`
		ActiveUserDefinition     string  `json:"active_user_definition"`
	}
}
type contractOwnReviewItem struct {
	ID         int64    `json:"id"`
	CourseID   int64    `json:"course_id"`
	Status     string   `json:"status"`
	Semester   string   `json:"semester"`
	Grading    int      `json:"grading"`
	Workload   int      `json:"workload"`
	Assessment int      `json:"assessment"`
	Usefulness int      `json:"usefulness"`
	Tags       []string `json:"tags"`
	Body       string   `json:"body"`
	CreatedAt  string   `json:"created_at"`
	CourseCode string   `json:"course_code"`
	CourseName string   `json:"course_name"`
}
type contractOwnReviewListOutput struct {
	Body struct {
		Items []contractOwnReviewItem `json:"items"`
	}
}
type contractOwnPostItem struct {
	ID                  int64    `json:"id"`
	CourseID            int64    `json:"course_id"`
	CourseCode          string   `json:"course_code"`
	CourseName          string   `json:"course_name"`
	Title               string   `json:"title"`
	Body                string   `json:"body"`
	ImageURLs           []string `json:"image_urls"`
	TopicTags           []string `json:"topic_tags"`
	AttendanceYear      *int     `json:"attendance_year"`
	AttendanceYearLabel string   `json:"attendance_year_label"`
	Status              string   `json:"status"`
	DisplayIdentity     string   `json:"display_identity"`
	CreatedAt           string   `json:"created_at"`
}
type contractOwnPostListOutput struct {
	Body struct {
		Items []contractOwnPostItem `json:"items"`
	}
}
type contractBookmarkItem struct {
	ID                  int64  `json:"id"`
	CourseID            int64  `json:"course_id"`
	Title               string `json:"title"`
	Body                string `json:"body"`
	AttendanceYear      *int   `json:"attendance_year"`
	AttendanceYearLabel string `json:"attendance_year_label"`
	DisplayIdentity     string `json:"display_identity"`
	DisplayName         string `json:"display_name"`
	CreatedAt           string `json:"created_at"`
	CourseCode          string `json:"course_code"`
	CourseName          string `json:"course_name"`
}
type contractBookmarkListOutput struct {
	Body struct {
		Items []contractBookmarkItem `json:"items"`
	}
}
type contractCompletedCourseItem struct {
	ID         int64   `json:"id"`
	CourseCode string  `json:"course_code"`
	CourseName string  `json:"course_name"`
	Professor  string  `json:"professor"`
	Semester   string  `json:"semester"`
	Department string  `json:"department"`
	Credits    float64 `json:"credits"`
}
type contractCompletedCourseListOutput struct {
	Body struct {
		Items []contractCompletedCourseItem `json:"items"`
	}
}
type contractCommentItem struct {
	ID           int64  `json:"id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
	Author       string `json:"author"`
	IsPostAuthor bool   `json:"is_post_author"`
}
type contractCommentListOutput struct {
	Body struct {
		Items []contractCommentItem `json:"items"`
	}
}
type contractReportItem struct {
	ID         int64  `json:"id"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	Body       string `json:"body"`
	Author     string `json:"author"`
}
type contractReportListOutput struct {
	Body struct {
		Items    []contractReportItem `json:"items"`
		Page     int                  `json:"page"`
		PageSize int                  `json:"page_size"`
	}
}
type contractCorrectionItem struct {
	ID             int64          `json:"id"`
	CourseID       int64          `json:"course_id"`
	CourseCode     string         `json:"course_code"`
	CourseName     string         `json:"course_name"`
	ProposedFields map[string]any `json:"proposed_fields"`
	CreatedAt      string         `json:"created_at"`
}
type contractCorrectionListOutput struct {
	Body struct {
		Items    []contractCorrectionItem `json:"items"`
		Page     int                      `json:"page"`
		PageSize int                      `json:"page_size"`
	}
}
type contractPendingReviewItem struct {
	ID         int64    `json:"id"`
	CourseID   int64    `json:"course_id"`
	CourseCode string   `json:"course_code"`
	CourseName string   `json:"course_name"`
	Semester   string   `json:"semester"`
	Grading    int      `json:"grading"`
	Workload   int      `json:"workload"`
	Assessment int      `json:"assessment"`
	Usefulness int      `json:"usefulness"`
	Tags       []string `json:"tags"`
	Body       string   `json:"body"`
	Status     string   `json:"status"`
	CreatedAt  string   `json:"created_at"`
}
type contractPendingReviewListOutput struct {
	Body struct {
		Items    []contractPendingReviewItem `json:"items"`
		Page     int                         `json:"page"`
		PageSize int                         `json:"page_size"`
	}
}
type contractPendingPostItem struct {
	ID                  int64    `json:"id"`
	CourseID            int64    `json:"course_id"`
	CourseCode          string   `json:"course_code"`
	CourseName          string   `json:"course_name"`
	Title               string   `json:"title"`
	Body                string   `json:"body"`
	ImageURLs           []string `json:"image_urls"`
	TopicTags           []string `json:"topic_tags"`
	AttendanceYear      *int     `json:"attendance_year"`
	AttendanceYearLabel string   `json:"attendance_year_label"`
	DisplayIdentity     string   `json:"display_identity"`
	DisplayName         string   `json:"display_name"`
	Status              string   `json:"status"`
	CreatedAt           string   `json:"created_at"`
}
type contractPendingPostListOutput struct {
	Body struct {
		Items    []contractPendingPostItem `json:"items"`
		Page     int                       `json:"page"`
		PageSize int                       `json:"page_size"`
	}
}
type contractTopicTagItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type contractTopicTagListOutput struct {
	Body struct {
		Items []contractTopicTagItem `json:"items"`
	}
}
type contractAdminTopicTagItem struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
type contractAdminTopicTagListOutput struct {
	Body struct {
		Items []contractAdminTopicTagItem `json:"items"`
	}
}
type contractCommunitySettingsOutput struct {
	Body struct {
		ContentReviewSLAHours *int   `json:"content_review_sla_hours"`
		CommunityGuidelines   string `json:"community_guidelines"`
	}
}
type contractCourseFacetsOutput struct {
	Body struct {
		Departments []string `json:"departments"`
		Semesters   []string `json:"semesters"`
	}
}
type coursePath struct {
	CourseID int64 `path:"courseID" minimum:"1"`
}
type postPath struct {
	PostID int64 `path:"postID" minimum:"1"`
}
type reviewPath struct {
	ReviewID int64 `path:"reviewID" minimum:"1"`
}
type batchPath struct {
	BatchID string `path:"batchID" minLength:"1" maxLength:"64"`
}
type tagPath struct {
	TagID int64 `path:"tagID" minimum:"1"`
}
type createTagInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"40"`
	}
}
type reviewDecisionInput struct {
	ReviewID int64 `path:"reviewID"`
	Body     struct {
		Decision string `json:"decision" enum:"publish,reject,remove"`
		Reason   string `json:"reason" minLength:"2" maxLength:"500"`
	}
}
type postDecisionInput struct {
	PostID int64 `path:"postID"`
	Body   struct {
		Decision string `json:"decision" enum:"publish,reject,remove"`
		Reason   string `json:"reason" minLength:"2" maxLength:"500"`
	}
}
type reviewUpdateInput struct {
	CourseID int64 `path:"courseID"`
	ReviewID int64 `path:"reviewID"`
	Body     struct {
		Grading    int      `json:"grading" minimum:"1" maximum:"5"`
		Workload   int      `json:"workload" minimum:"1" maximum:"5"`
		Assessment int      `json:"assessment" minimum:"1" maximum:"5"`
		Usefulness int      `json:"usefulness" minimum:"1" maximum:"5"`
		Tags       []string `json:"tags" maxItems:"10" required:"false"`
		Text       string   `json:"body" maxLength:"2000" required:"false"`
	}
}
type postUpdateInput struct {
	PostID int64 `path:"postID"`
	Body   struct {
		Title           string   `json:"title" minLength:"1" maxLength:"120"`
		Text            string   `json:"body" minLength:"1" maxLength:"2000"`
		ImageURLs       []string `json:"image_urls" maxItems:"9" required:"false"`
		TopicTags       []string `json:"topic_tags" maxItems:"10" required:"false"`
		AttendanceYear  *int     `json:"attendance_year" minimum:"1990" maximum:"2100" required:"false" doc:"实际修读年份，不得晚于当前年份；留空表示年份未注明"`
		DisplayIdentity string   `json:"display_identity" enum:"named,anonymous" required:"false" doc:"留空按署名展示"`
	}
}
type authRegisterInput struct {
	Body struct {
		Email    string `json:"email" format:"email" maxLength:"254"`
		Password string `json:"password" minLength:"10" maxLength:"72"`
		Nickname string `json:"nickname" minLength:"1" maxLength:"40"`
	}
}
type authLoginInput struct {
	Body struct {
		Email    string `json:"email" format:"email" maxLength:"254"`
		Password string `json:"password" minLength:"1" maxLength:"72"`
	}
}
type reviewCreateInput struct {
	CourseID int64 `path:"courseID"`
	Body     struct {
		Grading    int      `json:"grading" minimum:"1" maximum:"5"`
		Workload   int      `json:"workload" minimum:"1" maximum:"5"`
		Assessment int      `json:"assessment" minimum:"1" maximum:"5"`
		Usefulness int      `json:"usefulness" minimum:"1" maximum:"5"`
		Tags       []string `json:"tags" maxItems:"10" required:"false"`
		Text       string   `json:"body" maxLength:"2000" required:"false"`
	}
}
type postCreateInput struct {
	CourseID int64 `path:"courseID"`
	Body     struct {
		Title           string   `json:"title" minLength:"1" maxLength:"120"`
		Text            string   `json:"body" minLength:"1" maxLength:"2000"`
		ImageURLs       []string `json:"image_urls" maxItems:"9" required:"false"`
		TopicTags       []string `json:"topic_tags" maxItems:"10" required:"false"`
		AttendanceYear  *int     `json:"attendance_year" minimum:"1990" maximum:"2100" required:"false" doc:"实际修读年份，不得晚于当前年份；留空表示年份未注明"`
		DisplayIdentity string   `json:"display_identity" enum:"named,anonymous" required:"false" doc:"留空按署名展示"`
	}
}
type reportCreateInput struct {
	Body struct {
		ReviewID *int64 `json:"review_id" minimum:"1" required:"false"`
		PostID   *int64 `json:"post_id" minimum:"1" required:"false"`
		Category string `json:"category" enum:"spam,academic_integrity,copyright,personal_attack,teacher_claim,other"`
		Detail   string `json:"detail" maxLength:"900"`
	}
}
type refreshInput struct {
	Body struct {
		RefreshToken string `json:"refresh_token" minLength:"1" maxLength:"128"`
	}
}
type commentCreateInput struct {
	PostID int64 `path:"postID"`
	Body   struct {
		Text string `json:"body" minLength:"1" maxLength:"1000"`
	}
}
type notificationReadInput struct {
	NotificationID int64 `path:"notificationID"`
}
type correctionCreateInput struct {
	CourseID int64 `path:"courseID"`
	Body     struct {
		CourseName *string  `json:"course_name" maxLength:"200" required:"false"`
		Professor  *string  `json:"professor" maxLength:"120" required:"false"`
		Semester   *string  `json:"semester" maxLength:"40" required:"false"`
		Department *string  `json:"department" maxLength:"120" required:"false"`
		Credits    *float64 `json:"credits" minimum:"0" maximum:"99" required:"false"`
	}
}
type correctionDecisionInput struct {
	CorrectionID int64 `path:"correctionID"`
	Body         struct {
		Decision string `json:"decision" enum:"accept,reject"`
		Reason   string `json:"reason" minLength:"2" maxLength:"500"`
	}
}
type reportDecisionInput struct {
	ReportID int64 `path:"reportID"`
	Body     struct {
		Decision string `json:"decision" enum:"resolve,dismiss"`
		Action   string `json:"action" enum:"none,remove_content,disable_author"`
		Reason   string `json:"reason" minLength:"2" maxLength:"500"`
	}
}
type tagPatchInput struct {
	TagID int64 `path:"tagID"`
	Body  struct {
		Name   *string `json:"name" minLength:"1" maxLength:"40" required:"false"`
		Active *bool   `json:"active" required:"false"`
	}
}
type communitySettingsInput struct {
	Body struct {
		ContentReviewSLAHours *int   `json:"content_review_sla_hours" minimum:"1" maximum:"720" required:"false"`
		CommunityGuidelines   string `json:"community_guidelines" maxLength:"10000"`
	}
}
type courseListInput struct {
	Q          string `query:"q" maxLength:"100"`
	Page       int    `query:"page" minimum:"1" maximum:"1000000"`
	PageSize   int    `query:"page_size" minimum:"1" maximum:"100"`
	Department string `query:"department" maxLength:"100"`
	Semester   string `query:"semester" maxLength:"100"`
	Sort       string `query:"sort" enum:"course,rating,reviews,posts"`
}
type pageQueryInput struct {
	Page     int `query:"page" minimum:"1" maximum:"1000000"`
	PageSize int `query:"page_size" minimum:"1" maximum:"100"`
}
type reviewQueryInput struct {
	CourseID int64  `path:"courseID" minimum:"1"`
	Page     int    `query:"page" minimum:"1" maximum:"1000000"`
	PageSize int    `query:"page_size" minimum:"1" maximum:"100"`
	Semester string `query:"semester" maxLength:"40"`
	Tag      string `query:"tag" maxLength:"60"`
}
type postQueryInput struct {
	CourseID int64  `path:"courseID" minimum:"1"`
	Page     int    `query:"page" minimum:"1" maximum:"1000000"`
	PageSize int    `query:"page_size" minimum:"1" maximum:"100"`
	Year     string `query:"year" maxLength:"10" pattern:"^([0-9]{4}|unspecified)$"`
	Tag      string `query:"tag" maxLength:"60"`
	Sort     string `query:"sort" enum:"newest,popular"`
}

func newContractAPI(dispatch ...http.Handler) (http.Handler, *huma.OpenAPI) {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("CourseRep API", "1.0.0")
	config.OpenAPIPath = "/openapi"
	config.DocsPath = "/docs"
	config.SchemasPath = "/schemas"
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{"bearer": &huma.SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "HMAC access token"}}
	api := humago.New(mux, config)
	if len(dispatch) > 0 && dispatch[0] != nil {
		legacy := dispatch[0]
		api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
			r, w := humago.Unwrap(ctx)
			if strings.HasPrefix(r.URL.Path, "/api/v1/") || strings.HasPrefix(r.URL.Path, "/uploads/") {
				legacy.ServeHTTP(w, r)
				return
			}
			next(ctx)
		})
	}
	public := huma.Operation{Tags: []string{"认证"}}
	secured := []map[string][]string{{"bearer": {}}}
	huma.Register(api, mergeOperation(public, huma.Operation{OperationID: "register", Method: http.MethodPost, Path: "/api/v1/auth/register", Summary: "注册并发送学生邮箱验证码", DefaultStatus: http.StatusAccepted}), func(context.Context, *authRegisterInput) (*contractMessageOutput, error) {
		return &contractMessageOutput{}, nil
	})
	huma.Register(api, mergeOperation(public, huma.Operation{OperationID: "verifyEmail", Method: http.MethodPost, Path: "/api/v1/auth/verify", Summary: "验证学生邮箱验证码", DefaultStatus: http.StatusCreated}), func(context.Context, *struct {
		Body struct {
			Email string `json:"email"`
			Code  string `json:"code"`
		}
	}) (*contractVerificationOutput, error) {
		return &contractVerificationOutput{}, nil
	})
	huma.Register(api, mergeOperation(public, huma.Operation{OperationID: "login", Method: http.MethodPost, Path: "/api/v1/auth/login", Summary: "登录并创建会话"}), func(context.Context, *authLoginInput) (*contractSessionOutput, error) {
		return &contractSessionOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listCourses", Method: http.MethodGet, Path: "/api/v1/courses", Summary: "搜索课程目录", Tags: []string{"课程"}, Security: secured}, func(context.Context, *courseListInput) (*contractListOutput, error) {
		return &contractListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listCourseFacets", Method: http.MethodGet, Path: "/api/v1/course-facets", Summary: "获取专业和学期筛选项", Tags: []string{"课程"}, Security: secured}, func(context.Context, *struct{}) (*contractCourseFacetsOutput, error) {
		return &contractCourseFacetsOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getCourse", Method: http.MethodGet, Path: "/api/v1/courses/{courseID}", Summary: "读取课程详情与评分概览", Tags: []string{"课程"}, Security: secured}, func(context.Context, *coursePath) (*contractCourseDetailOutput, error) {
		return &contractCourseDetailOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listReviews", Method: http.MethodGet, Path: "/api/v1/courses/{courseID}/reviews", Summary: "读取已审核课程评价", Tags: []string{"评价"}, Security: secured}, func(context.Context, *reviewQueryInput) (*contractReviewListOutput, error) {
		return &contractReviewListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listCoursePosts", Method: http.MethodGet, Path: "/api/v1/courses/{courseID}/posts", Summary: "按课程筛选已审核经验帖", Tags: []string{"经验帖"}, Security: secured}, func(context.Context, *postQueryInput) (*contractPostListOutput, error) {
		return &contractPostListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getPost", Method: http.MethodGet, Path: "/api/v1/posts/{postID}", Summary: "读取经验帖详情", Tags: []string{"经验帖"}, Security: secured}, func(context.Context, *postPath) (*contractPostDetailOutput, error) {
		return &contractPostDetailOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getNotifications", Method: http.MethodGet, Path: "/api/v1/notifications", Summary: "分页读取个人通知", Tags: []string{"通知"}, Security: secured}, func(context.Context, *struct{}) (*contractNotificationListOutput, error) {
		return &contractNotificationListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getProfile", Method: http.MethodGet, Path: "/api/v1/users/me", Summary: "读取个人资料", Tags: []string{"个人"}, Security: secured}, func(context.Context, *struct{}) (*contractProfileOutput, error) { return &contractProfileOutput{}, nil })
	huma.Register(api, huma.Operation{OperationID: "updateProfile", Method: http.MethodPatch, Path: "/api/v1/users/me", Summary: "更新个人资料（选择专业）", Tags: []string{"个人"}, Security: secured}, func(context.Context, *profileUpdateInput) (*contractProfileOutput, error) {
		return &contractProfileOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getHome", Method: http.MethodGet, Path: "/api/v1/home", Summary: "按专业读取首页推荐与热门经验帖", Tags: []string{"首页"}, Security: secured}, func(context.Context, *struct {
		Department string `query:"department" maxLength:"100"`
	}) (*contractHomeOutput, error) {
		return &contractHomeOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listFeedPosts", Method: http.MethodGet, Path: "/api/v1/posts", Summary: "按专业、话题与年份浏览经验帖", Tags: []string{"经验帖"}, Security: secured}, func(context.Context, *struct {
		Department string `query:"department" maxLength:"100"`
		Tag        string `query:"tag" maxLength:"40"`
		Year       string `query:"year" doc:"年份筛选，可传具体年份或 unspecified"`
		Sort       string `query:"sort" enum:"newest,popular"`
		Page       int    `query:"page" minimum:"1" default:"1"`
		PageSize   int    `query:"page_size" minimum:"1" maximum:"100" default:"20"`
	}) (*contractFeedOutput, error) {
		return &contractFeedOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "createReview", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/courses/{courseID}/reviews", Summary: "提交匿名课程评价", Tags: []string{"评价"}, Security: secured}, func(context.Context, *reviewCreateInput) (*contractIDStatusMessageOutput, error) {
		return &contractIDStatusMessageOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "createCoursePost", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/courses/{courseID}/posts", Summary: "提交课程经验帖", Tags: []string{"经验帖"}, Security: secured}, func(context.Context, *postCreateInput) (*contractIDStatusMessageOutput, error) {
		return &contractIDStatusMessageOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "createReport", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/reports", Summary: "举报评价或经验帖", Tags: []string{"治理"}, Security: secured}, func(context.Context, *reportCreateInput) (*contractReportCreatedOutput, error) {
		return &contractReportCreatedOutput{}, nil
	})
	huma.Register(api, mergeOperation(public, huma.Operation{OperationID: "refreshSession", Method: http.MethodPost, Path: "/api/v1/auth/refresh", Summary: "轮换刷新会话令牌"}), func(context.Context, *refreshInput) (*contractSessionOutput, error) {
		return &contractSessionOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "createComment", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/posts/{postID}/comments", Summary: "发表评论", Tags: []string{"互动"}, Security: secured}, func(context.Context, *commentCreateInput) (*contractStatusOutput, error) {
		return &contractStatusOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "markNotificationRead", DefaultStatus: http.StatusNoContent, Method: http.MethodPost, Path: "/api/v1/notifications/{notificationID}/read", Summary: "标记通知已读", Tags: []string{"通知"}, Security: secured}, func(context.Context, *notificationReadInput) (*struct{}, error) {
		return nil, nil
	})
	huma.Register(api, huma.Operation{OperationID: "createCourseCorrection", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/courses/{courseID}/corrections", Summary: "提交课程资料纠错", Tags: []string{"课程"}, Security: secured}, func(context.Context, *correctionCreateInput) (*contractIDStatusMessageOutput, error) {
		return &contractIDStatusMessageOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listAdminReports", Method: http.MethodGet, Path: "/api/v1/admin/reports", Summary: "查看运营举报队列", Tags: []string{"运营"}, Security: secured}, func(context.Context, *struct{}) (*contractReportListOutput, error) {
		return &contractReportListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "decideReport", Method: http.MethodPost, Path: "/api/v1/admin/reports/{reportID}/decision", Summary: "处理举报与内容", Tags: []string{"运营"}, Security: secured}, func(context.Context, *reportDecisionInput) (*contractReportDecisionOutput, error) {
		return &contractReportDecisionOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listCourseCorrections", Method: http.MethodGet, Path: "/api/v1/admin/course-corrections", Summary: "查看课程纠错队列", Tags: []string{"运营"}, Security: secured}, func(context.Context, *struct{}) (*contractCorrectionListOutput, error) {
		return &contractCorrectionListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "decideCourseCorrection", Method: http.MethodPost, Path: "/api/v1/admin/course-corrections/{correctionID}/decision", Summary: "审核课程纠错", Tags: []string{"运营"}, Security: secured}, func(context.Context, *correctionDecisionInput) (*contractStatusOutput, error) {
		return &contractStatusOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listTopicTags", Method: http.MethodGet, Path: "/api/v1/topic-tags", Summary: "读取可用话题标签", Tags: []string{"内容"}, Security: secured}, func(context.Context, *struct{}) (*contractTopicTagListOutput, error) {
		return &contractTopicTagListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "updateTopicTag", Method: http.MethodPatch, Path: "/api/v1/admin/topic-tags/{tagID}", Summary: "维护话题标签", Tags: []string{"运营"}, Security: secured}, func(context.Context, *tagPatchInput) (*contractTopicTagOutput, error) {
		return &contractTopicTagOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getAdminMetrics", Method: http.MethodGet, Path: "/api/v1/admin/metrics", Summary: "查看运营指标", Tags: []string{"运营"}, Security: secured}, func(context.Context, *struct{}) (*contractMetricsOutput, error) { return &contractMetricsOutput{}, nil })
	// Keep the generated contract aligned with every externally reachable product route.
	huma.Register(api, mergeOperation(public, huma.Operation{OperationID: "logout", DefaultStatus: http.StatusNoContent, Method: http.MethodPost, Path: "/api/v1/auth/logout", Summary: "撤销刷新会话"}), func(context.Context, *refreshInput) (*struct{}, error) {
		return nil, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getCurrentUser", Method: http.MethodGet, Path: "/api/v1/auth/me", Summary: "检查当前会话", Tags: []string{"认证"}, Security: secured}, func(context.Context, *struct{}) (*contractCurrentUserOutput, error) {
		return &contractCurrentUserOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "updateReview", Method: http.MethodPut, Path: "/api/v1/courses/{courseID}/reviews/{reviewID}", Summary: "编辑评价并重新提交审核", Tags: []string{"评价"}, Security: secured}, func(context.Context, *reviewUpdateInput) (*contractStatusMessageOutput, error) {
		return &contractStatusMessageOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "reviewDecision", Method: http.MethodPost, Path: "/api/v1/admin/reviews/{reviewID}/decision", Summary: "审核课程评价", Tags: []string{"运营"}, Security: secured}, func(context.Context, *reviewDecisionInput) (*contractStatusOutput, error) {
		return &contractStatusOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listPendingReviews", Method: http.MethodGet, Path: "/api/v1/admin/reviews", Summary: "查看待审核课程评价", Tags: []string{"运营"}, Security: secured}, func(context.Context, *pageQueryInput) (*contractPendingReviewListOutput, error) {
		return &contractPendingReviewListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "updatePost", Method: http.MethodPut, Path: "/api/v1/posts/{postID}", Summary: "编辑经验帖并重新提交审核", Tags: []string{"经验帖"}, Security: secured}, func(context.Context, *postUpdateInput) (*contractStatusMessageOutput, error) {
		return &contractStatusMessageOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "postDecision", Method: http.MethodPost, Path: "/api/v1/admin/posts/{postID}/decision", Summary: "审核经验帖", Tags: []string{"运营"}, Security: secured}, func(context.Context, *postDecisionInput) (*contractStatusOutput, error) {
		return &contractStatusOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listPendingPosts", Method: http.MethodGet, Path: "/api/v1/admin/posts", Summary: "查看待审核经验帖", Tags: []string{"运营"}, Security: secured}, func(context.Context, *pageQueryInput) (*contractPendingPostListOutput, error) {
		return &contractPendingPostListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getPostComments", Method: http.MethodGet, Path: "/api/v1/posts/{postID}/comments", Summary: "读取经验帖一级评论", Tags: []string{"互动"}, Security: secured}, func(context.Context, *postPath) (*contractCommentListOutput, error) {
		return &contractCommentListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "togglePostLike", Method: http.MethodPut, Path: "/api/v1/posts/{postID}/like", Summary: "点赞或取消点赞", Tags: []string{"互动"}, Security: secured}, func(context.Context, *postPath) (*contractStatusOutput, error) { return &contractStatusOutput{}, nil })
	huma.Register(api, huma.Operation{OperationID: "togglePostBookmark", Method: http.MethodPut, Path: "/api/v1/posts/{postID}/bookmark", Summary: "收藏或取消收藏", Tags: []string{"互动"}, Security: secured}, func(context.Context, *postPath) (*contractStatusOutput, error) { return &contractStatusOutput{}, nil })
	huma.Register(api, huma.Operation{OperationID: "removePostLike", DefaultStatus: http.StatusNoContent, Method: http.MethodDelete, Path: "/api/v1/posts/{postID}/like", Summary: "取消点赞", Tags: []string{"互动"}, Security: secured}, func(context.Context, *postPath) (*struct{}, error) { return nil, nil })
	huma.Register(api, huma.Operation{OperationID: "removePostBookmark", DefaultStatus: http.StatusNoContent, Method: http.MethodDelete, Path: "/api/v1/posts/{postID}/bookmark", Summary: "取消收藏", Tags: []string{"互动"}, Security: secured}, func(context.Context, *postPath) (*struct{}, error) { return nil, nil })
	huma.Register(api, huma.Operation{OperationID: "uploadImage", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/uploads", Summary: "上传并清理图片元数据", Tags: []string{"内容"}, Security: secured}, func(context.Context, *struct{}) (*contractImageUploadOutput, error) {
		return &contractImageUploadOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getUploadedImage", DefaultStatus: http.StatusOK, Method: http.MethodGet, Path: "/uploads/{name}", Summary: "读取已上传图片", Tags: []string{"内容"}, Security: secured}, func(context.Context, *struct {
		Name string `path:"name"`
	}) (*struct{}, error) {
		return nil, nil
	})
	huma.Register(api, huma.Operation{OperationID: "myReviews", Method: http.MethodGet, Path: "/api/v1/users/me/reviews", Summary: "查看自己的评价及审核状态", Tags: []string{"个人"}, Security: secured}, func(context.Context, *struct{}) (*contractOwnReviewListOutput, error) {
		return &contractOwnReviewListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "myPosts", Method: http.MethodGet, Path: "/api/v1/users/me/posts", Summary: "查看自己的经验帖及审核状态", Tags: []string{"个人"}, Security: secured}, func(context.Context, *struct{}) (*contractOwnPostListOutput, error) {
		return &contractOwnPostListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "myBookmarks", Method: http.MethodGet, Path: "/api/v1/users/me/bookmarks", Summary: "查看自己的收藏", Tags: []string{"个人"}, Security: secured}, func(context.Context, *struct{}) (*contractBookmarkListOutput, error) {
		return &contractBookmarkListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "myCompletedCourses", Method: http.MethodGet, Path: "/api/v1/users/me/courses", Summary: "查看已修读课程", Tags: []string{"个人"}, Security: secured}, func(context.Context, *struct{}) (*contractCompletedCourseListOutput, error) {
		return &contractCompletedCourseListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "markCourseCompleted", Method: http.MethodPost, Path: "/api/v1/courses/{courseID}/completion", Summary: "标记已修读课程", Tags: []string{"个人"}, Security: secured}, func(context.Context, *coursePath) (*contractStatusOutput, error) {
		return &contractStatusOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "unmarkCourseCompleted", DefaultStatus: http.StatusNoContent, Method: http.MethodDelete, Path: "/api/v1/courses/{courseID}/completion", Summary: "取消已修读课程标记", Tags: []string{"个人"}, Security: secured}, func(context.Context, *coursePath) (*struct{}, error) {
		return nil, nil
	})
	huma.Register(api, huma.Operation{OperationID: "createTopicTag", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/admin/topic-tags", Summary: "创建话题标签", Tags: []string{"运营"}, Security: secured}, func(context.Context, *createTagInput) (*contractTopicTagOutput, error) {
		return &contractTopicTagOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listAllTopicTags", Method: http.MethodGet, Path: "/api/v1/admin/topic-tags", Summary: "管理所有话题标签", Tags: []string{"运营"}, Security: secured}, func(context.Context, *struct{}) (*contractAdminTopicTagListOutput, error) {
		return &contractAdminTopicTagListOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "previewCourseImport", DefaultStatus: http.StatusCreated, Method: http.MethodPost, Path: "/api/v1/admin/course-imports/preview", Summary: "预览 CSV 或 XLSX 课程导入", Tags: []string{"运营"}, Security: secured}, func(context.Context, *struct{}) (*contractImportPreviewOutput, error) {
		return &contractImportPreviewOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "confirmCourseImport", Method: http.MethodPost, Path: "/api/v1/admin/course-imports/{batchID}/confirm", Summary: "确认课程导入批次", Tags: []string{"运营"}, Security: secured}, func(context.Context, *batchPath) (*contractImportConfirmOutput, error) {
		return &contractImportConfirmOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "getCommunityGuidelines", Method: http.MethodGet, Path: "/api/v1/community-guidelines", Summary: "查看审核时限和社区规范", Tags: []string{"内容"}, Security: secured}, func(context.Context, *struct{}) (*contractCommunitySettingsOutput, error) {
		return &contractCommunitySettingsOutput{}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "updateCommunitySettings", Method: http.MethodPut, Path: "/api/v1/admin/community-settings", Summary: "配置审核时限和社区规范", Tags: []string{"运营"}, Security: secured}, func(context.Context, *communitySettingsInput) (*contractStatusOutput, error) {
		return &contractStatusOutput{}, nil
	})
	imageResponse := api.OpenAPI().Paths["/uploads/{name}"].Get.Responses["200"]
	imageResponse.Content = map[string]*huma.MediaType{"image/jpeg": {Schema: &huma.Schema{Type: "string", Format: "binary"}}}
	return mux, api.OpenAPI()
}

func mergeOperation(base, override huma.Operation) huma.Operation {
	base.OperationID = override.OperationID
	base.Method = override.Method
	base.Path = override.Path
	base.Summary = override.Summary
	if override.DefaultStatus != 0 {
		base.DefaultStatus = override.DefaultStatus
	}
	return base
}
func OpenAPIJSON() ([]byte, error) {
	_, spec := newContractAPI()
	return json.MarshalIndent(spec, "", "  ")
}

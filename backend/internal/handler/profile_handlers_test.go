package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"skill-hub/internal/config"
	"skill-hub/internal/model"
	"skill-hub/internal/service"
	"skill-hub/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestListMyUploads_ReturnsPaginatedUploadsStatsAndActivities(t *testing.T) {
	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)

	reviewerID := uint(7)
	reviewedAt := time.Now().Add(-30 * time.Minute)
	uploadAtOld := time.Now().Add(-4 * time.Hour)
	uploadAtMid := time.Now().Add(-2 * time.Hour)
	uploadAtNew := time.Now().Add(-1 * time.Hour)

	users := []model.User{
		{ID: reviewerID, Username: "alice", Password: "hash"},
		{ID: 8, Username: "bob", Password: "hash"},
	}
	for _, user := range users {
		if err := tdb.DB.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	ownedSkills := []model.Skill{
		{
			UserID:            reviewerID,
			Name:              "Alpha Skill",
			Author:            "alice",
			ResourceType:      "skill",
			Downloads:         10,
			LikesCount:        2,
			Tags:              "go,react",
			CreatedAt:         uploadAtOld,
			UpdatedAt:         uploadAtOld,
			AIApproved:        true,
			HumanReviewStatus: model.HumanReviewStatusApproved,
		},
		{
			UserID:            0,
			Name:              "Legacy Upload",
			Author:            "Alice",
			ResourceType:      "rules",
			Downloads:         4,
			LikesCount:        1,
			Tags:              "go,docker",
			CreatedAt:         uploadAtMid,
			UpdatedAt:         uploadAtMid,
			AIApproved:        false,
			HumanReviewStatus: model.HumanReviewStatusPending,
		},
		{
			UserID:            reviewerID,
			Name:              "Newest Tool",
			Author:            "alice",
			ResourceType:      "tools",
			Downloads:         1,
			LikesCount:        0,
			Tags:              "cli",
			CreatedAt:         uploadAtNew,
			UpdatedAt:         uploadAtNew,
			AIApproved:        true,
			HumanReviewStatus: model.HumanReviewStatusApproved,
		},
	}
	for _, skill := range ownedSkills {
		if err := tdb.DB.Create(&skill).Error; err != nil {
			t.Fatalf("create owned skill: %v", err)
		}
	}

	reviewedSkill := model.Skill{
		UserID:            8,
		Name:              "Reviewed Rule",
		Author:            "bob",
		ResourceType:      "rules",
		Downloads:         8,
		LikesCount:        3,
		CreatedAt:         uploadAtOld,
		UpdatedAt:         reviewedAt,
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		HumanReviewerID:   &reviewerID,
		HumanReviewedAt:   &reviewedAt,
	}
	if err := tdb.DB.Create(&reviewedSkill).Error; err != nil {
		t.Fatalf("create reviewed skill: %v", err)
	}

	handler := NewSkillHandler(svc, nil, nil, nil, &config.Config{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", reviewerID)
		c.Set("username", "alice")
		c.Next()
	})
	router.GET("/me/uploads", handler.ListMyUploads)

	req := httptest.NewRequest(http.MethodGet, "/me/uploads?page=1&page_size=2", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Skills     []model.Skill `json:"skills"`
		Total      int           `json:"total"`
		Page       int           `json:"page"`
		PageSize   int           `json:"page_size"`
		TopTags    []string      `json:"top_tags"`
		Activities []struct {
			Kind         string `json:"kind"`
			Target       string `json:"target"`
			ResourceType string `json:"resource_type"`
		} `json:"activities"`
		Stats struct {
			TotalItems     int `json:"total_items"`
			TotalDownloads int `json:"total_downloads"`
			TotalLikes     int `json:"total_likes"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if payload.Total != 3 {
		t.Fatalf("expected total uploads 3, got %d", payload.Total)
	}
	if payload.Page != 1 || payload.PageSize != 2 {
		t.Fatalf("expected page metadata 1/2, got page=%d size=%d", payload.Page, payload.PageSize)
	}
	if len(payload.Skills) != 2 {
		t.Fatalf("expected first page to contain 2 uploads, got %d", len(payload.Skills))
	}
	if payload.Skills[0].Name != "Alpha Skill" || payload.Skills[1].Name != "Legacy Upload" {
		t.Fatalf("expected uploads ordered by downloads desc, got %q then %q", payload.Skills[0].Name, payload.Skills[1].Name)
	}
	if payload.Stats.TotalItems != 3 {
		t.Fatalf("expected total_items=3, got %d", payload.Stats.TotalItems)
	}
	if payload.Stats.TotalDownloads != 15 {
		t.Fatalf("expected total_downloads=15, got %d", payload.Stats.TotalDownloads)
	}
	if payload.Stats.TotalLikes != 3 {
		t.Fatalf("expected total_likes=3, got %d", payload.Stats.TotalLikes)
	}
	if len(payload.TopTags) == 0 || payload.TopTags[0] != "go" {
		t.Fatalf("expected top tags to start with go, got %v", payload.TopTags)
	}
	if len(payload.Activities) < 2 {
		t.Fatalf("expected at least 2 activities, got %d", len(payload.Activities))
	}
	if payload.Activities[0].Kind != "approved" || payload.Activities[0].Target != "Reviewed Rule" {
		t.Fatalf("expected latest activity to be approved review, got %+v", payload.Activities[0])
	}
	if payload.Activities[1].Kind != "published" || payload.Activities[1].Target != "Newest Tool" {
		t.Fatalf("expected next activity to be newest upload, got %+v", payload.Activities[1])
	}
}

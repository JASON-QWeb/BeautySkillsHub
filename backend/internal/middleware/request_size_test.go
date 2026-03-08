package middleware

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimitMultipartBody_RejectsOversizedMultipartRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var reached bool
	router := gin.New()
	router.POST("/upload",
		LimitMultipartBody(128, 32<<10),
		func(c *gin.Context) {
			reached = true
			c.Status(http.StatusNoContent)
		},
	)

	req := newMultipartRequest(t, "file", "big.bin", bytes.Repeat([]byte("a"), 512))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if reached {
		t.Fatal("expected oversized multipart request to be rejected before handler")
	}
	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized multipart request, got %d", resp.Code)
	}
}

func TestLimitMultipartBody_AllowsMultipartRequestsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var reached bool
	router := gin.New()
	router.POST("/upload",
		LimitMultipartBody(4096, 32<<10),
		func(c *gin.Context) {
			reached = true
			f, _, err := c.Request.FormFile("file")
			if err != nil {
				t.Fatalf("expected parsed multipart file, got error: %v", err)
			}
			_ = f.Close()
			c.Status(http.StatusNoContent)
		},
	)

	req := newMultipartRequest(t, "file", "small.bin", []byte("ok"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if !reached {
		t.Fatal("expected request within limit to reach handler")
	}
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for allowed multipart request, got %d", resp.Code)
	}
}

func newMultipartRequest(t *testing.T, fieldName, fileName string, payload []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(payload)); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

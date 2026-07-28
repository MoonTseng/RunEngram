package handler

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"testing"

	"taskline_server/api/model"
	"taskline_server/internal/config"
	"taskline_server/internal/store"
)

func TestTaskAttachmentStorageDocLifecycle(t *testing.T) {
	storage := newTestAttachmentStorage(t)

	doc, err := storage.SaveDoc("task-1", "Spec", "# Draft")
	if err != nil {
		t.Fatalf("SaveDoc() error = %v", err)
	}
	if doc.ID == "" {
		t.Fatal("SaveDoc() returned empty doc id")
	}
	if filepath.Base(doc.StoragePath) != doc.ID+".md" {
		t.Fatalf("SaveDoc() storage path = %q, want id.md suffix", doc.StoragePath)
	}

	content, err := storage.ReadDocContent(doc)
	if err != nil {
		t.Fatalf("ReadDocContent() error = %v", err)
	}
	if string(content) != "# Draft" {
		t.Fatalf("ReadDocContent() = %q, want %q", string(content), "# Draft")
	}

	tempPath, err := storage.WriteDocContentTemp(doc, "# Updated")
	if err != nil {
		t.Fatalf("WriteDocContentTemp() error = %v", err)
	}
	if err := storage.CommitDocContent(doc, tempPath); err != nil {
		t.Fatalf("CommitDocContent() error = %v", err)
	}
	content, err = storage.ReadDocContent(doc)
	if err != nil {
		t.Fatalf("ReadDocContent() after commit error = %v", err)
	}
	if string(content) != "# Updated" {
		t.Fatalf("ReadDocContent() after commit = %q, want %q", string(content), "# Updated")
	}

	storage.AttachDocURL(doc)
	if doc.URL != "/api/v1/docs/"+doc.ID+"/content" {
		t.Fatalf("AttachDocURL() = %q", doc.URL)
	}
	if err := storage.DeleteFile(doc.StoragePath); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := storage.ReadDocContent(doc); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ReadDocContent() after delete error = %v, want ErrNotFound", err)
	}
}

func TestTaskAttachmentStorageImageLifecycle(t *testing.T) {
	storage := newTestAttachmentStorage(t)
	fh := multipartFileHeader(t, "example.png", "image/png", []byte("png-data"))

	img, err := storage.SaveImage("task-1", fh)
	if err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	if img.ID == "" {
		t.Fatal("SaveImage() returned empty image id")
	}
	if img.Filename != "example.png" || img.MimeType != "image/png" || img.SizeBytes != 8 {
		t.Fatalf("SaveImage() metadata = %#v", img)
	}

	content, err := storage.ReadImageContent(img)
	if err != nil {
		t.Fatalf("ReadImageContent() error = %v", err)
	}
	if content.Path != img.StoragePath || content.Filename != "example.png" {
		t.Fatalf("ReadImageContent() = %#v", content)
	}
	if content.ContentType != "image/png" {
		t.Fatalf("ReadImageContent() content type = %q, want image/png", content.ContentType)
	}

	storage.AttachImageURL(img)
	if img.URL != "/api/v1/images/"+img.ID {
		t.Fatalf("AttachImageURL() = %q", img.URL)
	}

	task := &model.Task{Images: []model.Image{*img}, Docs: []model.Doc{{ID: "doc-1"}}}
	storage.AttachTaskImageURLs(task)
	storage.AttachTaskDocURLs(task)
	if task.Images[0].URL != "/api/v1/images/"+img.ID {
		t.Fatalf("AttachTaskImageURLs() = %q", task.Images[0].URL)
	}
	if task.Docs[0].URL != "/api/v1/docs/doc-1/content" {
		t.Fatalf("AttachTaskDocURLs() = %q", task.Docs[0].URL)
	}

	if err := storage.DeleteFile(img.StoragePath); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := storage.ReadImageContent(img); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ReadImageContent() after delete error = %v, want ErrNotFound", err)
	}
}

func newTestAttachmentStorage(t *testing.T) *taskAttachmentStorage {
	t.Helper()
	tmp := t.TempDir()
	return newTaskAttachmentStorage(&config.Config{
		ImagesDir: filepath.Join(tmp, "images"),
		DocsDir:   filepath.Join(tmp, "docs"),
	})
}

func multipartFileHeader(t *testing.T, filename, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(textprotoMIMEHeader(filename, contentType))
	if err != nil {
		t.Fatalf("CreatePart() error = %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "/", &body)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(int64(body.Len())); err != nil {
		t.Fatalf("ParseMultipartForm() error = %v", err)
	}
	files := req.MultipartForm.File["file"]
	if len(files) != 1 {
		t.Fatalf("multipart file count = %d, want 1", len(files))
	}
	return files[0]
}

func textprotoMIMEHeader(filename, contentType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	return header
}

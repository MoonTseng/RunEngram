package handler

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"taskline_server/api/model"
	"taskline_server/internal/config"
	"taskline_server/internal/store"
)

type taskAttachmentStorage struct {
	imagesDir string
	docsDir   string
}

type imageContent struct {
	Path        string
	Filename    string
	ContentType string
}

func newTaskAttachmentStorage(cfg *config.Config) *taskAttachmentStorage {
	return &taskAttachmentStorage{
		imagesDir: cfg.ImagesDir,
		docsDir:   cfg.DocsDir,
	}
}

func (s *taskAttachmentStorage) SaveImage(taskID string, fh *multipart.FileHeader) (*model.Image, error) {
	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()

	taskDir := filepath.Join(s.imagesDir, taskID)
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		return nil, err
	}
	imgID := uuid.NewString()
	ext := filepath.Ext(fh.Filename)
	storagePath := filepath.Join(taskDir, imgID+ext)
	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, err
	}
	written, err := io.Copy(dst, src)
	cerr := dst.Close()
	if err != nil {
		_ = os.Remove(storagePath)
		return nil, err
	}
	if cerr != nil {
		_ = os.Remove(storagePath)
		return nil, cerr
	}

	mimeType := "application/octet-stream"
	if v := fh.Header.Get("Content-Type"); v != "" {
		mimeType = v
	}
	return &model.Image{
		ID:          imgID,
		TaskID:      taskID,
		Filename:    fh.Filename,
		MimeType:    mimeType,
		SizeBytes:   written,
		StoragePath: storagePath,
	}, nil
}

func (s *taskAttachmentStorage) ReadImageContent(img *model.Image) (*imageContent, error) {
	if _, err := os.Stat(img.StoragePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: image file missing", store.ErrNotFound)
		}
		return nil, err
	}
	contentType := img.MimeType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(img.Filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &imageContent{
		Path:        img.StoragePath,
		Filename:    img.Filename,
		ContentType: contentType,
	}, nil
}

func (s *taskAttachmentStorage) SaveDoc(taskID, title, content string) (*model.Doc, error) {
	docID := uuid.NewString()
	taskDir := filepath.Join(s.docsDir, taskID)
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		return nil, err
	}
	storagePath := filepath.Join(taskDir, docID+".md")
	if err := os.WriteFile(storagePath, []byte(content), 0o600); err != nil {
		return nil, err
	}
	return &model.Doc{
		ID:          docID,
		TaskID:      taskID,
		Title:       title,
		StoragePath: storagePath,
	}, nil
}

func (s *taskAttachmentStorage) ReadDocContent(doc *model.Doc) ([]byte, error) {
	content, err := os.ReadFile(doc.StoragePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: doc file missing", store.ErrNotFound)
		}
		return nil, err
	}
	return content, nil
}

func (s *taskAttachmentStorage) WriteDocContentTemp(doc *model.Doc, content string) (string, error) {
	dir := filepath.Dir(doc.StoragePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(dir, filepath.Base(doc.StoragePath)+".*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	if _, err := temp.WriteString(content); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return tempPath, nil
}

func (s *taskAttachmentStorage) CommitDocContent(doc *model.Doc, tempPath string) error {
	if err := os.Rename(tempPath, doc.StoragePath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func (s *taskAttachmentStorage) DeleteFile(storagePath string) error {
	if storagePath == "" {
		return nil
	}
	return os.Remove(storagePath)
}

func (s *taskAttachmentStorage) AttachTaskImageURLs(tasks ...*model.Task) {
	for _, task := range tasks {
		if task == nil {
			continue
		}
		for i := range task.Images {
			s.AttachImageURL(&task.Images[i])
		}
	}
}

func (s *taskAttachmentStorage) AttachTaskDocURLs(tasks ...*model.Task) {
	for _, task := range tasks {
		if task == nil {
			continue
		}
		for i := range task.Docs {
			s.AttachDocURL(&task.Docs[i])
		}
	}
}

func (s *taskAttachmentStorage) AttachImageURL(img *model.Image) {
	if img == nil || img.ID == "" {
		return
	}
	img.URL = "/api/v1/images/" + url.PathEscape(img.ID)
}

func (s *taskAttachmentStorage) AttachDocURL(doc *model.Doc) {
	if doc == nil || doc.ID == "" {
		return
	}
	doc.URL = "/api/v1/docs/" + url.PathEscape(doc.ID) + "/content"
}

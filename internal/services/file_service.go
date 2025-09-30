package services

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

type FileService struct {
	uploadDir   string
	maxFileSize int64
}

func NewFileService(uploadDir string, maxFileSize int64) *FileService {
	os.MkdirAll(uploadDir, 0755)

	return &FileService{
		uploadDir:   uploadDir,
		maxFileSize: maxFileSize,
	}
}

// SaveFile saves uploaded file and returns file path
func (s *FileService) SaveFile(file *multipart.FileHeader) (string, error) {
	if file.Size > s.maxFileSize {
		return "", errors.New("file size exceeds maximum allowed size")
	}

	allowedTypes := map[string]bool{
		"application/pdf": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"text/plain": true,
	}

	if !allowedTypes[file.Header.Get("Content-Type")] {
		return "", errors.New("unsupported file type")
	}

	filename := fmt.Sprintf("%d_%s", file.Size, file.Filename)
	filePath := filepath.Join(s.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	return filePath, nil
}

// ExtractTextFromFile extracts text from various file formats
func (s *FileService) ExtractTextFromFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".pdf":
		return s.extractTextFromPDF(filePath)
	case ".docx":
		return s.extractTextFromDOCX(filePath)
	case ".txt":
		return s.extractTextFromTXT(filePath)
	default:
		return "", errors.New("unsupported file format")
	}
}

func (s *FileService) extractTextFromPDF(filePath string) (string, error) {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var text strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		text.WriteString(content)
		text.WriteString("\n")
	}

	return text.String(), nil
}

func (s *FileService) extractTextFromDOCX(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer reader.Close()

	var text strings.Builder
	foundDocument := false

	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			foundDocument = true
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open document.xml: %w", err)
			}
			defer rc.Close()

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			if err != nil {
				return "", fmt.Errorf("failed to read document.xml: %w", err)
			}

			xmlContent := buf.String()
			textContent := s.extractTextFromXML(xmlContent)
			text.WriteString(textContent)
			break
		}
	}

	if !foundDocument {
		return "", fmt.Errorf("document.xml not found in DOCX file")
	}

	result := text.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("no readable text found in DOCX file")
	}

	return result, nil
}

func (s *FileService) extractTextFromXML(xmlContent string) string {
	var text strings.Builder

	lines := strings.Split(xmlContent, "\n")

	if len(lines) == 1 {
		// Handle single-line XML
		content := xmlContent
		start := 0

		for {
			tagStart := strings.Index(content[start:], "<w:t")
			if tagStart == -1 {
				break
			}
			tagStart += start

			openEnd := strings.Index(content[tagStart:], ">")
			if openEnd == -1 {
				break
			}
			openEnd += tagStart

			closeStart := strings.Index(content[openEnd:], "</w:t>")
			if closeStart != -1 {
				closeStart += openEnd
				textContent := content[openEnd+1 : closeStart]
				textContent = s.decodeXMLEntities(textContent)
				text.WriteString(textContent)
				text.WriteString(" ")

				start = closeStart
			} else {
				break
			}
		}
	} else {
		// Handle multi-line XML
		for _, line := range lines {
			line = strings.TrimSpace(line)

			start := strings.Index(line, "<w:t>")
			for start != -1 {
				end := strings.Index(line[start:], "</w:t>")
				if end != -1 {
					end += start
					textContent := line[start+5 : end]
					textContent = s.decodeXMLEntities(textContent)
					text.WriteString(textContent)
					text.WriteString(" ")

					start = strings.Index(line[end:], "<w:t>")
					if start != -1 {
						start += end
					}
				} else {
					break
				}
			}
		}
	}

	return text.String()
}

func (s *FileService) decodeXMLEntities(text string) string {
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	return text
}

func (s *FileService) extractTextFromTXT(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (s *FileService) CleanupFile(filePath string) error {
	return os.Remove(filePath)
}

func (s *FileService) GetFileInfo(filePath string) (os.FileInfo, error) {
	return os.Stat(filePath)
}

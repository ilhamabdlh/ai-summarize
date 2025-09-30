package handlers

import (
	"net/http"
	"path/filepath"

	"ai-cv-summarize/internal/models"
	"ai-cv-summarize/internal/services"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	fileService *services.FileService
}

func NewUploadHandler(fileService *services.FileService) *UploadHandler {
	return &UploadHandler{
		fileService: fileService,
	}
}

// UploadFiles handles file upload for CV and project report
func (h *UploadHandler) UploadFiles(c *gin.Context) {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	// Get CV file
	cvFiles := form.File["cv_file"]
	if len(cvFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CV file is required"})
		return
	}

	// Get project file
	projectFiles := form.File["project_file"]
	if len(projectFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project file is required"})
		return
	}

	// Save CV file
	cvFile := cvFiles[0]
	cvFilePath, err := h.fileService.SaveFile(cvFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save CV file: " + err.Error()})
		return
	}

	// Save project file
	projectFile := projectFiles[0]
	projectFilePath, err := h.fileService.SaveFile(projectFile)
	if err != nil {
		// Cleanup CV file if project file save fails
		h.fileService.CleanupFile(cvFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save project file: " + err.Error()})
		return
	}

	// Extract text content from files (for validation)
	_, err = h.fileService.ExtractTextFromFile(cvFilePath)
	if err != nil {
		// Cleanup files if text extraction fails
		h.fileService.CleanupFile(cvFilePath)
		h.fileService.CleanupFile(projectFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract CV content: " + err.Error()})
		return
	}

	_, err = h.fileService.ExtractTextFromFile(projectFilePath)
	if err != nil {
		// Cleanup files if text extraction fails
		h.fileService.CleanupFile(cvFilePath)
		h.fileService.CleanupFile(projectFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract project content: " + err.Error()})
		return
	}

	// Return success response with actual saved filenames
	response := models.UploadResponse{
		Message:     "Files uploaded successfully",
		CVFile:      filepath.Base(cvFilePath),      // Return the actual saved filename
		ProjectFile: filepath.Base(projectFilePath), // Return the actual saved filename
	}

	c.JSON(http.StatusOK, response)
}

// UploadFilesWithContent handles file upload and returns content
func (h *UploadHandler) UploadFilesWithContent(c *gin.Context) {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	// Get CV file
	cvFiles := form.File["cv_file"]
	if len(cvFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CV file is required"})
		return
	}

	// Get project file
	projectFiles := form.File["project_file"]
	if len(projectFiles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project file is required"})
		return
	}

	// Save CV file
	cvFile := cvFiles[0]
	cvFilePath, err := h.fileService.SaveFile(cvFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save CV file: " + err.Error()})
		return
	}

	// Save project file
	projectFile := projectFiles[0]
	projectFilePath, err := h.fileService.SaveFile(projectFile)
	if err != nil {
		// Cleanup CV file if project file save fails
		h.fileService.CleanupFile(cvFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save project file: " + err.Error()})
		return
	}

	// Extract text content from files
	cvContent, err := h.fileService.ExtractTextFromFile(cvFilePath)
	if err != nil {
		// Cleanup files if text extraction fails
		h.fileService.CleanupFile(cvFilePath)
		h.fileService.CleanupFile(projectFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract CV content: " + err.Error()})
		return
	}

	projectContent, err := h.fileService.ExtractTextFromFile(projectFilePath)
	if err != nil {
		// Cleanup files if text extraction fails
		h.fileService.CleanupFile(cvFilePath)
		h.fileService.CleanupFile(projectFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract project content: " + err.Error()})
		return
	}

	// Return success response with content
	response := gin.H{
		"message":         "Files uploaded and processed successfully",
		"cv_file":         filepath.Base(cvFilePath),
		"project_file":    filepath.Base(projectFilePath),
		"cv_content":      cvContent,
		"project_content": projectContent,
	}

	c.JSON(http.StatusOK, response)
}

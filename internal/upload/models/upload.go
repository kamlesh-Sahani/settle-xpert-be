package models

type PresignedURLRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	Folder      string `json:"folder" binding:"omitempty"`
}

type PresignedURLResponse struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
	Key    string            `json:"key"`
}

type VerifyUploadRequest struct {
	Key string `json:"key" binding:"required"`
}

type VerifyUploadResponse struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

type DeleteFilesRequest struct {
	Keys []string `json:"keys" binding:"required,min=1"`
}

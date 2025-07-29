package unit

import (
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/attachment"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
)

func TestAttachmentClient_NewAttachmentClient(t *testing.T) {
	client := &core.Client{} // Mock client
	attachmentClient := attachment.NewAttachmentClient(client)
	
	if attachmentClient == nil {
		t.Fatal("NewAttachmentClient should return a non-nil AttachmentClient")
	}
}

func TestAttachment(t *testing.T) {
	testAttachment := attachment.Attachment{
		SysID:          "test_sys_id",
		FileName:       "test_file.pdf",
		ContentType:    "application/pdf",
		SizeBytes:      1024,
		SizeCompressed: 512,
		Compressed:     true,
		State:          "available",
		TableName:      "incident",
		TableSysID:     "incident_sys_id",
		DownloadLink:   "/api/now/attachment/test_sys_id/file",
		CreatedBy:      "admin",
		CreatedOn:      time.Now(),
		UpdatedBy:      "admin",
		UpdatedOn:      time.Now(),
		Hash:           "abc123",
	}
	
	if testAttachment.SysID != "test_sys_id" {
		t.Errorf("Expected SysID 'test_sys_id', got '%s'", testAttachment.SysID)
	}
	
	if testAttachment.FileName != "test_file.pdf" {
		t.Errorf("Expected FileName 'test_file.pdf', got '%s'", testAttachment.FileName)
	}
	
	if testAttachment.SizeBytes != 1024 {
		t.Errorf("Expected SizeBytes 1024, got %d", testAttachment.SizeBytes)
	}
	
	if !testAttachment.Compressed {
		t.Error("Expected Compressed to be true")
	}
}

func TestUploadRequest(t *testing.T) {
	request := attachment.UploadRequest{
		TableName:   "incident",
		TableSysID:  "incident_sys_id",
		FileName:    "test.txt",
		ContentType: "text/plain",
		Encryption:  "none",
		Metadata: map[string]string{
			"description": "Test file",
			"category":    "documentation",
		},
	}
	
	if request.TableName != "incident" {
		t.Errorf("Expected TableName 'incident', got '%s'", request.TableName)
	}
	
	if request.TableSysID != "incident_sys_id" {
		t.Errorf("Expected TableSysID 'incident_sys_id', got '%s'", request.TableSysID)
	}
	
	if len(request.Metadata) != 2 {
		t.Errorf("Expected 2 metadata items, got %d", len(request.Metadata))
	}
	
	if request.Metadata["description"] != "Test file" {
		t.Errorf("Expected description 'Test file', got '%s'", request.Metadata["description"])
	}
}

func TestUploadFromBytesRequest(t *testing.T) {
	data := []byte("Hello, World!")
	request := attachment.UploadFromBytesRequest{
		UploadRequest: attachment.UploadRequest{
			TableName:   "task",
			TableSysID:  "task_sys_id",
			FileName:    "hello.txt",
			ContentType: "text/plain",
		},
		Data: data,
	}
	
	if request.TableName != "task" {
		t.Errorf("Expected TableName 'task', got '%s'", request.TableName)
	}
	
	if string(request.Data) != "Hello, World!" {
		t.Errorf("Expected Data 'Hello, World!', got '%s'", string(request.Data))
	}
	
	if len(request.Data) != 13 {
		t.Errorf("Expected Data length 13, got %d", len(request.Data))
	}
}

func TestUploadResult(t *testing.T) {
	result := attachment.UploadResult{
		Attachment: &attachment.Attachment{
			SysID:       "upload_sys_id",
			FileName:    "uploaded_file.txt",
			ContentType: "text/plain",
			SizeBytes:   256,
		},
		Success: true,
		Message: "File uploaded successfully",
		Error:   "",
	}
	
	if !result.Success {
		t.Error("Expected Success to be true")
	}
	
	if result.Message != "File uploaded successfully" {
		t.Errorf("Expected Message 'File uploaded successfully', got '%s'", result.Message)
	}
	
	if result.Attachment == nil {
		t.Fatal("Expected Attachment to be non-nil")
	}
	
	if result.Attachment.SysID != "upload_sys_id" {
		t.Errorf("Expected Attachment SysID 'upload_sys_id', got '%s'", result.Attachment.SysID)
	}
}

func TestAttachmentFilter(t *testing.T) {
	filter := attachment.AttachmentFilter{
		TableName:     "incident",
		TableSysID:    "incident_sys_id",
		FileName:      "report.pdf",
		ContentType:   "application/pdf",
		CreatedBy:     "admin",
		CreatedAfter:  "2024-01-01",
		CreatedBefore: "2024-12-31",
		MinSize:       1024,
		MaxSize:       10485760, // 10MB
		Limit:         50,
		Offset:        0,
	}
	
	if filter.TableName != "incident" {
		t.Errorf("Expected TableName 'incident', got '%s'", filter.TableName)
	}
	
	if filter.MinSize != 1024 {
		t.Errorf("Expected MinSize 1024, got %d", filter.MinSize)
	}
	
	if filter.MaxSize != 10485760 {
		t.Errorf("Expected MaxSize 10485760, got %d", filter.MaxSize)
	}
	
	if filter.Limit != 50 {
		t.Errorf("Expected Limit 50, got %d", filter.Limit)
	}
}

func TestValidateFileName(t *testing.T) {
	// Valid filenames
	validNames := []string{
		"document.pdf",
		"report_2024.xlsx",
		"image (1).jpg",
		"test-file.txt",
		"data.json",
	}
	
	for _, name := range validNames {
		err := attachment.ValidateFileName(name)
		if err != nil {
			t.Errorf("Expected '%s' to be valid, but got error: %v", name, err)
		}
	}
	
	// Invalid filenames
	invalidNames := []string{
		"",           // empty
		"file<test>", // contains < >
		"file:name",  // contains :
		"file\"name", // contains "
		"file|name",  // contains |
		"file?name",  // contains ?
		"file*name",  // contains *
		"file/name",  // contains /
		"file\\name", // contains \
	}
	
	for _, name := range invalidNames {
		err := attachment.ValidateFileName(name)
		if err == nil {
			t.Errorf("Expected '%s' to be invalid, but validation passed", name)
		}
	}
	
	// Test very long filename
	longName := string(make([]byte, 300)) // 300 characters
	err := attachment.ValidateFileName(longName)
	if err == nil {
		t.Error("Expected very long filename to be invalid")
	}
}

func TestGetMimeType(t *testing.T) {
	testCases := map[string]string{
		"document.pdf":        "application/pdf",
		"image.jpg":           "image/jpeg",
		"image.jpeg":          "image/jpeg",
		"image.png":           "image/png",
		"image.gif":           "image/gif",
		"spreadsheet.xlsx":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"document.docx":       "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"presentation.pptx":   "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text.txt":            "text/plain",
		"data.json":           "application/json",
		"data.xml":            "application/xml",
		"data.csv":            "text/csv",
		"page.html":           "text/html",
		"style.css":           "text/css",
		"script.js":           "application/javascript",
		"archive.zip":         "application/zip",
		"unknown.xyz":         "application/octet-stream", // unknown extension
		"no_extension":        "application/octet-stream", // no extension
	}
	
	for fileName, expectedMimeType := range testCases {
		actualMimeType := attachment.GetMimeType(fileName)
		if actualMimeType != expectedMimeType {
			t.Errorf("For file '%s', expected MIME type '%s', got '%s'", 
				fileName, expectedMimeType, actualMimeType)
		}
	}
}

func TestAttachmentWithMultipleFiles(t *testing.T) {
	// Test scenario with multiple attachments
	attachments := []attachment.Attachment{
		{
			SysID:       "attach1",
			FileName:    "file1.pdf",
			ContentType: "application/pdf",
			SizeBytes:   1024,
			TableName:   "incident",
			TableSysID:  "inc123",
		},
		{
			SysID:       "attach2",
			FileName:    "file2.jpg",
			ContentType: "image/jpeg",
			SizeBytes:   2048,
			TableName:   "incident",
			TableSysID:  "inc123",
		},
		{
			SysID:       "attach3",
			FileName:    "file3.txt",
			ContentType: "text/plain",
			SizeBytes:   512,
			TableName:   "task",
			TableSysID:  "task456",
		},
	}
	
	if len(attachments) != 3 {
		t.Errorf("Expected 3 attachments, got %d", len(attachments))
	}
	
	// Test filtering by table
	incidentAttachments := 0
	taskAttachments := 0
	
	for _, att := range attachments {
		if att.TableName == "incident" {
			incidentAttachments++
		} else if att.TableName == "task" {
			taskAttachments++
		}
	}
	
	if incidentAttachments != 2 {
		t.Errorf("Expected 2 incident attachments, got %d", incidentAttachments)
	}
	
	if taskAttachments != 1 {
		t.Errorf("Expected 1 task attachment, got %d", taskAttachments)
	}
	
	// Test total size calculation
	totalSize := int64(0)
	for _, att := range attachments {
		totalSize += att.SizeBytes
	}
	
	expectedTotal := int64(1024 + 2048 + 512)
	if totalSize != expectedTotal {
		t.Errorf("Expected total size %d, got %d", expectedTotal, totalSize)
	}
}

func TestAttachmentMetadata(t *testing.T) {
	// Test attachment with additional metadata in attributes
	testAttachment := attachment.Attachment{
		SysID:       "meta_test",
		FileName:    "test_with_metadata.pdf",
		ContentType: "application/pdf",
		SizeBytes:   4096,
		TableName:   "change_request",
		TableSysID:  "change123",
		Hash:        "md5:abcd1234",
		ImageWidth:  0, // Not an image
		ImageHeight: 0, // Not an image
	}
	
	// Verify basic properties
	if testAttachment.FileName != "test_with_metadata.pdf" {
		t.Errorf("Expected FileName 'test_with_metadata.pdf', got '%s'", testAttachment.FileName)
	}
	
	if testAttachment.TableName != "change_request" {
		t.Errorf("Expected TableName 'change_request', got '%s'", testAttachment.TableName)
	}
	
	if testAttachment.ImageWidth != 0 {
		t.Errorf("Expected ImageWidth 0 for PDF, got %d", testAttachment.ImageWidth)
	}
	
	if testAttachment.Hash != "md5:abcd1234" {
		t.Errorf("Expected Hash 'md5:abcd1234', got '%s'", testAttachment.Hash)
	}
}
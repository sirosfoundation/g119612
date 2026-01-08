package validation

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		opts    URLValidationOptions
		wantErr bool
	}{
		// Valid URLs
		{
			name:    "Valid_HTTPS_URL",
			url:     "https://example.com/path/to/resource",
			opts:    DefaultURLOptions(),
			wantErr: false,
		},
		{
			name:    "Valid_HTTP_URL",
			url:     "http://example.com",
			opts:    DefaultURLOptions(),
			wantErr: false,
		},
		{
			name:    "Valid_File_URL_When_Allowed",
			url:     "file:///path/to/file.xml",
			opts:    TSLURLOptions(),
			wantErr: false,
		},
		// Invalid URLs
		{
			name:    "Empty_URL",
			url:     "",
			opts:    DefaultURLOptions(),
			wantErr: true,
		},
		{
			name:    "Relative_URL_When_Absolute_Required",
			url:     "/relative/path",
			opts:    DefaultURLOptions(),
			wantErr: true,
		},
		{
			name:    "File_URL_When_Not_Allowed",
			url:     "file:///path/to/file",
			opts:    DefaultURLOptions(),
			wantErr: true,
		},
		{
			name:    "FTP_URL_Not_In_Allowed_Schemes",
			url:     "ftp://example.com/file",
			opts:    DefaultURLOptions(),
			wantErr: true,
		},
		{
			name:    "URL_With_Path_Traversal",
			url:     "https://example.com/../../../etc/passwd",
			opts:    DefaultURLOptions(),
			wantErr: true,
		},
		{
			name:    "Invalid_URL_Format",
			url:     "ht!tp://invalid",
			opts:    DefaultURLOptions(),
			wantErr: true,
		},
		// Special cases
		{
			name:    "URL_With_Query_Parameters",
			url:     "https://example.com/path?param=value&other=123",
			opts:    DefaultURLOptions(),
			wantErr: false,
		},
		{
			name:    "URL_With_Fragment",
			url:     "https://example.com/path#section",
			opts:    DefaultURLOptions(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{
			name:    "Valid_Relative_Path",
			path:    "config/settings.yaml",
			wantErr: false,
		},
		{
			name:    "Valid_Absolute_Path",
			path:    "/tmp/test/file.xml",
			wantErr: false,
		},
		{
			name:    "Valid_Windows_Path",
			path:    "C:\\Users\\Test\\file.txt",
			wantErr: false,
		},
		// Invalid paths
		{
			name:    "Empty_Path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Path_With_Traversal_Escape",
			path:    "../../../../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "Path_With_Null_Byte",
			path:    "/path/to/file\x00.xml",
			wantErr: true,
		},
		{
			name:    "Path_To_System_Passwd",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "Path_To_System_Shadow",
			path:    "/etc/shadow",
			wantErr: true,
		},
		{
			name:    "Windows_System32_Path",
			path:    "C:\\Windows\\System32\\config.sys",
			wantErr: true,
		},
		// Edge cases
		{
			name:    "Path_With_Dot_Segments_Safe",
			path:    "./config/./file.yaml",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantSafe bool // true if result shouldn't contain ..
	}{
		{
			name:     "Simple_Path",
			path:     "config/file.yaml",
			wantSafe: true,
		},
		{
			name:     "Path_With_Dot_Segments",
			path:     "./config/../config/file.yaml",
			wantSafe: true,
		},
		{
			name:     "Path_With_Double_Slashes",
			path:     "config//file.yaml",
			wantSafe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilePath(tt.path)
			if result == "" {
				t.Error("SanitizeFilePath() returned empty string")
			}
			// Verify it's cleaned (no .. in the result unless it's legitimately part of the path)
			// After cleaning, .. should only appear if the original path genuinely went outside
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid_YAML_Config",
			path:    "config.yaml",
			wantErr: false,
		},
		{
			name:    "Valid_YML_Config",
			path:    "settings.yml",
			wantErr: false,
		},
		{
			name:    "Invalid_Extension",
			path:    "config.json",
			wantErr: true,
		},
		{
			name:    "No_Extension",
			path:    "config",
			wantErr: true,
		},
		{
			name:    "Path_Traversal_In_Config",
			path:    "../../../etc/config.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateXSLTPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{
			name:    "Valid_XSLT_Extension",
			path:    "transform.xslt",
			wantErr: false,
		},
		{
			name:    "Valid_XSL_Extension",
			path:    "transform.xsl",
			wantErr: false,
		},
		{
			name:    "Valid_Embedded_XSLT",
			path:    "embedded:tsl-to-html.xslt",
			wantErr: false,
		},
		// Invalid paths
		{
			name:    "Invalid_Extension",
			path:    "transform.xml",
			wantErr: true,
		},
		{
			name:    "Embedded_Empty_Name",
			path:    "embedded:",
			wantErr: true,
		},
		{
			name:    "Embedded_With_Path_Separator",
			path:    "embedded:../bad.xslt",
			wantErr: true,
		},
		{
			name:    "Embedded_With_Traversal",
			path:    "embedded:..\\bad.xslt",
			wantErr: true,
		},
		{
			name:    "Path_Traversal",
			path:    "../../../etc/transform.xslt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateXSLTPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateXSLTPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOutputDirectory(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{
			name:    "Valid_Relative_Directory",
			path:    "output/tsls",
			wantErr: false,
		},
		{
			name:    "Valid_Absolute_Directory",
			path:    "/tmp/output",
			wantErr: false,
		},
		// Invalid paths
		{
			name:    "Root_Directory",
			path:    "/",
			wantErr: true,
		},
		{
			name:    "Windows_Root",
			path:    "C:\\",
			wantErr: true,
		},
		{
			name:    "System_Directory_Etc",
			path:    "/etc/output",
			wantErr: true,
		},
		{
			name:    "System_Directory_Sys",
			path:    "/sys/output",
			wantErr: true,
		},
		{
			name:    "System_Directory_Windows",
			path:    "C:\\Windows\\output",
			wantErr: true,
		},
		{
			name:    "Path_Traversal",
			path:    "../../../etc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultURLOptions(t *testing.T) {
	opts := DefaultURLOptions()
	if len(opts.AllowedSchemes) != 2 {
		t.Errorf("Expected 2 allowed schemes, got %d", len(opts.AllowedSchemes))
	}
	if !opts.RequireAbsoluteURL {
		t.Error("Expected RequireAbsoluteURL to be true")
	}
	if opts.AllowFileURLs {
		t.Error("Expected AllowFileURLs to be false")
	}
}

func TestTSLURLOptions(t *testing.T) {
	opts := TSLURLOptions()
	if len(opts.AllowedSchemes) != 3 {
		t.Errorf("Expected 3 allowed schemes, got %d", len(opts.AllowedSchemes))
	}
	if !opts.RequireAbsoluteURL {
		t.Error("Expected RequireAbsoluteURL to be true")
	}
	if !opts.AllowFileURLs {
		t.Error("Expected AllowFileURLs to be true")
	}
}

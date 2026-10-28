package db

import (
	"testing"
)

// TestTOsInfo_TableName tests the TableName method
func TestTOsInfo_TableName(t *testing.T) {
	osInfo := &TOsInfo{}
	expected := "t_os_info"
	if osInfo.TableName() != expected {
		t.Errorf("TableName() = %v, want %v", osInfo.TableName(), expected)
	}
}

// TestTOsInfo_QueryParameter tests the QueryParameter method
func TestTOsInfo_QueryParameter(t *testing.T) {
	osInfo := &TOsInfo{
		ID: 42,
	}

	result := osInfo.QueryParameter()
	expected := "id=eq.42"

	if result != expected {
		t.Errorf("QueryParameter() = %v, want %v", result, expected)
	}
}

// TestTOsInfo_Parse tests parsing JSON array
func TestTOsInfo_Parse(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
		expected  int
	}{
		{
			name: "Valid single item",
			input: []interface{}{
				map[string]interface{}{
					"id":         float64(1),
					"login_user": "ubuntu",
				},
			},
			expectErr: false,
			expected:  1,
		},
		{
			name:      "Invalid input type",
			input:     "not an array",
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osInfo := &TOsInfo{}
			result, err := osInfo.Parse(tt.input)
			
			if tt.expectErr && err == nil {
				t.Error("Parse() expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("Parse() result length = %v, want %v", len(result), tt.expected)
			}
		})
	}
}

// TestTableNameTOsInfo tests the constant
func TestTableNameTOsInfo(t *testing.T) {
	expected := "t_os_info"
	if TableNameTOsInfo != expected {
		t.Errorf("TableNameTOsInfo = %v, want %v", TableNameTOsInfo, expected)
	}
}

package api

import (
	"os"
	"testing"
)

func TestCustomValidator(t *testing.T) {
	v := NewCustomValidator()

	type TestStruct struct {
		GitURL     string `validate:"giturl"`
		RepoPath   string `validate:"path"`
		Kubeconfig string `validate:"kubeconfigfile"`
	}

	// Create a fake kubeconfig for the test
	tmpFile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	kubeconfig := `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost:6443
  name: test-cluster
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
`
	_ = os.WriteFile(tmpFile.Name(), []byte(kubeconfig), 0644)
	_ = tmpFile.Close()

	tests := []struct {
		name    string
		data    TestStruct
		wantErr bool
	}{
		{
			name: "Valid Data",
			data: TestStruct{
				GitURL:     "https://github.com/user/repo.git",
				RepoPath:   "apps/my-app",
				Kubeconfig: tmpFile.Name(),
			},
			wantErr: false,
		},
		{
			name: "Invalid GitURL",
			data: TestStruct{
				GitURL:     "invalid-url",
				RepoPath:   "apps/my-app",
				Kubeconfig: tmpFile.Name(),
			},
			wantErr: true,
		},
	}

	// Create a fake kubeconfig for the test
	_ = testing.T{} // just to keep imports clean if I had more

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

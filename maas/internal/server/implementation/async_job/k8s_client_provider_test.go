package async_job

import (
	"os"
	"testing"
)

func TestResolveNamespace(t *testing.T) {
	t.Run("explicit namespace", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "env-ns")
		if got := resolveNamespace("explicit-ns"); got != "explicit-ns" {
			t.Fatalf("expected explicit-ns, got %s", got)
		}
	})

	t.Run("from env", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "env-ns")
		if got := resolveNamespace(""); got != "env-ns" {
			t.Fatalf("expected env-ns, got %s", got)
		}
	})

	t.Run("default", func(t *testing.T) {
		_ = os.Unsetenv("POD_NAMESPACE")
		if got := resolveNamespace(""); got != "default" {
			t.Fatalf("expected default, got %s", got)
		}
	})
}

func TestInClusterK8sClientProvider_GetClientset_Error(t *testing.T) {
	provider := &InClusterK8sClientProvider{}
	_, err := provider.GetClientset()
	if err == nil {
		t.Fatalf("expected error outside cluster environment")
	}
}

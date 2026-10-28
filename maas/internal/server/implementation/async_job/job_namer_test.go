package async_job

import (
	"regexp"
	"testing"
)

func TestGenerateK8sJobName_Format(t *testing.T) {
	name := GenerateK8sJobName(JobTypeMachineRegister, "AbC_12", "req-1234")
	pattern := regexp.MustCompile(`^pi-maas-mr-[a-z0-9-]+-[0-9]+-[a-f0-9]{8}$`)
	if !pattern.MatchString(name) {
		t.Fatalf("generated name format invalid: %s", name)
	}
}

func TestGenerateK8sJobName_SanitizeSystemID(t *testing.T) {
	name := GenerateK8sJobName(JobTypeVMCompose, "**SYSTEM__ID??", "req-abc")
	if regexp.MustCompile(`[A-Z_*?]`).MatchString(name) {
		t.Fatalf("generated name includes unsupported characters: %s", name)
	}
}

func TestGenerateK8sJobName_HashStabilityByRequestID(t *testing.T) {
	name1 := GenerateK8sJobName(JobTypeKubeadmJoin, "sys-1", "same-request")
	name2 := GenerateK8sJobName(JobTypeKubeadmJoin, "sys-1", "same-request")

	hash1 := name1[len(name1)-8:]
	hash2 := name2[len(name2)-8:]
	if hash1 != hash2 {
		t.Fatalf("expected same hash suffix for same request ID, got %s and %s", hash1, hash2)
	}
}

func TestGenerateK8sJobName_UnknownTypeAndEmptySystemID(t *testing.T) {
	name := GenerateK8sJobName(JobType("unknown"), "___", "")
	pattern := regexp.MustCompile(`^pi-maas-na-na-[0-9]+-[a-f0-9]{8}$`)
	if !pattern.MatchString(name) {
		t.Fatalf("unexpected generated name: %s", name)
	}
}

func TestGenerateK8sJobName_SystemIDTruncation(t *testing.T) {
	name := GenerateK8sJobName(JobTypeMachineRegister, "abcdefghijklmnopqrstuvwxyz", "req-trunc")
	pattern := regexp.MustCompile(`^pi-maas-mr-abcdefghijklmnopqrst-[0-9]+-[a-f0-9]{8}$`)
	if !pattern.MatchString(name) {
		t.Fatalf("expected truncated system-id token, got: %s", name)
	}
}

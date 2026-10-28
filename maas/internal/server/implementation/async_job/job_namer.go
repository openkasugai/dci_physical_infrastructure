package async_job

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var k8sNameSanitizer = regexp.MustCompile(`[^a-z0-9-]`)

func sanitizeNameToken(raw string) string {
	n := strings.ToLower(raw)
	n = strings.ReplaceAll(n, "_", "-")
	n = k8sNameSanitizer.ReplaceAllString(n, "-")
	n = strings.Trim(n, "-")
	if n == "" {
		return "na"
	}
	return n
}

func shortHash(raw string) string {
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])[:8]
}

func jobTypeToken(jobType JobType) string {
	switch jobType {
	case JobTypeMachineRegister:
		return "mr"
	case JobTypeVMCompose:
		return "vc"
	case JobTypeKubeadmJoin:
		return "kj"
	default:
		return "na"
	}
}

// GenerateK8sJobName creates a DNS-1123 compatible job name.
// Format: pi-maas-{jt}-{sid}-{ts}-{h8}
func GenerateK8sJobName(jobType JobType, systemID string, requestID string) string {
	sid := sanitizeNameToken(systemID)
	if len(sid) > 20 {
		sid = sid[:20]
	}

	ts := time.Now().UTC().Unix()
	hashBase := requestID
	if hashBase == "" {
		hashBase = fmt.Sprintf("%s-%s-%d", jobType, sid, ts)
	}
	h8 := shortHash(hashBase)

	return fmt.Sprintf("pi-maas-%s-%s-%d-%s", jobTypeToken(jobType), sid, ts, h8)
}

package async_job

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type providerStub struct {
	clientset *kubernetes.Clientset
	err       error
}

func (p *providerStub) GetClientset() (*kubernetes.Clientset, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.clientset, nil
}

type mockJobAPI struct {
	mu sync.Mutex

	namespace string

	jobs map[string]batchv1.Job

	createCalls int
	listCalls   int
	deleteCalls int
	lastCreateBody string

	forceCreateError bool
	forceListError   bool
	forceDeleteError bool
	createConflict   bool
	deleteNotFound   bool
}

func newMockJobAPI(namespace string) *mockJobAPI {
	return &mockJobAPI{
		namespace: namespace,
		jobs:      map[string]batchv1.Job{},
	}
}

func parseSelector(raw string) map[string]string {
	out := map[string]string{}
	if raw == "" {
		return out
	}
	for _, kv := range strings.Split(raw, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func labelsMatch(labels map[string]string, selector map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func (m *mockJobAPI) handler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	base := "/apis/batch/v1/namespaces/" + m.namespace + "/jobs"
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == base && r.Method == http.MethodGet {
		m.listCalls++
		if m.forceListError {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure"}`))
			return
		}
		selRaw, _ := url.QueryUnescape(r.URL.Query().Get("labelSelector"))
		sel := parseSelector(selRaw)
		items := []batchv1.Job{}
		for _, j := range m.jobs {
			if labelsMatch(j.Labels, sel) {
				items = append(items, j)
			}
		}
		resp := batchv1.JobList{Items: items}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if r.URL.Path == base && r.Method == http.MethodPost {
		m.createCalls++
		if m.createConflict {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"AlreadyExists","code":409}`))
			return
		}
		if m.forceCreateError {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":500}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		m.lastCreateBody = string(body)
		var job batchv1.Job
		_ = json.Unmarshal(body, &job)
		m.jobs[job.Name] = job
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(job)
		return
	}

	if strings.HasPrefix(r.URL.Path, base+"/") && r.Method == http.MethodDelete {
		m.deleteCalls++
		if m.forceDeleteError {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":500}`))
			return
		}
		name := strings.TrimPrefix(r.URL.Path, base+"/")
		if _, ok := m.jobs[name]; !ok && m.deleteNotFound {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404}`))
			return
		}
		delete(m.jobs, name)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Success"}`))
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":404}`))
}

func newClientsetForMockServer(t *testing.T, h http.HandlerFunc) (*kubernetes.Clientset, func()) {
	t.Helper()
	server := httptest.NewServer(h)
	cfg := &rest.Config{
		Host:    server.URL,
		APIPath: "/",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &schema.GroupVersion{Group: "", Version: "v1"},
		},
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		server.Close()
		t.Fatalf("failed to create clientset: %v", err)
	}
	return cs, server.Close
}

func TestNewK8sJobStore_ProviderError(t *testing.T) {
	_, err := NewK8sJobStore(K8sJobStoreOptions{
		ClientProvider: &providerStub{err: fmt.Errorf("boom")},
	})
	if err == nil {
		t.Fatalf("expected provider error")
	}
}

func TestNewK8sJobStore_DefaultAndNamespace(t *testing.T) {
	m := newMockJobAPI("env-ns")
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	t.Setenv("POD_NAMESPACE", "env-ns")

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		BackoffLimit:          -1,
		TTLSecondsAfterFinish: 0,
		ActiveDeadlineSeconds: 0,
		ClientProvider:        &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.namespace != "env-ns" {
		t.Fatalf("expected namespace env-ns, got %s", store.namespace)
	}
	if store.backoffLimit != 0 {
		t.Fatalf("expected normalized backoff 0, got %d", store.backoffLimit)
	}
	if store.ttlSecondsAfterFinish != 600 {
		t.Fatalf("expected default ttl 600, got %d", store.ttlSecondsAfterFinish)
	}
	if store.activeDeadlineSeconds != int64(2*time.Hour/time.Second) {
		t.Fatalf("expected default deadline 7200, got %d", store.activeDeadlineSeconds)
	}
}

func TestNewK8sJobStore_ExplicitNamespaceOverridesEnv(t *testing.T) {
	m := newMockJobAPI("explicit-ns")
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	t.Setenv("POD_NAMESPACE", "env-ns")

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		Namespace:      "explicit-ns",
		WorkerImage:    "worker:latest",
		ClientProvider: &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.namespace != "explicit-ns" {
		t.Fatalf("expected explicit namespace, got %s", store.namespace)
	}
}

func TestK8sJobStore_Register_EmptyWorkerImage(t *testing.T) {
	store := &K8sJobStore{}
	err := store.Register("sys-1", JobInfo{Type: JobTypeMachineRegister})
	if err == nil || !strings.Contains(err.Error(), "worker image is empty") {
		t.Fatalf("expected worker image error, got %v", err)
	}
}

func TestK8sJobStore_Register_DuplicateByRequestIDAndActive(t *testing.T) {
	m := newMockJobAPI("ns")
	m.jobs["job-succeeded"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			"dci.io/system-id":  "sys-1",
			"dci.io/job-type":   string(JobTypeMachineRegister),
			"dci.io/request-id": "req-1",
		}},
		Status: batchv1.JobStatus{Succeeded: 1},
	}
	m.jobs["job-active"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			"dci.io/system-id": "sys-2",
			"dci.io/job-type":  string(JobTypeVMCompose),
		}},
		Status: batchv1.JobStatus{Active: 1},
	}

	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		Namespace:      "ns",
		WorkerImage:    "worker:latest",
		ClientProvider: &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Register("sys-1", JobInfo{Type: JobTypeMachineRegister, RequestID: "req-1"})
	if err != nil {
		t.Fatalf("expected no error for request-id duplicate, got %v", err)
	}

	err = store.Register("sys-2", JobInfo{Type: JobTypeVMCompose})
	if err != nil {
		t.Fatalf("expected no error for active duplicate, got %v", err)
	}

	if m.createCalls != 0 {
		t.Fatalf("expected no create call, got %d", m.createCalls)
	}
}

func TestK8sJobStore_Register_CreateSuccessAndConflict(t *testing.T) {
	m := newMockJobAPI("ns")
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	t.Setenv("ANSIBLE_PATH", "/opt/ansible")
	t.Setenv("MAAS_API_RETRY_COUNT", "10")
	t.Setenv("MAAS_API_RETRY_INTERVAL_SECONDS", "5")

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		Namespace:             "ns",
		WorkerImage:           "worker:latest",
		ServiceAccountName:    "sa-test",
		BackoffLimit:          2,
		TTLSecondsAfterFinish: 100,
		ActiveDeadlineSeconds: 200,
		ClientProvider:        &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Register("sys-1", JobInfo{Type: JobTypeMachineRegister, RequestID: "req-a", TraceID: "trace-a", Payload: "{}"})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if m.createCalls != 1 {
		t.Fatalf("expected create call count 1, got %d", m.createCalls)
	}

	if len(m.jobs) == 0 {
		t.Fatalf("expected created job to exist")
	}
	if !regexp.MustCompile(`ANSIBLE_PATH[\s\x00-\x1F]*/opt/ansible`).MatchString(m.lastCreateBody) {
		t.Fatalf("expected ANSIBLE_PATH to be propagated in create request body")
	}
	if !regexp.MustCompile(`MAAS_API_RETRY_COUNT[\s\x00-\x1F]*10`).MatchString(m.lastCreateBody) {
		t.Fatalf("expected MAAS_API_RETRY_COUNT to be propagated in create request body")
	}
	if !regexp.MustCompile(`MAAS_API_RETRY_INTERVAL_SECONDS[\s\x00-\x1F]*5`).MatchString(m.lastCreateBody) {
		t.Fatalf("expected MAAS_API_RETRY_INTERVAL_SECONDS to be propagated in create request body")
	}

	m.createConflict = true
	err = store.Register("sys-3", JobInfo{Type: JobTypeKubeadmJoin, RequestID: "req-b"})
	if err != nil {
		t.Fatalf("expected nil for already exists conflict, got %v", err)
	}
}

func TestK8sJobStore_Register_CreateError(t *testing.T) {
	m := newMockJobAPI("ns")
	m.forceCreateError = true
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		Namespace:      "ns",
		WorkerImage:    "worker:latest",
		ClientProvider: &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Register("sys-1", JobInfo{Type: JobTypeMachineRegister})
	if err == nil || !strings.Contains(err.Error(), "failed to create k8s job") {
		t.Fatalf("expected create error, got %v", err)
	}
}

func TestK8sJobStore_Cancel_HasJob_ListJobs(t *testing.T) {
	m := newMockJobAPI("ns")
	m.jobs["active-job"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "active-job",
			CreationTimestamp: metav1.NewTime(time.Unix(1000, 0)),
			Labels: map[string]string{
				"dci.io/system-id":  "sys-1",
				"dci.io/job-type":   string(JobTypeMachineRegister),
				"dci.io/request-id": "req-1",
				"dci.io/trace-id":   "trace-1",
			},
		},
		Status: batchv1.JobStatus{Active: 1},
	}
	m.jobs["inactive-job"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "inactive-job",
			Labels: map[string]string{"dci.io/system-id": "sys-1", "dci.io/job-type": string(JobTypeVMCompose)},
		},
		Status: batchv1.JobStatus{Active: 0},
	}

	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		Namespace:      "ns",
		WorkerImage:    "worker:latest",
		ClientProvider: &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !store.HasJob("sys-1") {
		t.Fatalf("expected HasJob true")
	}
	jobs := store.ListJobs("sys-1")
	if len(jobs) != 2 {
		t.Fatalf("expected 2 list jobs, got %d", len(jobs))
	}
	if jobs[0].SystemID != "sys-1" {
		t.Fatalf("unexpected system ID in list job: %+v", jobs[0])
	}

	if err := store.Cancel("sys-1"); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if m.deleteCalls != 1 {
		t.Fatalf("expected delete only active job once, got %d", m.deleteCalls)
	}
}

func TestK8sJobStore_HasJob_FalseWhenNoActive(t *testing.T) {
	m := newMockJobAPI("ns")
	m.jobs["done-job"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "done-job", Labels: map[string]string{"dci.io/system-id": "sys-1"}},
		Status:     batchv1.JobStatus{Active: 0, Succeeded: 1},
	}
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	store, err := NewK8sJobStore(K8sJobStoreOptions{Namespace: "ns", WorkerImage: "worker", ClientProvider: &providerStub{clientset: cs}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.HasJob("sys-1") {
		t.Fatalf("expected HasJob false when no active jobs")
	}
}

func TestK8sJobStore_CancelAndHasJob_ErrorPaths(t *testing.T) {
	m := newMockJobAPI("ns")
	m.forceListError = true
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	store, err := NewK8sJobStore(K8sJobStoreOptions{
		Namespace:      "ns",
		WorkerImage:    "worker:latest",
		ClientProvider: &providerStub{clientset: cs},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.HasJob("sys-1") {
		t.Fatalf("expected HasJob false on list error")
	}
	if got := store.ListJobs("sys-1"); len(got) != 0 {
		t.Fatalf("expected empty list on list error")
	}
	if err := store.Cancel("sys-1"); err == nil {
		t.Fatalf("expected cancel list error")
	}
}

func TestK8sJobStore_Cancel_DeleteErrors(t *testing.T) {
	t.Run("ignore not found", func(t *testing.T) {
		m := newMockJobAPI("ns")
		m.jobs["active-job"] = batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "active-job", Labels: map[string]string{"dci.io/system-id": "sys-1"}},
			Status:     batchv1.JobStatus{Active: 1},
		}
		m.deleteNotFound = true
		cs, closeFn := newClientsetForMockServer(t, m.handler)
		defer closeFn()
		store, err := NewK8sJobStore(K8sJobStoreOptions{Namespace: "ns", WorkerImage: "worker", ClientProvider: &providerStub{clientset: cs}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := store.Cancel("sys-1"); err != nil {
			t.Fatalf("expected nil when delete returns not found, got %v", err)
		}
	})

	t.Run("delete failure", func(t *testing.T) {
		m := newMockJobAPI("ns")
		m.jobs["active-job"] = batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "active-job", Labels: map[string]string{"dci.io/system-id": "sys-1"}},
			Status:     batchv1.JobStatus{Active: 1},
		}
		m.forceDeleteError = true
		cs, closeFn := newClientsetForMockServer(t, m.handler)
		defer closeFn()
		store, err := NewK8sJobStore(K8sJobStoreOptions{Namespace: "ns", WorkerImage: "worker", ClientProvider: &providerStub{clientset: cs}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := store.Cancel("sys-1"); err == nil {
			t.Fatalf("expected delete failure error")
		}
	})
}

func TestK8sJobStore_HelperMethods_ErrorAndPositive(t *testing.T) {
	m := newMockJobAPI("ns")
	m.jobs["active"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"dci.io/system-id": "sys-1", "dci.io/job-type": string(JobTypeMachineRegister)}},
		Status:     batchv1.JobStatus{Active: 1},
	}
	m.jobs["succeeded"] = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"dci.io/system-id": "sys-1", "dci.io/job-type": string(JobTypeMachineRegister), "dci.io/request-id": "req-1"}},
		Status:     batchv1.JobStatus{Succeeded: 1},
	}
	cs, closeFn := newClientsetForMockServer(t, m.handler)
	defer closeFn()

	store, err := NewK8sJobStore(K8sJobStoreOptions{Namespace: "ns", WorkerImage: "worker", ClientProvider: &providerStub{clientset: cs}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	active, err := store.hasActiveJob("sys-1", JobTypeMachineRegister)
	if err != nil || !active {
		t.Fatalf("expected active true, err=%v", err)
	}
	dup, err := store.hasActiveOrSucceededJobByRequestID("sys-1", JobTypeMachineRegister, "req-1")
	if err != nil || !dup {
		t.Fatalf("expected duplicate true, err=%v", err)
	}

	m.forceListError = true
	if _, err := store.hasActiveJob("sys-1", JobTypeMachineRegister); err == nil {
		t.Fatalf("expected error from hasActiveJob list failure")
	}
	if _, err := store.hasActiveOrSucceededJobByRequestID("sys-1", JobTypeMachineRegister, "req-1"); err == nil {
		t.Fatalf("expected error from hasActiveOrSucceededJobByRequestID list failure")
	}
}

func TestK8sJobStore_Register_ListErrorBranches(t *testing.T) {
	t.Run("request-id duplication check list error", func(t *testing.T) {
		m := newMockJobAPI("ns")
		m.forceListError = true
		cs, closeFn := newClientsetForMockServer(t, m.handler)
		defer closeFn()

		store, err := NewK8sJobStore(K8sJobStoreOptions{Namespace: "ns", WorkerImage: "worker", ClientProvider: &providerStub{clientset: cs}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.Register("sys-1", JobInfo{Type: JobTypeMachineRegister, RequestID: "req-1"})
		if err == nil {
			t.Fatalf("expected error from request-id list failure")
		}
	})

	t.Run("active duplication check list error", func(t *testing.T) {
		m := newMockJobAPI("ns")
		m.forceListError = true
		cs, closeFn := newClientsetForMockServer(t, m.handler)
		defer closeFn()

		store, err := NewK8sJobStore(K8sJobStoreOptions{Namespace: "ns", WorkerImage: "worker", ClientProvider: &providerStub{clientset: cs}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.Register("sys-1", JobInfo{Type: JobTypeMachineRegister})
		if err == nil {
			t.Fatalf("expected error from active-check list failure")
		}
	})
}

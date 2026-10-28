package async_job

import (
	"errors"
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type testStore struct {
	registerCalls   int
	deregisterCalls int
	cancelCalls     int
	lastSystemID    string
	lastJobType     JobType
	lastInfo        JobInfo
	hasJob          bool
	cancelErr       error
}

func (s *testStore) Register(systemID string, info JobInfo) error {
	s.registerCalls++
	s.lastSystemID = systemID
	s.lastInfo = info
	return nil
}

func (s *testStore) Deregister(systemID string, jobType JobType) {
	s.deregisterCalls++
	s.lastSystemID = systemID
	s.lastJobType = jobType
}

func (s *testStore) HasJob(systemID string) bool {
	s.lastSystemID = systemID
	return s.hasJob
}

func (s *testStore) ListJobs(systemID string) []JobInfo {
	s.lastSystemID = systemID
	return []JobInfo{{SystemID: systemID, Type: JobTypeMachineRegister}}
}

func (s *testStore) Cancel(systemID string) error {
	s.cancelCalls++
	s.lastSystemID = systemID
	return s.cancelErr
}

func TestInMemoryJobStore_BasicLifecycle(t *testing.T) {
	store := NewInMemoryJobStore()

	if store.HasJob("sys-1") {
		t.Fatalf("expected no job at start")
	}

	err := store.Register("sys-1", JobInfo{SystemID: "sys-1", Type: JobTypeMachineRegister})
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	err = store.Register("sys-1", JobInfo{SystemID: "sys-1", Type: JobTypeVMCompose})
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	if !store.HasJob("sys-1") {
		t.Fatalf("expected HasJob true after register")
	}

	jobs := store.ListJobs("sys-1")
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	store.Deregister("sys-1", JobTypeMachineRegister)
	jobs = store.ListJobs("sys-1")
	if len(jobs) != 1 || jobs[0].Type != JobTypeVMCompose {
		t.Fatalf("unexpected jobs after deregister: %+v", jobs)
	}

	if err := store.Cancel("sys-1"); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if store.HasJob("sys-1") {
		t.Fatalf("expected no job after cancel")
	}
}

func TestNoopJobStore_Behavior(t *testing.T) {
	store := &NoopJobStore{}
	if err := store.Register("sys", JobInfo{}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	store.Deregister("sys", JobTypeMachineRegister)
	if store.HasJob("sys") {
		t.Fatalf("expected no jobs in noop store")
	}
	if len(store.ListJobs("sys")) != 0 {
		t.Fatalf("expected empty list in noop store")
	}
	if err := store.Cancel("sys"); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
}

func TestJobManager_RegisterAndRegisterWithInfo(t *testing.T) {
	store := &testStore{}
	mgr := NewJobManager(store)

	mgr.Register("sys-1", JobTypeMachineRegister)
	if store.registerCalls != 1 {
		t.Fatalf("expected register call count 1, got %d", store.registerCalls)
	}
	if store.lastInfo.Type != JobTypeMachineRegister {
		t.Fatalf("unexpected job type: %s", store.lastInfo.Type)
	}
	if store.lastInfo.StartedAt.IsZero() {
		t.Fatalf("expected StartedAt to be set")
	}

	mgr.RegisterWithInfo("sys-2", JobTypeVMCompose, "req-1", "trace-1", "payload")
	if store.registerCalls != 2 {
		t.Fatalf("expected register call count 2, got %d", store.registerCalls)
	}
	if store.lastInfo.RequestID != "req-1" || store.lastInfo.TraceID != "trace-1" || store.lastInfo.Payload != "payload" {
		t.Fatalf("unexpected info fields: %+v", store.lastInfo)
	}
}

func TestJobManager_DeregisterAndCancel(t *testing.T) {
	store := &testStore{}
	mgr := NewJobManager(store)

	mgr.Deregister("sys-1", JobTypeKubeadmJoin)
	if store.deregisterCalls != 1 {
		t.Fatalf("expected deregister call count 1, got %d", store.deregisterCalls)
	}
	if store.lastSystemID != "sys-1" || store.lastJobType != JobTypeKubeadmJoin {
		t.Fatalf("unexpected deregister args: systemID=%s type=%s", store.lastSystemID, store.lastJobType)
	}

	store.cancelErr = errors.New("cancel failed")
	err := mgr.CancelJobs("sys-1")
	if err == nil {
		t.Fatalf("expected cancel error")
	}
	if store.cancelCalls != 1 {
		t.Fatalf("expected cancel call count 1, got %d", store.cancelCalls)
	}
}

func TestJobManager_HasProcessingJobAndNilGuards(t *testing.T) {
	store := &testStore{hasJob: true}
	mgr := NewJobManager(store)

	if !mgr.HasProcessingJob("sys-1") {
		t.Fatalf("expected HasProcessingJob true")
	}

	var nilMgr *JobManager
	if nilMgr.HasProcessingJob("sys") {
		t.Fatalf("expected false for nil manager")
	}
	if err := nilMgr.CancelJobs("sys"); err != nil {
		t.Fatalf("expected nil error for nil manager, got %v", err)
	}
	if nilMgr.IsK8sBacked() {
		t.Fatalf("expected false for nil manager")
	}
}

func TestJobManager_IsK8sBacked(t *testing.T) {
	k8sMgr := NewJobManager(&K8sJobStore{})
	if !k8sMgr.IsK8sBacked() {
		t.Fatalf("expected k8s-backed manager")
	}

	memMgr := NewJobManager(NewInMemoryJobStore())
	if memMgr.IsK8sBacked() {
		t.Fatalf("expected non-k8s manager")
	}
}

func TestNewInMemoryJobManager_Singleton(t *testing.T) {
	m1 := NewInMemoryJobManager()
	m2 := NewInMemoryJobManager()
	if m1 != m2 {
		t.Fatalf("expected singleton manager instance")
	}
}

func TestNewK8sJobManager_FallbackAndSuccess(t *testing.T) {
	logger := klog.NewKlogr()

	fallback := NewK8sJobManager(logger, K8sJobStoreOptions{
		ClientProvider: &providerStub{err: errors.New("provider fail")},
	})
	if fallback == nil {
		t.Fatalf("expected manager instance")
	}
	if fallback.IsK8sBacked() {
		t.Fatalf("expected fallback manager to be non-k8s backed")
	}

	ok := NewK8sJobManager(logger, K8sJobStoreOptions{
		Namespace:      "ns",
		WorkerImage:    "worker:latest",
		ClientProvider: &providerStub{clientset: &kubernetes.Clientset{}},
	})
	if ok == nil {
		t.Fatalf("expected manager instance")
	}
	if !ok.IsK8sBacked() {
		t.Fatalf("expected k8s-backed manager on successful init")
	}
}

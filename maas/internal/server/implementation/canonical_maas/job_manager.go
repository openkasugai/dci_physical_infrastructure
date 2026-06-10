// Copyright 2026 NTT, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package canonical_maas

import (
	"sync"
	"time"
)

// JobType identifies the type of async job.
type JobType string

const (
	JobTypeMachineRegister JobType = "machine_register"
	JobTypeVMCompose       JobType = "vm_compose"
	JobTypeKubeadmJoin     JobType = "kubeadm_join"
)

// JobInfo holds metadata for a single in-progress job.
type JobInfo struct {
	SystemID  string
	Type      JobType
	StartedAt time.Time
	// K8sJobName string  // reserved for future k8s Job externalisation
}

// JobStore is the persistence abstraction for job state.
// Current implementation: InMemoryJobStore.
// Future implementation: K8sJobStore (reading k8s Job / CRD status).
type JobStore interface {
	// Register adds a job entry for the given systemID.
	// Multiple jobs per systemID are allowed.
	Register(systemID string, info JobInfo) error

	// Deregister removes one job of the given type for the systemID.
	Deregister(systemID string, jobType JobType)

	// HasJob returns true if at least one job exists for the systemID.
	HasJob(systemID string) bool

	// ListJobs returns all jobs registered for the systemID.
	ListJobs(systemID string) []JobInfo
}

/*
 * InMemoryJobStore
 */

// InMemoryJobStore is the in-process, non-persistent implementation of JobStore.
type InMemoryJobStore struct {
	mu   sync.RWMutex
	jobs map[string][]JobInfo // key: systemID
}

// NewInMemoryJobStore creates a new InMemoryJobStore.
func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobs: make(map[string][]JobInfo),
	}
}

func (s *InMemoryJobStore) Register(systemID string, info JobInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[systemID] = append(s.jobs[systemID], info)
	return nil
}

func (s *InMemoryJobStore) Deregister(systemID string, jobType JobType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.jobs[systemID]
	for i, j := range list {
		if j.Type == jobType {
			s.jobs[systemID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(s.jobs[systemID]) == 0 {
		delete(s.jobs, systemID)
	}
}

func (s *InMemoryJobStore) HasJob(systemID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs[systemID]) > 0
}

func (s *InMemoryJobStore) ListJobs(systemID string) []JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.jobs[systemID]
	result := make([]JobInfo, len(src))
	copy(result, src)
	return result
}

/*
 * JobManager
 */

// JobManager orchestrates job lifecycle via a JobStore.
type JobManager struct {
	store JobStore
}

// NewJobManager creates a JobManager backed by the given JobStore.
func NewJobManager(store JobStore) *JobManager {
	return &JobManager{store: store}
}

// sharedJobManager is the process-wide singleton JobManager shared across all gRPC requests.
var sharedJobManager = NewJobManager(NewInMemoryJobStore())

// NewInMemoryJobManager returns the process-wide singleton JobManager backed by InMemoryJobStore.
// All gRPC handlers share the same instance so that jobs registered by one request
// (e.g. MachineRegister) are visible to subsequent requests (e.g. MachineShow).
func NewInMemoryJobManager() *JobManager {
	return sharedJobManager
}

// Register registers a new in-progress job for the given systemID.
func (j *JobManager) Register(systemID string, jobType JobType) {
	_ = j.store.Register(systemID, JobInfo{
		SystemID:  systemID,
		Type:      jobType,
		StartedAt: time.Now(),
	})
}

// Deregister removes the job of the given type for the systemID.
func (j *JobManager) Deregister(systemID string, jobType JobType) {
	j.store.Deregister(systemID, jobType)
}

// HasProcessingJob returns true if at least one job is in progress for the systemID.
func (j *JobManager) HasProcessingJob(systemID string) bool {
	if j == nil {
		return false
	}
	return j.store.HasJob(systemID)
}



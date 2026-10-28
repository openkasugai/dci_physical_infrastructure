package async_job

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sJobStoreOptions configures K8sJobStore behaviour.
type K8sJobStoreOptions struct {
	Namespace             string
	WorkerImage           string
	ServiceAccountName    string
	SshSecretName         string
	TlsSecretName         string
	TlsEnabled            bool
	TlsCertPath           string
	BackoffLimit          int32
	TTLSecondsAfterFinish int32
	ActiveDeadlineSeconds int64
	ClientProvider        K8sClientProvider
}

// K8sJobStore stores job state in Kubernetes Jobs.
type K8sJobStore struct {
	clientset             *kubernetes.Clientset
	namespace             string
	workerImage           string
	serviceAccountName    string
	sshSecretName         string
	tlsSecretName         string
	tlsEnabled            bool
	tlsCertPath           string
	backoffLimit          int32
	ttlSecondsAfterFinish int32
	activeDeadlineSeconds int64
}

// NewK8sJobStore initializes a K8sJobStore.
func NewK8sJobStore(options K8sJobStoreOptions) (*K8sJobStore, error) {
	provider := options.ClientProvider
	if provider == nil {
		provider = &InClusterK8sClientProvider{}
	}
	clientset, err := provider.GetClientset()
	if err != nil {
		return nil, err
	}

	namespace := resolveNamespace(options.Namespace)
	if options.BackoffLimit < 0 {
		options.BackoffLimit = 0
	}
	if options.TTLSecondsAfterFinish <= 0 {
		options.TTLSecondsAfterFinish = 600
	}
	if options.ActiveDeadlineSeconds <= 0 {
		options.ActiveDeadlineSeconds = int64(2 * time.Hour / time.Second)
	}

	return &K8sJobStore{
		clientset:             clientset,
		namespace:             namespace,
		workerImage:           options.WorkerImage,
		serviceAccountName:    options.ServiceAccountName,
		sshSecretName:         options.SshSecretName,
		tlsSecretName:         options.TlsSecretName,
		tlsEnabled:            options.TlsEnabled,
		tlsCertPath:           options.TlsCertPath,
		backoffLimit:          options.BackoffLimit,
		ttlSecondsAfterFinish: options.TTLSecondsAfterFinish,
		activeDeadlineSeconds: options.ActiveDeadlineSeconds,
	}, nil
}

func (s *K8sJobStore) Register(systemID string, info JobInfo) error {
	if s.workerImage == "" {
		return fmt.Errorf("k8s worker image is empty")
	}

	if info.RequestID != "" {
		exists, err := s.hasActiveOrSucceededJobByRequestID(systemID, info.Type, info.RequestID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
	}

	activeExists, err := s.hasActiveJob(systemID, info.Type)
	if err != nil {
		return err
	}
	if activeExists {
		return nil
	}

	jobName := GenerateK8sJobName(info.Type, systemID, info.RequestID)
	info.K8sJobName = jobName

	labels := map[string]string{
		"dci.io/system-id":  systemID,
		"dci.io/job-type":   string(info.Type),
		"dci.io/source":     "maas-api",
		"dci.io/job-name":   jobName,
		"dci.io/request-id": info.RequestID,
		"dci.io/trace-id":   info.TraceID,
	}

	envs := []corev1.EnvVar{
		{Name: "MAAS_RUN_MODE", Value: "worker"},
		{Name: "MAAS_JOB_TYPE", Value: string(info.Type)},
		{Name: "MAAS_SYSTEM_ID", Value: systemID},
		{Name: "MAAS_REQUEST_ID", Value: info.RequestID},
		{Name: "MAAS_TRACE_ID", Value: info.TraceID},
		{Name: "MAAS_JOB_PAYLOAD", Value: info.Payload},
		{Name: "POD_NAMESPACE", Value: s.namespace},
	}

	for _, key := range []string{
		"LOG_LEVEL",
		"MAAS_SERVER_PORT",
		"VM_HOST_DISK",
		"LXD_PORT",
		"ANSIBLE_PATH",
		"SSH_KEY",
		"TLS_ENABLE",
		"TLS_CERT_PATH",
		"MAAS_API_RETRY_COUNT",
		"MAAS_API_RETRY_INTERVAL_SECONDS",
		"PRODUCT_MAPPINGS",
		"K8S_JOB_WORKER_IMAGE",
		"K8S_JOB_SERVICE_ACCOUNT",
		"K8S_JOB_BACKOFF_LIMIT",
		"K8S_JOB_TTL_SECONDS",
		"K8S_JOB_ACTIVE_DEADLINE_SECONDS",
		"K8S_JOB_SSH_SECRET_NAME",
		"K8S_JOB_TLS_SECRET_NAME",
	} {
		if value := os.Getenv(key); value != "" {
			envs = append(envs, corev1.EnvVar{Name: key, Value: value})
		}
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	if s.sshSecretName != "" {
		mode := int32(0400)
		volumes = append(volumes, corev1.Volume{
			Name: "ssh-keys",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  s.sshSecretName,
					DefaultMode: &mode,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ssh-keys", MountPath: "/etc/ssh-keys", ReadOnly: true})
	}
	if s.tlsEnabled && s.tlsSecretName != "" && s.tlsCertPath != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: s.tlsSecretName},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "tls-certs", MountPath: s.tlsCertPath, ReadOnly: true})
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: s.namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &s.backoffLimit,
			TTLSecondsAfterFinished: &s.ttlSecondsAfterFinish,
			ActiveDeadlineSeconds:   &s.activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: s.serviceAccountName,
					Volumes:            volumes,
					Containers: []corev1.Container{
						{
							Name:  "maas-worker",
							Image: s.workerImage,
							Env:          envs,
							VolumeMounts: volumeMounts,
						},
					},
				},
			},
		},
	}

	_, err = s.clientset.BatchV1().Jobs(s.namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create k8s job: %w", err)
	}
	return nil
}

func (s *K8sJobStore) Deregister(systemID string, jobType JobType) {
	// no-op: lifecycle is controlled by Kubernetes Job status and TTL cleanup.
}

func (s *K8sJobStore) Cancel(systemID string) error {
	selector := fmt.Sprintf("dci.io/system-id=%s", systemID)
	jobs, err := s.clientset.BatchV1().Jobs(s.namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return fmt.Errorf("failed to list jobs for cancel: %w", err)
	}

	policy := metav1.DeletePropagationBackground
	for _, job := range jobs.Items {
		if job.Status.Active == 0 {
			continue
		}
		err = s.clientset.BatchV1().Jobs(s.namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{PropagationPolicy: &policy})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete active job %s: %w", job.Name, err)
		}
	}

	return nil
}

func (s *K8sJobStore) HasJob(systemID string) bool {
	selector := fmt.Sprintf("dci.io/system-id=%s", systemID)
	jobs, err := s.clientset.BatchV1().Jobs(s.namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false
	}

	for _, job := range jobs.Items {
		if job.Status.Active > 0 {
			return true
		}
	}
	return false
}

func (s *K8sJobStore) ListJobs(systemID string) []JobInfo {
	selector := fmt.Sprintf("dci.io/system-id=%s", systemID)
	jobs, err := s.clientset.BatchV1().Jobs(s.namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return []JobInfo{}
	}

	result := make([]JobInfo, 0, len(jobs.Items))
	for _, item := range jobs.Items {
		jobType := JobType(item.Labels["dci.io/job-type"])
		startedAt := time.Now()
		if item.CreationTimestamp.Time.Unix() > 0 {
			startedAt = item.CreationTimestamp.Time
		}
		result = append(result, JobInfo{
			SystemID:   systemID,
			Type:       jobType,
			StartedAt:  startedAt,
			K8sJobName: item.Name,
			RequestID:  item.Labels["dci.io/request-id"],
			TraceID:    item.Labels["dci.io/trace-id"],
			Payload:    strconv.FormatInt(int64(item.Status.Active), 10),
		})
	}
	return result
}

func (s *K8sJobStore) hasActiveJob(systemID string, jobType JobType) (bool, error) {
	selector := fmt.Sprintf("dci.io/system-id=%s,dci.io/job-type=%s", systemID, string(jobType))
	jobs, err := s.clientset.BatchV1().Jobs(s.namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false, fmt.Errorf("failed to list jobs for duplication check: %w", err)
	}

	for _, job := range jobs.Items {
		if job.Status.Active > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (s *K8sJobStore) hasActiveOrSucceededJobByRequestID(systemID string, jobType JobType, requestID string) (bool, error) {
	selector := fmt.Sprintf("dci.io/system-id=%s,dci.io/job-type=%s,dci.io/request-id=%s", systemID, string(jobType), requestID)
	jobs, err := s.clientset.BatchV1().Jobs(s.namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return false, fmt.Errorf("failed to list jobs for request-id duplication check: %w", err)
	}

	for _, job := range jobs.Items {
		if job.Status.Active > 0 || job.Status.Succeeded > 0 {
			return true, nil
		}
	}

	return false, nil
}

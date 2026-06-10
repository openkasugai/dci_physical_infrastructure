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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	pb "maas_module/api/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	serverAddr = flag.String("server_addr", "maas-server:50053", "The server address in the format of host:port")
	method     = flag.String("method", "", "The gRPC method to call (MachineRegister, MachineDelete, OsDeploy, OsRelease, VmCompose, VmDelete, MachineList, MachineShow, Cancel, NetworkUpdate, PowerOn, PowerOff, KubeadmReset, KubeadmJoin)")
	jsonFile   = flag.String("json", "", "Path to the JSON configuration file")
	useTLS     = flag.Bool("tls", true, "Enable TLS connection (default: true)")
	secretName = flag.String("secret", "maas-client-tls", "Kubernetes secret name for TLS certificates")
	secretNS   = flag.String("namespace", "default", "Kubernetes namespace")
	kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig file (default: $HOME/.kube/config)")
)

// getTLSConfigFromSecret retrieves TLS configuration from Kubernetes secret
func getTLSConfigFromSecret() (*tls.Config, error) {
	// Set kubeconfig path
	kubeconfigPath := *kubeconfig
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		kubeconfigPath = homeDir + "/.kube/config"
	}

	// Create Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	// Get secret
	secret, err := clientset.CoreV1().Secrets(*secretNS).Get(
		context.Background(),
		*secretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %v", *secretNS, *secretName, err)
	}

	// Get CA certificate data
	caCertData, ok := secret.Data["ca.crt"]
	if !ok {
		return nil, fmt.Errorf("ca.crt not found in secret")
	}

	// Create CA certificate pool to verify server certificate
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertData) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Get client certificate and key for mTLS
	clientCertData, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, fmt.Errorf("tls.crt not found in secret")
	}
	clientKeyData, ok := secret.Data["tls.key"]
	if !ok {
		return nil, fmt.Errorf("tls.key not found in secret")
	}

	// Load client certificate
	clientCert, err := tls.X509KeyPair(clientCertData, clientKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientCert},
		ServerName:   "physical-infrastructure-maas-service.default.svc.cluster.local",
	}

	return tlsConfig, nil
}

func main() {
	flag.Parse()

	if *method == "" || *jsonFile == "" {
		fmt.Println("Usage: go run client.go -server_addr <server_address> -method <grpc_method> -json <json_file> [-tls] [-secret <secret_name>] [-namespace <namespace>] [-kubeconfig <path>]")
		os.Exit(1)
	}

	var conn *grpc.ClientConn
	var err error

	if *useTLS {
		// Get TLS configuration from Kubernetes secret
		tlsConfig, tlsErr := getTLSConfigFromSecret()
		if tlsErr != nil {
			log.Fatalf("failed to get TLS config from secret: %v", tlsErr)
		}

		creds := credentials.NewTLS(tlsConfig)

		// Establish gRPC connection with TLS
		conn, err = grpc.Dial(*serverAddr, grpc.WithTransportCredentials(creds)) //nolint:staticcheck
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
	} else {
		// Establish gRPC connection without TLS
		conn, err = grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:staticcheck
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
	}
	defer func() { _ = conn.Close() }()

	c := pb.NewMaasClient(conn)

	jsonData, err := os.ReadFile(*jsonFile)
	if err != nil {
		log.Fatalf("failed to read JSON file: %v", err)
	}

	switch *method {
	case "MachineRegister":
		handleMachineRegister(c, jsonData)
	case "MachineDelete":
		handleMachineDelete(c, jsonData)
	case "OsDeploy":
		handleOsDeploy(c, jsonData)
	case "OsRelease":
		handleOsRelease(c, jsonData)
	case "VmCompose":
		handleVmCompose(c, jsonData)
	case "VmDelete":
		handleVmDelete(c, jsonData)
	case "MachineList":
		handleMachineList(c, jsonData)
	case "MachineShow":
		handleMachineShow(c, jsonData)
	case "Cancel":
		handleCancel(c, jsonData)
	case "NetworkUpdate":
		handleNetworkUpdate(c, jsonData)
	case "PowerOn":
		handlePowerOn(c, jsonData)
	case "PowerOff":
		handlePowerOff(c, jsonData)
	case "KubeadmReset":
		handleKubeadmReset(c, jsonData)
	case "KubeadmJoin":
		handleKubeadmJoin(c, jsonData)
	default:
		log.Fatalf("invalid method: %s", *method)
	}
}

func handleMachineRegister(c pb.MaasClient, jsonData []byte) {
	req := &pb.MachineRegisterRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineRegisterRequest: %v", err)
	}

	r, err := c.MachineRegister(context.Background(), req)
	if err != nil {
		log.Fatalf("could not register machine: %v", err)
	}
	log.Printf("MachineRegister Response: %s", protojson.Format(r))
}

func handleMachineDelete(c pb.MaasClient, jsonData []byte) {
	req := &pb.MachineDeleteRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineDeleteRequest: %v", err)
	}

	r, err := c.MachineDelete(context.Background(), req)
	if err != nil {
		log.Fatalf("could not delete machine: %v", err)
	}
	log.Printf("MachineDelete Response: %s", protojson.Format(r))
}

func handleOsDeploy(c pb.MaasClient, jsonData []byte) {
	req := &pb.OsDeployRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal OsDeployRequest: %v", err)
	}

	r, err := c.OsDeploy(context.Background(), req)
	if err != nil {
		log.Fatalf("could not deploy OS: %v", err)
	}
	log.Printf("OsDeploy Response: %s", protojson.Format(r))
}

func handleOsRelease(c pb.MaasClient, jsonData []byte) {
	req := &pb.OsReleaseRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal OsReleaseRequest: %v", err)
	}

	r, err := c.OsRelease(context.Background(), req)
	if err != nil {
		log.Fatalf("could not release OS: %v", err)
	}
	log.Printf("OsRelease Response: %s", protojson.Format(r))
}

func handleVmCompose(c pb.MaasClient, jsonData []byte) {
	req := &pb.VmComposeRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal VmComposeRequest: %v", err)
	}

	r, err := c.VmCompose(context.Background(), req)
	if err != nil {
		log.Fatalf("could not compose VM: %v", err)
	}
	log.Printf("VmCompose Response: %s", protojson.Format(r))
}

func handleVmDelete(c pb.MaasClient, jsonData []byte) {
	req := &pb.VmDeleteRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal VmDeleteRequest: %v", err)
	}

	r, err := c.VmDelete(context.Background(), req)
	if err != nil {
		log.Fatalf("could not delete VM: %v", err)
	}
	log.Printf("VmDelete Response: %s", protojson.Format(r))
}

func handleMachineList(c pb.MaasClient, jsonData []byte) {
	req := &pb.MachineListRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineListRequest: %v", err)
	}

	r, err := c.MachineList(context.Background(), req)
	if err != nil {
		log.Fatalf("could not list machines: %v", err)
	}
	log.Printf("MachineList Response: %s", protojson.Format(r))
}

func handleMachineShow(c pb.MaasClient, jsonData []byte) {
	req := &pb.MachineShowRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineShowRequest: %v", err)
	}

	r, err := c.MachineShow(context.Background(), req)
	if err != nil {
		log.Fatalf("could not show machine: %v", err)
	}
	log.Printf("MachineShow Response: %s", protojson.Format(r))
}

func handleCancel(c pb.MaasClient, jsonData []byte) {
	req := &pb.CancelRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineShowRequest: %v", err)
	}

	r, err := c.Cancel(context.Background(), req)
	if err != nil {
		log.Fatalf("could not show machine: %v", err)
	}
	log.Printf("MachineShow Response: %s", protojson.Format(r))
}

func handleNetworkUpdate(c pb.MaasClient, jsonData []byte) {
	req := &pb.NetworkUpdateRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal NetworkUpdateRequest: %v", err)
	}

	r, err := c.NetworkUpdate(context.Background(), req)
	if err != nil {
		log.Fatalf("could not update network: %v", err)
	}
	log.Printf("NetworkUpdate Response: %s", protojson.Format(r))
}

func handlePowerOn(c pb.MaasClient, jsonData []byte) {
	req := &pb.PowerOnRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal PowerOnRequest: %v", err)
	}

	r, err := c.PowerOn(context.Background(), req)
	if err != nil {
		log.Fatalf("could not power on machine: %v", err)
	}
	log.Printf("PowerOn Response: %s", protojson.Format(r))
}

func handlePowerOff(c pb.MaasClient, jsonData []byte) {
	req := &pb.PowerOffRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal PowerOffRequest: %v", err)
	}

	r, err := c.PowerOff(context.Background(), req)
	if err != nil {
		log.Fatalf("could not power off machine: %v", err)
	}
	log.Printf("PowerOff Response: %s", protojson.Format(r))
}

func handleKubeadmReset(c pb.MaasClient, jsonData []byte) {
	req := &pb.KubeadmResetRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal KubeadmResetRequest: %v", err)
	}

	r, err := c.KubeadmReset(context.Background(), req)
	if err != nil {
		log.Fatalf("could not kubeadm reset machine: %v", err)
	}
	log.Printf("KubeadmReset Response: %s", protojson.Format(r))
}

func handleKubeadmJoin(c pb.MaasClient, jsonData []byte) {
	req := &pb.KubeadmJoinRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal KubeadmJoinRequest: %v", err)
	}

	r, err := c.KubeadmJoin(context.Background(), req)
	if err != nil {
		log.Fatalf("could not kubeadm join machine: %v", err)
	}
	log.Printf("KubeadmJoin Response: %s", protojson.Format(r))
}

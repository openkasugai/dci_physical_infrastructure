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

	pb "cdi_module/api/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	serverAddr = flag.String("server_addr", "cdi-server:50051", "The server address in the format of host:port")
	method     = flag.String("method", "", "The gRPC method to call (MachineCreate, MachineDestroy, MachineShow, ResourceList, ResourceShow)")
	jsonFile   = flag.String("json", "", "Path to the JSON configuration file")
	useTLS     = flag.Bool("tls", true, "Enable TLS connection (default: true)")
	secretName = flag.String("secret", "cdi-client-tls", "Kubernetes secret name for TLS certificates")
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
		ServerName:   "physical-infrastructure-cdi-service.default.svc.cluster.local",
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
	defer conn.Close()

	c := pb.NewCdiClient(conn)

	jsonData, err := os.ReadFile(*jsonFile)
	if err != nil {
		log.Fatalf("failed to read JSON file: %v", err)
	}

	switch *method {
	case "MachineCreate":
		handleMachineCreate(c, jsonData)
	case "MachineDestroy":
		handleMachineDestroy(c, jsonData)
	case "MachineShow":
		handleMachineShow(c, jsonData)
	case "ResourceList":
		handleResourceList(c, jsonData)
	case "ResourceShow":
		handleResourceShow(c, jsonData)
	case "CardScaling":
		handleCardScaling(c, jsonData)
	default:
		log.Fatalf("invalid method: %s", *method)
	}
}

func handleMachineCreate(c pb.CdiClient, jsonData []byte) {
	req := &pb.MachineCreateRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineCreateRequest: %v", err)
	}

	r, err := c.MachineCreate(context.Background(), req)
	if err != nil {
		log.Fatalf("could not create machine: %v", err)
	}
	log.Printf("MachineCreate Response: %s", protojson.Format(r))
}

func handleMachineDestroy(c pb.CdiClient, jsonData []byte) {
	req := &pb.MachineDestroyRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal MachineDestroyRequest: %v", err)
	}

	r, err := c.MachineDestroy(context.Background(), req)
	if err != nil {
		log.Fatalf("could not destroy machine: %v", err)
	}
	log.Printf("MachineDestroy Response: %s", protojson.Format(r))
}

func handleMachineShow(c pb.CdiClient, jsonData []byte) {
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

func handleResourceList(c pb.CdiClient, jsonData []byte) {
	req := &pb.ResourceListRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal ResourceListRequest: %v", err)
	}

	r, err := c.ResourceList(context.Background(), req)
	if err != nil {
		log.Fatalf("could not list resources: %v", err)
	}
	log.Printf("ResourceList Response: %s", protojson.Format(r))
}

func handleResourceShow(c pb.CdiClient, jsonData []byte) {
	req := &pb.ResourceShowRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal ResourceShowRequest: %v", err)
	}

	r, err := c.ResourceShow(context.Background(), req)
	if err != nil {
		log.Fatalf("could not show resource: %v", err)
	}
	log.Printf("ResourceShow Response: %s", protojson.Format(r))
}

func handleCardScaling(c pb.CdiClient, jsonData []byte) {
	req := &pb.CardScalingRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal CardScalingRequest: %v", err)
	}

	r, err := c.CardScaling(context.Background(), req)
	if err != nil {
		log.Fatalf("could not scale card: %v", err)
	}
	log.Printf("CardScaling Response: %s", protojson.Format(r))
}

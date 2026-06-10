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

	pb "network_module/api/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	serverAddr = flag.String("server_addr", "network-server:50052", "The server address in the format of host:port")
	method     = flag.String("method", "", "The gRPC method to call (VlanAdd, VlanDelete, VswVlanAdd, VswVlanDelete)")
	jsonFile   = flag.String("json", "", "Path to the JSON configuration file")
	useTLS     = flag.Bool("tls", true, "Enable TLS connection (default: true)")
	secretName = flag.String("secret", "network-client-tls", "Kubernetes secret name for TLS certificates")
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
		ServerName:   "physical-infrastructure-network-service.default.svc.cluster.local",
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

	c := pb.NewNetworkClient(conn)

	jsonData, err := os.ReadFile(*jsonFile)
	if err != nil {
		log.Fatalf("failed to read JSON file: %v", err)
	}

	switch *method {
	case "VlanAdd":
		handleVlanAdd(c, jsonData)
	case "VlanDelete":
		handleVlanDelete(c, jsonData)
	case "VswVlanAdd":
		handleVswVlanAdd(c, jsonData)
	case "VswVlanDelete":
		handleVswVlanDelete(c, jsonData)
	default:
		log.Fatalf("invalid method: %s", *method)
	}
}

func handleVlanAdd(c pb.NetworkClient, jsonData []byte) {
	req := &pb.VlanAddRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal VlanAddRequest: %v", err)
	}

	r, err := c.VlanAdd(context.Background(), req)
	if err != nil {
		log.Fatalf("could not add VLAN: %v", err)
	}
	log.Printf("VlanAdd Response: %s", protojson.Format(r))
}

func handleVlanDelete(c pb.NetworkClient, jsonData []byte) {
	req := &pb.VlanDeleteRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal VlanDeleteRequest: %v", err)
	}

	r, err := c.VlanDelete(context.Background(), req)
	if err != nil {
		log.Fatalf("could not delete VLAN: %v", err)
	}
	log.Printf("VlanDelete Response: %s", protojson.Format(r))
}

func handleVswVlanAdd(c pb.NetworkClient, jsonData []byte) {
	req := &pb.VswVlanAddRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal VswVlanAddRequest: %v", err)
	}

	r, err := c.VswVlanAdd(context.Background(), req)
	if err != nil {
		log.Fatalf("could not add VSW VLAN: %v", err)
	}
	log.Printf("VswVlanAdd Response: %s", protojson.Format(r))
}

func handleVswVlanDelete(c pb.NetworkClient, jsonData []byte) {
	req := &pb.VswVlanDeleteRequest{}
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		log.Fatalf("failed to unmarshal VswVlanDeleteRequest: %v", err)
	}

	r, err := c.VswVlanDelete(context.Background(), req)
	if err != nil {
		log.Fatalf("could not delete VSW VLAN: %v", err)
	}
	log.Printf("VswVlanDelete Response: %s", protojson.Format(r))
}

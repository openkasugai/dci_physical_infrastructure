[Japanese](./README_ja.md) / [English](./README.md)

> This repository is an early public release of DCI2.0 / DHRC; bug reports are welcome, while feature requests and pull requests will be accepted after the official release planned for the end of September 2026.

# dci_physical_infrastructure

This repository provides physical infrastructure functionality.

For an overview of DCI2.0 / DHRC, the overall architecture, and repository structure, see [dci_common](https://github.com/openkasugai/dci_common).

## Table of Contents

- [Building the repository](#building-the-repository)
- [SSH Key Setup](#ssh-key-setup)
- [Deployment](#deployment)

## Building the repository

To build the repository, follow these steps:

1.  **Prerequisites:**
    *   Go (version 1.24.2)
    *   and, required helm, docer, make, buildah

2.  **Clone the repository:**

    ```bash
    git clone https://github.com/compsysg/dci_physical_infrastructure.git
    cd dci_physical_infrastructure
    ```

3.  **Build the repository using makefile:**

    ```bash
    make build-all IMG_TAG=x.x.x
    ```

## SSH Key Setup

Before deploying the infrastructure, you need to set up SSH keys that will be used by all services (CDI, MAAS, Network, Exporter, and Log) to connect to remote hosts.

1.  **Generate SSH Key Pair:**

    Generate an ed25519 SSH key pair on the deployment host:

    ```bash
    ssh-keygen -t ed25519 -f ./id_ed25519 -N ""
    ```

    This will create two files:
    - `id_ed25519` (private key)
    - `id_ed25519.pub` (public key)

2.  **Create Kubernetes Secret:**

    Create a Kubernetes Secret containing the private key:

    ```bash
    kubectl create secret generic physical-infrastructure-ssh-keys \
      --from-file=id_ed25519=./id_ed25519
    ```

    **Important:** Keep the private key file secure and delete it from the local filesystem after creating the Secret.

3.  **Distribute Public Key to Remote Hosts:**

    Copy the public key to all target hosts that the services will access:

    ```bash
    # For each remote host
    ssh-copy-id -i ./id_ed25519.pub <user>@<remote-host>
    ```

## Deployment

1.  **Deployment the repository using makefile:**

    ```bash
    make deploy
    ```

2.  **Undeployment the repository using makefile:**

    ```bash
    make clean
    ```

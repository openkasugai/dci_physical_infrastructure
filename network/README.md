# Module Name: network_module

This module provides network controller functionality.

## Table of Contents

- [Building and Executing the Module](#building-and-executing-the-module)
- [Generating Protobuf Code](#generating-protobuf-code)
- [Running Tests](#running-tests)
- [License]("#license)

## Building and Executing the Module

To build the module, follow these steps:

1.  **Prerequisites:**
    *   Go (version 1.24.2)
    *   and, required helm, docer, make

2.  **Clone the repository:**

    ```bash
    git clone https://github.com/compsysg/dci_physical_infrastructure.git
    cd dci_physical_infrastructure/network
    ```

3.  **Download go module:**

    ```bash
    go mod download
    ```

4.  **Build the module:**

    ```bash
    go build -o network_module cmd/server/main.go
    ```

5.  **Excecute the module:**

    ```bash
    ./network_module
    ```

## Generating Protobuf Code

This module uses Protocol Buffers (protobuf) for data serialization. To generate the Go code from the `.proto` files, follow these steps:

1.  **Prerequisites:**
    *   `protoc` (Protocol Buffer compiler)
    *   `protoc-gen-go` (Go protobuf code generator)
    *   `protoc-gen-go-grpc` (Go gRPC code generator)
    *   `protoc-gen-validate` (Go protobuf validation code generator)

2.  **Generate common protobuf code (required first):**

    This module depends on common definitions. Generate the common module code first:

    ```bash
    cd ../common
    protoc --proto_path=. \
        --go_out=. \
        --go_opt=paths=source_relative \
        api/proto/common_interface.proto
    ```

3.  **Generate network_module protobuf code:**

    ```bash
    protoc --proto_path=. \
        --proto_path=.. \
        --proto_path=$GOPATH/pkg/mod/github.com/envoyproxy/protoc-gen-validate@v<version>/ \
        --go_out=. \
        --go_opt=paths=source_relative \
        --go-grpc_out=. \
        --go-grpc_opt=paths=source_relative \
        --validate_out=. \
        --validate_opt=paths=source_relative,lang=go \
        ./api/proto/network_interface.proto
    ```

    *   `--proto_path=.`: Specifies the current directory as the location to search for imported `.proto` files.
    *   `--proto_path=..`: Specifies the parent directory to search for common proto files (e.g., `common/api/proto/common_interface.proto`).
    *   `--proto_path=$GOPATH/pkg/mod/github.com/envoyproxy/protoc-gen-validate@v<version>/`: Specifies the path to protoc-gen-validate proto files.
    *   `--go_out=.`: Specifies the output directory for the Go code.
    *   `--go_opt=paths=source_relative`: Generates Go code with relative import paths.
    *   `--go-grpc_out=.`: Specifies the output directory for the gRPC Go code.
    *   `--go-grpc_opt=paths=source_relative`: Generates gRPC Go code with relative import paths.
    *   `--validate_out=.`: Specifies the output directory for the validation code.
    *   `--validate_opt=paths=source_relative,lang=go`: Generates validation code using `protoc-gen-validate` with relative import paths and specifies the language as Go.
    *   `./api/proto/network_interface.proto`: The path to the `.proto` file.

    This command generates the Go code, gRPC stubs, and validation code based on the `network_interface.proto` file. The output files will be placed in the same directory as the `network_interface.proto` file (specified by the `.` in the `--go_out`, `--go-grpc_out`, and `--validate_out` options, combined with the path to the `.proto` file).

    **Note:** This module imports common definitions from [`common/api/proto/common_interface.proto`](../common/api/proto/common_interface.proto). Make sure to generate the common module code before generating this module's code.

## Running Tests

To run the tests for this module, use the following command:

```bash
go test ./... 
```

This will run all tests in the current directory and its subdirectories.

For more detailed test output, including the names of the tests being run and their status (pass/fail), add the -v (verbose) flag:

```bash
go test -v ./...
```

The test results will be displayed in the terminal. Ensure all tests pass before deploying the module. Using the -v flag can help you identify and debug any failing tests more easily.


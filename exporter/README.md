# Module Name: exporter_module

This module provides Exporter functionality.

## Table of Contents

- [Building and Executing the Module](#building-and-executing-the-module)
- [Running Tests](#running-tests)

## Building and Executing the Module

To build the module, follow these steps:

1.  **Prerequisites:**
    *   Go (version 1.24.2)
    *   and, required helm, docer, make

2.  **Clone the repository:**

    ```bash
    git clone https://github.com/compsysg/dci_physical_infrastructure.git
    cd dci_physical_infrastructure/exporter
    ```

3.  **Download go module:**

    ```bash
    go mod download
    ```

4.  **Build the module:**

    ```bash
    go build -o exporter_module cmd/server/main.go
    ```

5.  **Excecute the module:**

    ```bash
    ./exporter_module
    ```

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

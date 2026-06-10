FROM golang:1.24.2

WORKDIR /usr/src/exporter_module

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-venv \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

RUN python3 -m venv /venv

ENV VIRTUAL_ENV=/venv
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN python3 -m pip install --upgrade pip
RUN pip3 install ansible
RUN ansible --version

COPY exporter/internal/server/implementation/ansible.cfg /root/.ansible.cfg

COPY common /usr/src/common

COPY exporter/go.mod exporter/go.sum ./
RUN go mod download

COPY exporter/cmd cmd/
COPY exporter/factory factory/
COPY exporter/internal internal/

RUN go build -o exporter_module cmd/server/main.go

CMD ["./exporter_module"]

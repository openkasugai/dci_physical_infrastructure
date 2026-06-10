FROM golang:1.24.2

WORKDIR /usr/src/cdi_module

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

COPY cdi/internal/server/implementation/ansible.cfg /root/.ansible.cfg

COPY common /usr/src/common

COPY cdi/go.mod cdi/go.sum ./
RUN go mod download

COPY cdi/api api/
COPY cdi/cmd cmd/
COPY cdi/factory factory/
COPY cdi/internal internal/

RUN go build -o cdi_module cmd/server/main.go

CMD ["./cdi_module"]

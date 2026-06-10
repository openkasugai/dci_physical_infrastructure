FROM golang:1.24.2

WORKDIR /usr/src/log_module

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

COPY log/internal/server/implementation/ansible.cfg /root/.ansible.cfg

COPY common /usr/src/common

COPY log/go.mod log/go.sum ./
RUN go mod download

COPY log/cmd cmd/
COPY log/factory factory/
COPY log/internal internal/

RUN go build -o log_module cmd/server/main.go

CMD ["./log_module"]

<h1 align="center">ZMQ Pub Sub Pattern Example</h1>

## Prerequisites

- Go 1.x or higher
- ZeroMQ library installed on your system
  - macOS: `brew install zeromq`
  - Linux: `sudo apt-get install libzmq3-dev`
  - Windows: Download from [ZeroMQ releases](https://github.com/zeromq/libzmq/releases)

## Getting Started

1. Clone the repository

```bash
git clone <repository-url>
cd go-pub-sub
```

2. Build the publisher and subscriber

```bash
make build
```

This will create both executables in the `dist` directory.

3. Run the applications

In another terminal, start the publisher:

```bash
./dist/pub
```

In one terminal, start the subscriber:

```bash
./dist/sub
```

## Project Structure

```
.
├── cmd/
│   ├── pub/    # Publisher application
│   └── sub/    # Subscriber application
└── internal/   # Internal packages
```


```bash
go mod init github.com/ryuichi24/zmq-pub-sub
```

```bash
go mod download
```
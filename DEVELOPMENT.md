# Development Note

This document is intended to be a guide for developers who want to contribute to the project. It is not intended to be a guide for users who want to use the project.

## References

- This project is based on the book "Distributed Services with Go" by Travis Jeffery.
  - [en / Distributed Services with Go](https://pragprog.com/titles/tjgo/distributed-services-with-go/)
    - repo: https://github.com/travisjeffery/proglog (This is the original repo, but it is written in Go v1.13 style.)
  - [jp / Go言語による分散サービス](https://www.oreilly.co.jp/books/9784873119977/)
    - repo: https://github.com/YoshikiShibata/proglog (This is a forked repo, but it is written in Go v1.16 style. <-- recommended)


## Prerequisites

- Go v1.16 or later


## Install dependencies

- Install protoc
  - https://grpc.io/docs/protoc-installation/
- Install protobuff runtime for Go
  - https://developers.google.com/protocol-buffers/docs/gotutorial
  - `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`


## Third-party Go packages

See: [go.mod](./go.mod)
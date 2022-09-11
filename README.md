# proglog
A distributed commit log service. Ref: "Distributed Services with Go"

## References

- This project is based on the book "Distributed Services with Go" by Travis Jeffery.
  - [en / Distributed Services with Go](https://pragprog.com/titles/tjgo/distributed-services-with-go/)
    - repo: https://github.com/travisjeffery/proglog (This is the original repo, but it is written in Go v1.13 style.)
  - [jp / Go言語による分散サービス](https://www.oreilly.co.jp/books/9784873119977/)
    - repo: https://github.com/YoshikiShibata/proglog (This is a forked repo, but it is written in Go v1.16 style. <-- recommended)


## Usage

```bash

# Start a server
$ go run cmd/server/main.go

# API test
# Add a record
curl -X POST localhost:8080 -d '{"record": {"value": "record0"}}'
curl -X POST localhost:8080 -d '{"record": {"value": "record1"}}'
curl -X POST localhost:8080 -d '{"record": {"value": "record2"}}'

# Get a record
curl -X GET localhost:8080 -d '{"offset": 0}'
curl -X GET localhost:8080 -d '{"offset": 1}'
```


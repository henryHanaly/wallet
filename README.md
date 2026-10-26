# wallet api

Small Gin service that keeps user wallets in memory and lets you move money
between them. No database: everything is gone when the process stops.

## Running it

Either way the server listens on `:8080`. Check it's up with:

```
curl localhost:8080/ping
```

### Without Docker

Needs Go 1.26 or newer.

```
go mod download
go run .
```

Or build a binary and run that (see [Building a binary](#building-a-binary)):

```
go build -o api .
./api
```

This starts gin in debug mode, which prints the route list and a couple of
warnings on startup. That's expected locally. To turn it off:

```
GIN_MODE=release go run .           # bash
$env:GIN_MODE="release"; go run .   # PowerShell
```

### With Docker (untested)

> The Dockerfile hasn't been built yet because Docker wasn't available on the
> machine it was written on. The Go build step it runs works, but the image
> itself hasn't been tried. If something breaks, please open an issue.

```
docker build -t wallet-api .
docker run --rm -p 8080:8080 wallet-api
```

The image only has the static binary (distroless, runs as non-root) and
starts gin in release mode. Data is kept in memory, so it is lost every time
the container stops.

## Building a binary

For the machine you're on:

```
go build -o api .        # Linux / macOS
go build -o api.exe .    # Windows
```

For a smaller release build, drop the debug info and local paths:

```
go build -trimpath -ldflags="-s -w" -o api .
```

Go can build for other platforms from any machine. Set `GOOS` and `GOARCH`.
`CGO_ENABLED=0` gives a static binary that doesn't need anything installed
on the target.

bash:

```
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o api-linux-amd64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o api-linux-arm64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o api-mac-arm64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o api-mac-amd64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o api-windows-amd64.exe .
```

PowerShell (the variables stay set for the session, so reset them after):

```
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -o api-linux-amd64 .
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
```

`darwin/arm64` is Apple Silicon (M1 and later), `darwin/amd64` is older
Intel Macs. On Linux or macOS you may need `chmod +x api` before running it.

## Endpoints

| Method | Path                     | Body                                          |
|--------|--------------------------|-----------------------------------------------|
| GET    | `/ping`                  |                                               |
| POST   | `/create`                | `{"username": "alice", "balance": 100}`       |
| GET    | `/getBalance/:username`  |                                               |
| POST   | `/transfer`              | `{"from": "alice", "to": "bob", "amount": 25}` |

Rules:

- username is 3 to 20 characters, starting balance has to be more than 0
- transfer amount has to be more than 0 and the sender needs enough money
- both users must already exist

`/create` answers 201 on success, the rest answer 200.

Errors come back as `{"error": "..."}`, with a 400 for bad input or not enough
funds and a 404 for unknown users.

Quick try:

```
curl -X POST localhost:8080/create -d '{"username":"alice","balance":100}'
curl -X POST localhost:8080/create -d '{"username":"bob","balance":5}'
curl -X POST localhost:8080/transfer -d '{"from":"alice","to":"bob","amount":25}'
curl localhost:8080/getBalance/bob
```

## Layout

```
main.go                      wiring only, builds everything and starts the server
internal/
  router/                    route table
  middleware/                request logging (logrus)
  controller/                HTTP layer: parse + validate request, map errors to status codes
  service/
    command/                 writes (create user, transfer)
    query/                   reads (get balance)
  repository/                in-memory store behind a RWMutex
  model/                     Wallet type and the domain errors
```

Reads and writes are split (command/query) so they can change independently.
For example, the read side could later move to a cache without touching transfers.
Both sides share the same store for now.

The store hands out the map only inside `View` / `Update` callbacks, so the
lock is always held while the map is in use. A transfer does its checks and
the balance change inside one `Update`, so two transfers running at the same
time can't both spend the same money.

The controller depends on small interfaces (`CommandService`, `QueryService`)
rather than the concrete handlers. That's what lets the controller tests use
fakes.

## Logging

Every request gets one logrus line from `internal/middleware/logger.go` with
status, method, path, client IP and latency. 2xx/3xx log at info, 4xx at
warn, 5xx at error. If the handler failed, the reason goes in the `errors`
field:

```
level=warning msg="Not Found" errors="Error #01: user not found\n" method=GET path=/getBalance/ghost status=404
```

## Tests

```
go test ./... -cover
```

Each package under `internal/` is at 100% statement coverage. `router` has an
end-to-end test that goes through the real stack. `main.go` only does the
wiring, so it has no tests.

## Known limitations

- balances are `float32`, so expect rounding on cents. Storing integer cents
  would fix that.
- no persistence, no auth.

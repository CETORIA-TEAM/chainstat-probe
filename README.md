# chainstat-probe

Tiny HTTP server to wrap around the chainstat shell script for CETORIA EVMs.

It provides the JSON output of `chainstat -j` via HTTP while setting a proper
status code (200 for a healthy node and 500 if there is any issue such as being
out of sync or offline).

## Development

### Requirements

- Go 1.23 or higher

### Running

There is not CLI manual/help. First argument is the port, second argument is the
number of required minPeers, third argument is the maximum distance for the CL
head slot and the expected slot as well as the maximum distance for the EL block
with expected block computed from the CL.

```sh
go run . 9788 1 3
```

Then in a different terminal:

```sh
curl -i localhost:9788
```

## Build

Building for production requires the `-ldflags` option to set the environment.
This way, the server will not attempt to load `chainstat-sample.json` to mock
data.

```sh
go build -ldflags "-X main.env=prod" -o bin/chainstat-probe
```

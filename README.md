# 👀 Monitor

This tool can be used to wrap commands and report their outputs as `Success` or `Error` metrics (`Counter`); this is useful when running _health checks_ or _probes_ as one can use Prometheus to monitor the results. It will transparently forward outputs but will `exit` till Prometheus fetched the metrics.

## Usage

```sh
./monitor <flags> <command> <arguments>
```

### Examples

```sh
./monitor --delay 60 curl http://google.com
```

_This configuration will run curl every 60s and use report success unless `http://google.com` is unreachable and `curl` returns a non zero exit code._

```sh
./monitor --delay 10 --success-pattern=E --error-pattern=E pytest
```

_This configuration will run `pytest` every 10s and scan stdout for `E` and `P` and reports successes and errors accordingly._
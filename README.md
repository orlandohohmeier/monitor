# 👀 Monitor

This tool can be used to wrap commands and report their outputs as `Success` or `Error` metrics (`Counter`); this is useful when running _health checks_ or _probes_ as one can use Prometheus to monitor the results. It will transparently forward outputs but will `exit` till Prometheus fetched the metrics.

## Usage

```sh
./monitor <flags> <command> <arguments>
```

### Examples

```sh
./monitor curl http://google.com
```

_This configuration will run curl report success unless `http://google.com` is unreachable and `curl` returns a non zero exit code._

```sh
./monitor --success-pattern=E --error-pattern=E pytest
```

_This configuration will run `pytest`, scan stdout for `E` and `P` and reports successes and errors accordingly._
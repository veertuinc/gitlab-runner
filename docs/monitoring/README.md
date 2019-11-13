# GitLab Runner monitoring

GitLab Runner can be monitored using [Prometheus].

## Embedded Prometheus metrics

> The embedded HTTP Statistics Server with Prometheus metrics was
introduced in GitLab Runner 1.8.0.

The GitLab Runner is instrumented with native Prometheus
metrics, which can be exposed via an embedded HTTP server on the `/metrics`
path. The server - if enabled - can be scraped by the Prometheus monitoring
system or accessed with any other HTTP client.

The exposed information includes:

- Runner business logic metrics (e.g., the number of currently running jobs)
- Go-specific process metrics (garbage collection stats, goroutines, memstats, etc.)
- general process metrics (memory usage, CPU usage, file descriptor usage, etc.)
- build version information

The metrics format is documented in Prometheus'
[Exposition formats](https://prometheus.io/docs/instrumenting/exposition_formats/)
specification.

These metrics are meant as a way for operators to monitor and gain insight into
GitLab Runners. For example, you may be interested if the load average increase
on your runner's host is related to an increase of processed jobs or not. Or
you are running a cluster of machines to be used for the jobs and you want to
track build trends to plan changes in your infrastructure.

### Learning more about Prometheus

To learn how to set up a Prometheus server to scrape this HTTP endpoint and
make use of the collected metrics, see Prometheus's [Getting
started](https://prometheus.io/docs/prometheus/latest/getting_started/) guide. Also
see the [Configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)
section for more details on how to configure Prometheus, as well as the section
on [Alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/) and setting up
an [Alertmanager](https://prometheus.io/docs/alerting/alertmanager/) to
dispatch alert notifications.

## `pprof` HTTP endpoints

> `pprof` integration was introduced in GitLab Runner 1.9.0.

While having metrics about internal state of Runner process is useful
we've found that in some cases it would be good to check what is happening
inside of the Running process in real time. That's why we've introduced
the `pprof` HTTP endpoints.

`pprof` endpoints will be available via an embedded HTTP server on `/debug/pprof/`
path.

You can read more about using `pprof` in its [documentation][go-pprof].

## Configuration of the metrics HTTP server

> **Note:**
The metrics server exports data about the internal state of the
GitLab Runner process and should not be publicly available!

The metrics HTTP server can be configured in two ways:

- with a `listen_address` global configuration option in `config.toml` file,
- with a `--listen-address` command line option for the `run` command.

In both cases the option accepts a string with the format `[host]:<port>`,
where:

- `host` can be an IP address or a host name,
- `port` is a valid TCP port or symbolic service name (like `http`). We recommend to use port `9252` which is already [allocated in Prometheus](https://github.com/prometheus/prometheus/wiki/Default-port-allocations).

If the listen address does not contain a port, it will default to `9252`.

Examples of addresses:

- `:9252` - will listen on all IPs of all interfaces on port `9252`
- `localhost:9252` - will only listen on the loopback interface on port `9252`
- `[2001:db8::1]:http` - will listen on IPv6 address `[2001:db8::1]` on the HTTP port `80`

Remember that for listening on ports below `1024` - at least on Linux/Unix
systems - you need to have root/administrator rights.

Also please notice, that HTTP server is opened on selected `host:port`
**without any authorization**. If you plan to bind the metrics server
to a public interface then you should consider to use your firewall to
limit access to this server or add a HTTP proxy which will add the
authorization and access control layer.

[go-pprof]: https://golang.org/pkg/net/http/pprof/
[prometheus]: https://prometheus.io

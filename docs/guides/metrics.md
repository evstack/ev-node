# Evolve Metrics Guide

## How to configure metrics

Evolve can report and serve Prometheus metrics, which can be consumed by Prometheus collector(s).

This functionality is disabled by default.

To enable Prometheus metrics, set `instrumentation.prometheus=true` in your Evolve node's configuration file.

Metrics will be served under `/metrics` on port 26660 by default. The listening address can be changed using the `instrumentation.prometheus_listen_addr` configuration option.

## List of available metrics

You can find the full list of available metrics in the [Technical Specifications](../learn/specs/block-manager.html#metrics).

## Viewing Metrics

Once your Evolve node is running with metrics enabled, you can view the metrics by:

1. Accessing the metrics endpoint directly:

   ```bash
   curl http://localhost:26660/metrics
   ```

2. Configuring Prometheus to scrape these metrics by adding the following to your `prometheus.yml`:

   ```yaml
   scrape_configs:
     - job_name: evolve
       static_configs:
         - targets: ['localhost:26660']
   ```

3. Using Grafana with Prometheus as a data source to visualize the metrics.

## Example Prometheus Configuration

Here's a basic Prometheus configuration to scrape metrics from a Evolve node:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: evolve
    static_configs:
      - targets: ['localhost:26660']
```

## Troubleshooting

If you're not seeing metrics:

1. Ensure metrics are enabled in your configuration with `instrumentation.prometheus=true`
2. Verify the metrics endpoint is accessible: `curl http://localhost:26660/metrics`
3. Check your Prometheus configuration is correctly pointing to your Evolve node
4. Examine the Evolve node logs for any errors related to the metrics server

## Advanced Configuration

For more advanced metrics configuration, you can adjust the following settings in your configuration file:

```yaml
instrumentation:
  prometheus: true
  prometheus_listen_addr: ":26660"
  max_open_connections: 3
  namespace: "evolve"
```

These settings allow you to:

- Enable/disable Prometheus metrics
- Change the listening address for the metrics server
- Limit the maximum number of open connections to the metrics server
- Set a custom namespace for all metrics

# DA Visualizer

The Data Availability (DA) Visualizer is a built-in monitoring tool in Evolve that provides real-time insights into blob submissions to the DA layer. It offers a web-based interface for tracking submission statistics, monitoring DA layer health, and analyzing blob details.

**Note**: Only aggregator nodes submit data to the DA layer. Non-aggregator nodes will not display submission data.

## Overview

The DA Visualizer provides:

- Real-time monitoring of blob submissions (last 100 submissions)
- Success/failure statistics and trends
- Gas price tracking and cost analysis
- DA layer health monitoring
- Detailed blob inspection capabilities
- Recent submission history

## Enabling the DA Visualizer

The DA Visualizer is disabled by default. To enable it, use the following configuration:

### Via Command-line Flag

```bash
testapp start --rollkit.rpc.enable_da_visualization
```

### Via Configuration File

Add the following to your `evnode.yml` configuration file:

```yaml
rpc:
  enable_da_visualization: true
```

## Accessing the DA Visualizer

Once enabled, the DA Visualizer is accessible through your node's RPC server. By default, this is:

```
http://localhost:7331/da
```

The visualizer provides several API endpoints and a web interface:

### Web Interface

Navigate to `http://localhost:7331/da` in your web browser to access the interactive dashboard.

### API Endpoints

The following REST API endpoints are available for programmatic access:

#### Get Recent Submissions

```bash
GET /da/submissions
```

Returns the most recent blob submissions (up to 100 kept in memory).

#### Get Blob Details

```bash
GET /da/blob?id={blob_id}
```

Returns detailed information about a specific blob submission.

#### Get DA Statistics

```bash
GET /da/stats
```

Returns aggregated statistics including:

- Total submissions count
- Success/failure rates
- Average gas price
- Total gas spent
- Average blob size
- Submission trends

#### Get DA Health Status

```bash
GET /da/health
```

Returns the current health status of the DA layer including:

- Connection status
- Recent error rates
- Performance metrics
- Last successful submission timestamp

## Features

### Real-time Monitoring

The dashboard automatically updates every 30 seconds, displaying:

- Recent submission feed with status indicators (last 100 submissions)
- Success rate percentage
- Current gas price trends
- Submission history

### Submission Details

Each submission entry shows:

- Timestamp
- Blob ID with link to detailed view
- Number of blobs in the batch
- Submission status (success/failure)
- Gas price used
- Error messages (if any)

### Statistics Dashboard

The statistics section provides:

- **Performance Metrics**: Success rate, average submission time
- **Cost Analysis**: Total gas spent, average gas price over time
- **Volume Metrics**: Total blobs submitted, average blob size
- **Trend Analysis**: Hourly and daily submission patterns

### Health Monitoring

The health status indicator shows:

- 🟢 **Healthy**: DA layer responding normally
- 🟡 **Warning**: Some failures but overall functional
- 🔴 **Critical**: High failure rate or connection issues

## Use Cases

### For Node Operators

- Monitor the reliability of DA submissions
- Track gas costs and optimize gas price settings
- Identify patterns in submission failures
- Ensure DA layer connectivity

### For Developers

- Debug DA submission issues
- Analyze blob data structure
- Monitor application-specific submission patterns
- Test DA layer integration

### For Network Monitoring

- Track overall network DA usage
- Identify congestion periods
- Monitor gas price fluctuations
- Analyze submission patterns across the network

## Configuration Options

When enabling the DA Visualizer, you may want to adjust related RPC settings:

```yaml
rpc:
  address: "0.0.0.0:7331" # Bind to all interfaces for remote access
  enable_da_visualization: true
```

**Security Note**: If binding to all interfaces (`0.0.0.0`), ensure proper firewall rules are in place to restrict access to trusted sources only.

## Troubleshooting

### Visualizer Not Accessible

1. Verify the DA Visualizer is enabled:
   - Check your configuration file or ensure the flag is set
   - Look for log entries confirming "DA visualization endpoints registered"

2. Check the RPC server is running:
   - Verify the RPC address in logs
   - Ensure no port conflicts

3. For remote access:
   - Ensure the RPC server is bound to an accessible interface
   - Check firewall settings

### No Data Displayed

1. Verify your node is in aggregator mode (only aggregators submit to DA)
2. Check DA layer connectivity in the node logs
3. Ensure transactions are being processed
4. Note that the visualizer only keeps the last 100 submissions in memory

### API Errors

- **404 Not Found**: DA Visualizer not enabled
- **500 Internal Server Error**: Check node logs for DA connection issues
- **Empty responses**: No submissions have been made yet

## Example Usage

### Using curl to access the API

```bash
# Get recent submissions (returns up to 100)
curl http://localhost:7331/da/submissions

# Get specific blob details
curl http://localhost:7331/da/blob?id=abc123...

# Get statistics
curl http://localhost:7331/da/stats

# Check DA health
curl http://localhost:7331/da/health
```

### Monitoring with scripts

```bash
#!/bin/bash
# Simple monitoring script

while true; do
  health=$(curl -s http://localhost:7331/da/health | jq -r '.status')
  if [ "$health" != "healthy" ]; then
    echo "DA layer issue detected: $health"
    # Send alert...
  fi
  sleep 30
done
```

## Related Configuration

For complete DA layer configuration options, see the [Config Reference](../../learn/config.md#data-availability-configuration-da).

For metrics and monitoring setup, see the [Metrics Guide](../metrics.md).

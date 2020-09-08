# Critic
Critic is a very simple web monitoring tool.
It periodically (every 30 seconds) polls URLs provided as values of environment variables with the name prefix `WATCH_` and exports them as prometheus metrics.

## Exported metrics
Following metrics are exported for each target:

### critic_target_is_healthy
The value of this metric will be set to `1` if the probe deems target to be healthy.
Target is assumed to be healthy when the HTTP request succeeds with the stats code between `100` and `499`, excluding `404`.

### critic_target_probe_failed_counter
This counter increases every time probe for the target fails (either can't connect or the status code is `> 500` or `404`)

### critic_target_request_duration
Request duration of the last probe in seconds.

### critic_target_server_certificate_expiration_time
Unix time of the TLS Certificate expiration.
If the protocol is not `https`, this value will be `0`

### critic_target_status_code
HTTP Status code of the last probe.
If creating request has failed, value will be `0`.
In the case that performing request has failed, the value will be `1`.

# ELB Health Check

AWS elastic load balancers and application load balancers are awesome, but it's not possible to specify any headers (such as the Host header) to allow for virtual hosts to be taken into account when determining if the host is healthy or not.

## What problems does this app intend to solve?

- A health check pinging the default virtualhost is not a meaningful indication of whether or not your app is up or down.
- You have multiple applications that all must be up for the ELB/ALB to consider your host healthy.
- Hitting port 80 with default ELB health checks triggers a 301 redirect to HTTPS even though HTTPS is terminated at the ELB.

## How do I use this?

Set a `HEALTH_CHECK_HOSTNAMES` environment variable that contains a comma separated list of hostnames that should be checked as part of each health check.

Add a `/health_check` endpoint in each application that returns any `20x` status code in response to a `HEAD` request when it should be considered healthy.

Compile it with Go and let it run!

[Point ELB's health checker](http://docs.aws.amazon.com/elasticloadbalancing/latest/application/target-group-health-checks.html) at the port where this Rack application is running (default is 9292). Use an HTTP for the protocol and use `/` as the path.

Verify it succeeds/fails based on the status of your app. Here's an example of what to expect if you're running it on the default port.

```
export HEALTH_CHECK_HOSTNAMES=github.com,google.com
go run main.go
curl -I http://localhost:9292/

HTTP/1.1 200 OK
Content-Type: application/json
Transfer-Encoding: chunked

[{"github.com":{"status":200,"check":"success"}},{"google.com":{"status":200,"check":"success"}}]
```

You probably want to keep this running with something like upstart, monit, or systemd instead of running it manually, but that's outside the scope of this document and specific to your configuration.

## Additional options

Sometimes you may have more than one load balancer in front of a group of EC2 instances, and there may be applications included in `HEALTH_CHECK_HOSTNAMES` that a particular load balancer doesn't care about. In this case you don't want the load balancer to fail a health check for an unrelated application. Similarly, there may just be a single app that a load balancer cares about.

You can provide a whitelist or blacklist of hostnames for a given load balancer.

In your ELB `HealthCheckPath`, specify _either_ `allowed_to_fail` or `must_succeed` as a URL parameter with a comma-separated list of hostnames that should be required by or omitted from the health check.

```
HEALTH_CHECK_HOSTNAMES=www.important.com,whocares.com
HealthCheckPath: /?allowed_to_fail=whocares.com
```

In the above case, a health check ping will hit _both_ hostnames and the results for _both_ hostnames will be returned, but the result of `whocares.com` will not impact the overall ability of the health check to pass.

```
HEALTH_CHECK_HOSTNAMES=www.important.com,whocares.com
HealthCheckPath: /?must_succeed=www.important.com
```

In the above case, a health check ping will hit _both_ hostnames and the results for _both_ hostnames will be returned but the health check's ability to pass depends solely on the status of `www.important.com`

## Caveats

- Requests to applications are made concurrently to keep things as quick as possible, but if you have oodles of applications running on a server, this may time out before it's able to respond to ELB. You can tune the `HealthCheckTimeoutSeconds` setting if necessary.
- If your application spawns processes from the first request after a timeout, keep in mind this is going to hit all the apps at the same time, which could cause load spikes and/or timeouts.
- You better monitor this process and make sure it stays running! If it isn't running, your health check is going to fail and the node will be taken out of service.

## Copyright

&copy; 2022 James Miller

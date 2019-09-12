# hashworks.net source
[![coverage report](https://git.hashworks.net/hashworks/hashworksNET/badges/master/coverage.svg)](https://git.hashworks.net/hashworks/hashworksNET/commits/master) [![codecov](https://codecov.io/gh/hashworks/hashworksNET/branch/dev/graph/badge.svg)](https://codecov.io/gh/hashworks/hashworksNET)

Repository of [hashworks.net](https://hashworks.net).

While serving my contact information it mainly is acting as my playground for web technologies.

## HTTP Handling
Nginx is used as a TLS proxy, forwarding HTTP requests to the server.
Currently I'm using [gin](https://github.com/gin-gonic/gin) for routing and middleware handling. Caching is handled by the [gin-contrib/cache](https://github.com/gin-contrib/cache) middleware, using an included memcache.

## Frontend
I'm using the Go template engine to provide everything. CSS is included as inline stylesheets to avoid preloading issues, beside some exceptions for page size. I wanted to avoid absurd amounts of large requests and performance issues altogether, so I decided to strictly avoid any JavaScript and off-site requests. Any scripts are forbidden by [CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP) and CSS is tightly controlled as well.

## Testing
Using the [httptest](https://golang.org/pkg/net/http/httptest/) package we can unit test all routing endpoints quite easily. I try to keep the coverage over 85 percent.

Additionally, acception tests are done by the CD system.

## Continuous Delivery
[Drone](https://drone.io/) in combination with [Ansible](https://www.ansible.com/) is used as the CD system. Any commit or pull request is built, tested and provided with a coverage report. Merges on the protected master branch will be deployed after successfull unit tests. Since the server is a single binary, including all static files thanks to [fileb0x](https://github.com/UnnoTed/fileb0x/) this process becomes trivial. Using systemd the service is sandboxed.

After a successfull deployment acception tests using [agouti](https://github.com/sclevine/agouti) are run. On failure a backup is restored.

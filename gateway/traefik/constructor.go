package traefik

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CXeon/tiles/gateway"
)

type constructor struct {
	prefix string
}

func NewConstructor(prefix ...string) *constructor {
	pre := "traefik"
	if len(prefix) > 0 {
		pre = prefix[0]
	}
	return &constructor{
		prefix: pre,
	}
}

/** Routers **/

// 生成默认的router名
func (con *constructor) genDefaultRouterName(endpoint *gateway.Endpoint) string {
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s", endpoint.Env, endpoint.Cluster, endpoint.Company, endpoint.Project, endpoint.Service, endpoint.Protocol, endpoint.Color)
}

// 生成默认的key前缀
func (con *constructor) genDefaultRouterPrefix(endpoint *gateway.Endpoint) string {
	protocol := strings.ToLower(endpoint.Protocol)
	if protocol == "https" {
		protocol = "http"
	}

	return fmt.Sprintf("%s/%s/%s/%s/", con.prefix, protocol, "routers", con.genDefaultRouterName(endpoint))
}

// GenRouterRuleKey 生成router的rule key
func (con *constructor) GenRouterRuleKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/rule"
}

// GenRouterEntrypointKeyPrefix 生成router的entrypoint key前缀
func (con *constructor) GenRouterEntrypointKeyPrefix(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/entrypoints/"
}

// GenRouterEntrypointKey 生成router的entrypoint key
func (con *constructor) GenRouterEntrypointKey(index int, endpoint gateway.Endpoint) string {
	return con.GenRouterEntrypointKeyPrefix(endpoint) + strconv.Itoa(index)
}

// GenRouterMiddlewareKey 生成router的middleware key
func (con *constructor) GenRouterMiddlewareKey(index int, endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/middlewares/" + strconv.Itoa(index)
}

// GenRouterServiceKey 生成router的service key
func (con *constructor) GenRouterServiceKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/service"
}

// GenRouterPriorityKey 生成router的priority key
func (con *constructor) GenRouterPriorityKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/priority"
}

// GenRouterObservabilityAccesslogsKey 生成router的observability accesslogs key
func (con *constructor) GenRouterObservabilityAccesslogsKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/observability/accesslogs"
}

// GenRouterObservabilityMetricsKey 生成router的observability metrics key
func (con *constructor) GenRouterObservabilityMetricsKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/observability/metrics"
}

// GenRouterObservabilityTracingKey 生成router的observability tracing key
func (con *constructor) GenRouterObservabilityTracingKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "/observability/tracing"
}

/**Services**/

// 生成默认的service名
func (con *constructor) genDefaultServiceName(endpoint *gateway.Endpoint) string {
	return endpoint.ID()
}

// 生成默认的service key前缀
func (con *constructor) genDefaultServicePrefix(endpoint *gateway.Endpoint) string {
	protocol := strings.ToLower(endpoint.Protocol)
	if protocol == "https" {
		protocol = "http"
	}
	return fmt.Sprintf("%s/%s/%s/%s/", con.prefix, protocol, "services", con.genDefaultServiceName(endpoint))
}

func (con *constructor) GenServiceLoadbalancerServiceKeyPrefix(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/servers/"
}

// GenServiceUrlKey 生成service的url key
func (con *constructor) GenServiceUrlKey(index int, endpoint gateway.Endpoint) string {
	return con.GenServiceLoadbalancerServiceKeyPrefix(endpoint) + strconv.Itoa(index) + "/url"
}

// GenServicePreservePathKey 生成service的preservePath key
func (con *constructor) GenServicePreservePathKey(index int, endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/servers/" + strconv.Itoa(index) + "/preservePath"
}

// GenServiceWeightKey 生成service的weight key
func (con *constructor) GenServiceWeightKey(index int, endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/servers/" + strconv.Itoa(index) + "/weight"
}

// GenServicePassHostHeaderKey 生成service的passHostHeader key
func (con *constructor) GenServicePassHostHeaderKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/passhostheader"
}

// GenServiceHealthCheckHeadersKey 生成service的healthCheckHeaders key
func (con *constructor) GenServiceHealthCheckHeadersKey(headerKey string, endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/headers/" + headerKey
}

// GenServiceHealthCheckHostNameKey 生成service的healthCheckHostName key
func (con *constructor) GenServiceHealthCheckHostNameKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/hostname"
}

// GenServiceHealthCheckIntervalKey 生成service的healthCheckInterval key
func (con *constructor) GenServiceHealthCheckIntervalKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/interval"
}

// GenServiceHealthCheckPathKey 生成service的healthCheckPath key
func (con *constructor) GenServiceHealthCheckPathKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/path"
}

// GenServiceHealthCheckMethodKey 生成service的healthCheckMethod key
func (con *constructor) GenServiceHealthCheckMethodKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/method"
}

// GenServiceHealthCheckStatusKey 生成service的healthCheckStatus key
func (con *constructor) GenServiceHealthCheckStatusKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/status"
}

// GenServiceHealthCheckPortKey 生成service的healthCheckPort key
func (con *constructor) GenServiceHealthCheckPortKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/port"
}

// GenServiceHealthCheckSchemeKey 生成service的healthCheckScheme key
func (con *constructor) GenServiceHealthCheckSchemeKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/scheme"
}

// GenServiceHealthCheckTimeoutKey 生成service的healthCheckTimeout key
func (con *constructor) GenServiceHealthCheckTimeoutKey(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/healthcheck/timeout"
}

// GenServiceAddressKey 生成service的address key
func (con *constructor) GenServiceAddressKey(index int, endpoint gateway.Endpoint) string {
	protocol := strings.ToLower(endpoint.Protocol)
	if protocol != "udp" {
		return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/servers/" + strconv.Itoa(index) + "/address"
	}
	prefix := con.genDefaultServicePrefix(&endpoint)
	prefix = strings.Replace(prefix, endpoint.ID()+"/", "", 1)
	return prefix + "loadBalancer/servers/" + strconv.Itoa(index) + "/address"
}

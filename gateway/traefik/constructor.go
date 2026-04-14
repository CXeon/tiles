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
func (con *constructor) genDefaultRouterName(endpoint *gateway.Endpoint, suffix ...string) string {
	name := fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s", endpoint.Env, endpoint.Cluster, endpoint.Company, endpoint.Project, endpoint.Service, endpoint.Protocol, endpoint.Color)
	if len(suffix) > 0 && suffix[0] != "" {
		name += "." + suffix[0]
	}
	return name
}

// 生成默认的key前缀
func (con *constructor) genDefaultRouterPrefix(endpoint *gateway.Endpoint, suffix ...string) string {
	protocol := strings.ToLower(string(endpoint.Protocol))
	if protocol == "https" {
		protocol = "http"
	}

	return fmt.Sprintf("%s/%s/%s/%s/", con.prefix, protocol, "routers", con.genDefaultRouterName(endpoint, suffix...))
}

// GenRouterRuleKey 生成router的rule key
func (con *constructor) GenRouterRuleKey(endpoint gateway.Endpoint, suffix ...string) string {
	return con.genDefaultRouterPrefix(&endpoint, suffix...) + "rule"
}

// GenRouterEntrypointKeyPrefix 生成router的entrypoint key前缀
func (con *constructor) GenRouterEntrypointKeyPrefix(endpoint gateway.Endpoint, suffix ...string) string {
	return con.genDefaultRouterPrefix(&endpoint, suffix...) + "entrypoints/"
}

// GenRouterEntrypointKey 生成router的entrypoint key
func (con *constructor) GenRouterEntrypointKey(index int, endpoint gateway.Endpoint, suffix ...string) string {
	return con.GenRouterEntrypointKeyPrefix(endpoint, suffix...) + strconv.Itoa(index)
}

// GenRouterMiddlewareKey 生成router的middleware key
func (con *constructor) GenRouterMiddlewareKey(index int, endpoint gateway.Endpoint, suffix ...string) string {
	return con.genDefaultRouterPrefix(&endpoint, suffix...) + "middlewares/" + strconv.Itoa(index)
}

// GenRouterServiceKey 生成router的service key
func (con *constructor) GenRouterServiceKey(endpoint gateway.Endpoint, suffix ...string) string {
	return con.genDefaultRouterPrefix(&endpoint, suffix...) + "service"
}

// GenRouterPriorityKey 生成router的priority key
func (con *constructor) GenRouterPriorityKey(endpoint gateway.Endpoint, suffix ...string) string {
	return con.genDefaultRouterPrefix(&endpoint, suffix...) + "priority"
}

// GenRouterObservabilityAccesslogsKey 生成router的observability accesslogs key
func (con *constructor) GenRouterObservabilityAccesslogsKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "observability/accesslogs"
}

// GenRouterObservabilityMetricsKey 生成router的observability metrics key
func (con *constructor) GenRouterObservabilityMetricsKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "observability/metrics"
}

// GenRouterObservabilityTracingKey 生成router的observability tracing key
func (con *constructor) GenRouterObservabilityTracingKey(endpoint gateway.Endpoint) string {
	return con.genDefaultRouterPrefix(&endpoint) + "observability/tracing"
}

// GenRouterPrefixAll 返回所有 router 的共同前缀（用于批量删除 protected 和 public router）
// 例如: traefik/http/routers/dev.china.testco.testprj.testsvc.http.blue
func (con *constructor) GenRouterPrefixAll(endpoint gateway.Endpoint) string {
	protocol := strings.ToLower(string(endpoint.Protocol))
	if protocol == "https" {
		protocol = "http"
	}
	// 返回到 router name 的基础前缀，包含所有后缀（如 .public）
	baseName := fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s",
		endpoint.Env, endpoint.Cluster, endpoint.Company,
		endpoint.Project, endpoint.Service, protocol, endpoint.Color)
	return fmt.Sprintf("%s/%s/%s/%s", con.prefix, protocol, "routers", baseName)
}

/**Services**/

// genDefaultServiceName 生成默认的service名（基于服务逻辑标识，不包含实例信息）
// 同一服务的多个实例共享同一个 Service Name，通过 loadbalancer 实现负载均衡
func (con *constructor) genDefaultServiceName(endpoint *gateway.Endpoint) string {
	protocol := strings.ToLower(string(endpoint.Protocol))
	if protocol == "https" {
		protocol = "http"
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s",
		endpoint.Env, endpoint.Cluster, endpoint.Company,
		endpoint.Project, endpoint.Service, protocol, endpoint.Color)
}

// GenServiceName 导出方法，调用私有方法实现
func (con *constructor) GenServiceName(endpoint gateway.Endpoint) string {
	return con.genDefaultServiceName(&endpoint)
}

// 生成默认的service key前缀
func (con *constructor) genDefaultServicePrefix(endpoint *gateway.Endpoint) string {
	protocol := strings.ToLower(string(endpoint.Protocol))
	if protocol == "https" {
		protocol = "http"
	}
	return fmt.Sprintf("%s/%s/%s/%s/", con.prefix, protocol, "services", con.genDefaultServiceName(endpoint))
}

// GenServicePrefix 返回整个 service 的前缀（用于批量删除）
func (con *constructor) GenServicePrefix(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint)
}

func (con *constructor) GenServiceLoadbalancerServiceKeyPrefix(endpoint gateway.Endpoint) string {
	return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/servers/"
}

// GenServiceInstancePrefix 返回单个服务实例的前缀（用于删除特定实例）
// 例如: traefik/http/services/[service-id]/loadbalancer/servers/0/
func (con *constructor) GenServiceInstancePrefix(index int, endpoint gateway.Endpoint) string {
	return con.GenServiceLoadbalancerServiceKeyPrefix(endpoint) + strconv.Itoa(index) + "/"
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
	protocol := strings.ToLower(string(endpoint.Protocol))
	if protocol != "udp" {
		return con.genDefaultServicePrefix(&endpoint) + "loadbalancer/servers/" + strconv.Itoa(index) + "/address"
	}
	prefix := con.genDefaultServicePrefix(&endpoint)
	prefix = strings.Replace(prefix, endpoint.ID()+"/", "", 1)
	return prefix + "loadBalancer/servers/" + strconv.Itoa(index) + "/address"
}

/**Middlewares**/

// MiddlewareName generates the ForwardAuth middleware name.
// Format: {Env}.{Cluster}.{Company}.{Project}.ForwardAuth
func (con *constructor) MiddlewareName(env, cluster, company, project string) string {
	return fmt.Sprintf("%s.%s.%s.%s.ForwardAuth", env, cluster, company, project)
}

// MiddlewareKeyPrefix returns the KV key prefix for a ForwardAuth middleware.
// Format: {prefix}/http/middlewares/{name}/
func (con *constructor) MiddlewareKeyPrefix(name string) string {
	return fmt.Sprintf("%s/http/middlewares/%s/", con.prefix, name)
}

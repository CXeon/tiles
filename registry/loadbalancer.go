package registry

type LoadBalancer interface {
	Select([]Endpoint) *Endpoint
}

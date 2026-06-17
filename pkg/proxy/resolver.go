package proxy

import "fmt"

// KubeResolver resolves service names to their Kubernetes DNS address.
// Format: <service-name>.<namespace>.svc.cluster.local:<port>
//
// In local development (outside Kubernetes), set RIDEGO_LOCAL_DEV=true
// and each service's address is resolved as localhost:<port> instead.
type KubeResolver struct {
	namespace string
	localDev  bool
}

func NewKubeResolver(namespace string, localDev bool) *KubeResolver {
	return &KubeResolver{namespace: namespace, localDev: localDev}
}

var servicePorts = map[string]string{
	"user-service":     "6001",
	"trip-service":     "6002",
	"location-service": "6003",
	"matching-service": "6004",
	"payment-service":  "6005",
}

// Resolve returns the address for a service.
//   - In cluster:  user-service.ridego.svc.cluster.local:6001
//   - Local dev:   localhost:6001
func (r *KubeResolver) Resolve(name string) (string, error) {
	port, ok := servicePorts[name]
	if !ok {
		return "", fmt.Errorf("unknown service: %s", name)
	}
	if r.localDev {
		return fmt.Sprintf("localhost:%s", port), nil
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local:%s", name, r.namespace, port), nil
}

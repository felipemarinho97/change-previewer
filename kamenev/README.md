# kamenev

This service is responsible for managing the services that are running on the cluster. Provides a REST API for creating, updating, and destroying resources.

## Development

Install kubebuilder:
```bash
# download kubebuilder and install locally.
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```
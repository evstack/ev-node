//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sync"

	"github.com/celestiaorg/tastora/framework/docker/container"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	"github.com/moby/moby/api/types/network"
	"go.uber.org/zap"
)

type victoriaTracesNodeType int

func (victoriaTracesNodeType) String() string { return "victoriatraces" }

const defaultVictoriaTracesHTTPPort = "10428"

type victoriaTracesConfig struct {
	Logger          *zap.Logger
	DockerClient    tastoratypes.TastoraDockerClient
	DockerNetworkID string
	Image           container.Image
}

type victoriaTracesNode struct {
	*container.Node

	cfg     victoriaTracesConfig
	logger  *zap.Logger
	started bool
	mu      sync.Mutex

	internalHTTPPort string
	externalHTTPPort string

	Internal victoriaTracesScope
	External victoriaTracesScope
}

func newVictoriaTracesNode(ctx context.Context, cfg victoriaTracesConfig, testName string, index int) (*victoriaTracesNode, error) {
	img := cfg.Image
	if img.Repository == "" {
		img = container.NewImage("victoriametrics/victoria-traces", "latest", "")
	}

	log := cfg.Logger.With(zap.String("component", "victoriatraces"), zap.Int("i", index))
	home := "/home/victoriatraces"
	n := &victoriaTracesNode{cfg: cfg, logger: log}
	n.Node = container.NewNode(cfg.DockerNetworkID, cfg.DockerClient, testName, img, home, index, victoriaTracesNodeType(0), log)
	n.SetContainerLifecycle(container.NewLifecycle(cfg.Logger, cfg.DockerClient, n.Name()))
	if err := n.CreateAndSetupVolume(ctx, n.Name()); err != nil {
		return nil, err
	}
	n.Internal = victoriaTracesScope{hostname: func() string { return n.Name() }, port: &n.internalHTTPPort}
	n.External = victoriaTracesScope{hostname: func() string { return "0.0.0.0" }, port: &n.externalHTTPPort}
	return n, nil
}

func (n *victoriaTracesNode) Name() string {
	return fmt.Sprintf("victoriatraces-%d-%s", n.Index, sanitizeDockerResourceName(n.TestName))
}

func (n *victoriaTracesNode) HostName() string {
	return condenseHostName(n.Name())
}

func (n *victoriaTracesNode) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.started {
		return n.StartContainer(ctx)
	}
	if err := n.createContainer(ctx); err != nil {
		return err
	}
	if err := n.ContainerLifecycle.StartContainer(ctx); err != nil {
		return err
	}
	hostPorts, err := n.ContainerLifecycle.GetHostPorts(ctx, n.internalHTTPPort+"/tcp")
	if err != nil {
		return err
	}
	n.externalHTTPPort, err = extractPort(hostPorts[0])
	if err != nil {
		return err
	}
	n.started = true
	return nil
}

func (n *victoriaTracesNode) createContainer(ctx context.Context) error {
	if n.internalHTTPPort == "" {
		n.internalHTTPPort = defaultVictoriaTracesHTTPPort
	}

	ports := network.PortMap{
		network.MustParsePort(n.internalHTTPPort + "/tcp"): {},
	}
	cmd := []string{"-storageDataPath", n.HomeDir() + "/data"}
	return n.CreateContainer(ctx, n.TestName, n.NetworkID, n.Image, ports, "", n.Bind(), nil, n.HostName(), cmd, nil, nil)
}

type victoriaTracesScope struct {
	hostname func() string
	port     *string
}

func (s victoriaTracesScope) IngestHTTPEndpoint() string {
	return fmt.Sprintf("http://%s:%s/insert/opentelemetry/v1/traces", s.hostname(), *s.port)
}

func (s victoriaTracesScope) OTLPBaseEndpoint() string {
	return fmt.Sprintf("http://%s:%s/insert/opentelemetry", s.hostname(), *s.port)
}

func (s victoriaTracesScope) QueryURL() string {
	return fmt.Sprintf("http://%s:%s", s.hostname(), *s.port)
}

func extractPort(address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	return port, err
}

func condenseHostName(name string) string {
	if len(name) < 64 {
		return name
	}
	return name[:30] + "_._" + name[len(name)-30:]
}

var validContainerCharsRE = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

func sanitizeDockerResourceName(name string) string {
	return validContainerCharsRE.ReplaceAllLiteralString(name, "_")
}

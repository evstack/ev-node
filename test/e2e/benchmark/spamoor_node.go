//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/consts"
	tastoracontainer "github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/celestiaorg/tastora/framework/docker/port"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	"github.com/containerd/errdefs"
	mobycontainer "github.com/moby/moby/api/types/container"
	mobynetwork "github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
	"go.uber.org/zap"
)

type spamoorBenchmarkNodeType int

func (spamoorBenchmarkNodeType) String() string { return "spamoor" }

const defaultSpamoorHTTPPort = "8080"

type spamoorBenchmarkConfig struct {
	DockerClient    tastoratypes.TastoraDockerClient
	DockerNetworkID string
	Logger          *zap.Logger
	Image           tastoracontainer.Image

	RPCHosts            []string
	PrivateKey          string
	AdditionalStartArgs []string
	HostNetwork         bool
}

type spamoorBenchmarkBuilder struct {
	testName      string
	dockerClient  tastoratypes.TastoraDockerClient
	dockerNetwork string
	logger        *zap.Logger
	image         tastoracontainer.Image

	rpcHosts            []string
	privateKey          string
	nameSuffix          string
	additionalStartArgs []string
	hostNetwork         bool
}

func newSpamoorNodeBuilder(testName string) *spamoorBenchmarkBuilder {
	return &spamoorBenchmarkBuilder{
		testName: testName,
		image:    tastoracontainer.NewImage("ethpandaops/spamoor", "latest", ""),
	}
}

func (b *spamoorBenchmarkBuilder) WithDockerClient(c tastoratypes.TastoraDockerClient) *spamoorBenchmarkBuilder {
	b.dockerClient = c
	return b
}

func (b *spamoorBenchmarkBuilder) WithDockerNetworkID(id string) *spamoorBenchmarkBuilder {
	b.dockerNetwork = id
	return b
}

func (b *spamoorBenchmarkBuilder) WithLogger(l *zap.Logger) *spamoorBenchmarkBuilder {
	b.logger = l
	return b
}

func (b *spamoorBenchmarkBuilder) WithImage(img tastoracontainer.Image) *spamoorBenchmarkBuilder {
	b.image = img
	return b
}

func (b *spamoorBenchmarkBuilder) WithRPCHosts(hosts ...string) *spamoorBenchmarkBuilder {
	b.rpcHosts = hosts
	return b
}

func (b *spamoorBenchmarkBuilder) WithPrivateKey(pk string) *spamoorBenchmarkBuilder {
	b.privateKey = pk
	return b
}

func (b *spamoorBenchmarkBuilder) WithNameSuffix(s string) *spamoorBenchmarkBuilder {
	b.nameSuffix = s
	return b
}

func (b *spamoorBenchmarkBuilder) WithAdditionalStartArgs(args ...string) *spamoorBenchmarkBuilder {
	b.additionalStartArgs = args
	return b
}

func (b *spamoorBenchmarkBuilder) WithHostNetwork() *spamoorBenchmarkBuilder {
	b.hostNetwork = true
	return b
}

func (b *spamoorBenchmarkBuilder) Build(ctx context.Context) (*spamoorBenchmarkNode, error) {
	cfg := spamoorBenchmarkConfig{
		DockerClient:        b.dockerClient,
		DockerNetworkID:     b.dockerNetwork,
		Logger:              b.logger,
		Image:               b.image,
		RPCHosts:            b.rpcHosts,
		PrivateKey:          b.privateKey,
		AdditionalStartArgs: b.additionalStartArgs,
		HostNetwork:         b.hostNetwork,
	}
	return newSpamoorBenchmarkNode(ctx, cfg, b.testName, 0, b.nameSuffix)
}

type spamoorBenchmarkNode struct {
	*tastoracontainer.Node

	cfg               spamoorBenchmarkConfig
	logger            *zap.Logger
	started           bool
	mu                sync.Mutex
	external          tastoratypes.Ports
	name              string
	containerID       string
	preStartListeners port.Listeners
}

func newSpamoorBenchmarkNode(ctx context.Context, cfg spamoorBenchmarkConfig, testName string, index int, name string) (*spamoorBenchmarkNode, error) {
	if cfg.Image.Repository == "" {
		cfg.Image = tastoracontainer.NewImage("ethpandaops/spamoor", "latest", "")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	log := cfg.Logger.With(zap.String("component", "spamoor-daemon"), zap.Int("i", index))
	n := &spamoorBenchmarkNode{cfg: cfg, logger: log, name: name}
	n.Node = tastoracontainer.NewNode(cfg.DockerNetworkID, cfg.DockerClient, testName, cfg.Image, "/home/spamoor", index, spamoorBenchmarkNodeType(0), log)
	if err := n.CreateAndSetupVolume(ctx, n.Name()); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *spamoorBenchmarkNode) Name() string {
	if n.name != "" {
		return fmt.Sprintf("spamoor-%s-%d-%s", n.name, n.Index, sanitizeDockerResourceName(n.TestName))
	}
	return fmt.Sprintf("spamoor-%d-%s", n.Index, sanitizeDockerResourceName(n.TestName))
}

func (n *spamoorBenchmarkNode) HostName() string {
	return condenseHostName(n.Name())
}

func (n *spamoorBenchmarkNode) GetNetworkInfo(ctx context.Context) (tastoratypes.NetworkInfo, error) {
	internalIP, err := n.containerInternalIP(ctx)
	if err != nil {
		return tastoratypes.NetworkInfo{}, err
	}
	return tastoratypes.NetworkInfo{
		Internal: tastoratypes.Network{Hostname: n.HostName(), IP: internalIP, Ports: tastoratypes.Ports{HTTP: defaultSpamoorHTTPPort}},
		External: tastoratypes.Network{Hostname: "0.0.0.0", Ports: n.external},
	}, nil
}

func (n *spamoorBenchmarkNode) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.started {
		return n.startContainer(ctx)
	}
	if err := n.createContainer(ctx); err != nil {
		return err
	}
	if err := n.startContainer(ctx); err != nil {
		return err
	}

	mapped := defaultSpamoorHTTPPort
	if !n.cfg.HostNetwork {
		hostPort, err := n.hostPort(ctx, defaultSpamoorHTTPPort+"/tcp")
		if err != nil {
			return err
		}
		mapped, err = extractPort(hostPort)
		if err != nil {
			return err
		}
	}
	n.external = tastoratypes.Ports{HTTP: mapped}
	n.started = true
	waitHTTP("http://127.0.0.1:"+n.external.HTTP+"/metrics", 20*time.Second)
	return nil
}

func (n *spamoorBenchmarkNode) API() *spamoor.API {
	return spamoor.NewAPI("http://127.0.0.1:" + n.external.HTTP)
}

func (n *spamoorBenchmarkNode) Remove(ctx context.Context, opts ...tastoratypes.RemoveOption) error {
	removeOpts := mobyclient.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}
	for _, opt := range opts {
		opt(&removeOpts)
	}

	if n.containerID != "" {
		if err := n.stopContainer(ctx); err != nil {
			return err
		}
		if _, err := n.DockerClient.ContainerRemove(ctx, n.containerID, removeOpts); err != nil && !errdefs.IsNotFound(err) {
			return fmt.Errorf("remove container %s: %w", n.Name(), err)
		}
	}
	if removeOpts.RemoveVolumes && n.VolumeName != "" {
		if _, err := n.DockerClient.VolumeRemove(ctx, n.VolumeName, mobyclient.VolumeRemoveOptions{Force: true}); err != nil && !errdefs.IsNotFound(err) {
			return fmt.Errorf("remove volume %s: %w", n.VolumeName, err)
		}
	}
	return nil
}

func (n *spamoorBenchmarkNode) createContainer(ctx context.Context) error {
	imageRef := n.cfg.Image.Ref()
	if err := n.cfg.Image.PullImage(ctx, n.DockerClient); err != nil {
		return err
	}

	cmd := []string{
		"--privkey", n.cfg.PrivateKey,
		"--port", defaultSpamoorHTTPPort,
		"--db", fmt.Sprintf("%s/%s", n.HomeDir(), "spamoor.db"),
	}
	for _, host := range n.cfg.RPCHosts {
		if host = strings.TrimSpace(host); host != "" {
			cmd = append(cmd, "--rpchost", host)
		}
	}
	cmd = append(cmd, n.cfg.AdditionalStartArgs...)

	containerCfg := &mobycontainer.Config{
		Image:      imageRef,
		Entrypoint: []string{"/app/spamoor-daemon"},
		Cmd:        cmd,
		Hostname:   n.HostName(),
		Labels:     map[string]string{consts.CleanupLabel: n.DockerClient.CleanupLabel()},
	}
	hostCfg := &mobycontainer.HostConfig{
		Binds:      n.Bind(),
		AutoRemove: false,
		DNS:        []netip.Addr{},
	}
	var netCfg *mobynetwork.NetworkingConfig
	if n.cfg.HostNetwork {
		hostCfg.NetworkMode = mobycontainer.NetworkMode(mobynetwork.NetworkHost)
	} else {
		ports := mobynetwork.PortMap{
			mobynetwork.MustParsePort(defaultSpamoorHTTPPort + "/tcp"): {},
		}
		portBindings, listeners, err := port.GenerateBindings(ports)
		if err != nil {
			return fmt.Errorf("generate port bindings: %w", err)
		}
		n.preStartListeners = listeners
		containerCfg.ExposedPorts = portSet(ports)
		hostCfg.PortBindings = portBindings
		hostCfg.PublishAllPorts = true
		netCfg = &mobynetwork.NetworkingConfig{
			EndpointsConfig: map[string]*mobynetwork.EndpointSettings{
				n.NetworkID: &mobynetwork.EndpointSettings{},
			},
		}
	}

	n.logger.Info("Will run spamoor daemon", zap.String("image", imageRef), zap.String("container", n.Name()))
	created, err := n.DockerClient.ContainerCreate(ctx, mobyclient.ContainerCreateOptions{
		Name:             n.Name(),
		Config:           containerCfg,
		HostConfig:       hostCfg,
		NetworkingConfig: netCfg,
	})
	if err != nil {
		n.preStartListeners.CloseAll()
		n.preStartListeners = nil
		return err
	}
	n.containerID = created.ID
	return nil
}

func (n *spamoorBenchmarkNode) startContainer(ctx context.Context) error {
	n.preStartListeners.CloseAll()
	n.preStartListeners = nil
	if _, err := n.DockerClient.ContainerStart(ctx, n.containerID, mobyclient.ContainerStartOptions{}); err != nil {
		return err
	}
	n.logger.Info("Container started", zap.String("container", n.Name()))
	return nil
}

func (n *spamoorBenchmarkNode) stopContainer(ctx context.Context) error {
	timeoutSec := 30
	if _, err := n.DockerClient.ContainerStop(ctx, n.containerID, mobyclient.ContainerStopOptions{Timeout: &timeoutSec}); err != nil && !errdefs.IsNotFound(err) && !errdefs.IsNotModified(err) {
		return fmt.Errorf("stop container %s: %w", n.Name(), err)
	}
	return nil
}

func (n *spamoorBenchmarkNode) hostPort(ctx context.Context, portID string) (string, error) {
	inspectResult, err := n.DockerClient.ContainerInspect(ctx, n.containerID, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return "", err
	}
	return port.GetForHost(inspectResult.Container, portID), nil
}

func (n *spamoorBenchmarkNode) containerInternalIP(ctx context.Context) (string, error) {
	inspectResult, err := n.DockerClient.ContainerInspect(ctx, n.containerID, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspecting container: %w", err)
	}
	inspect := inspectResult.Container
	if inspect.NetworkSettings == nil || inspect.NetworkSettings.Networks == nil {
		return "", nil
	}
	for _, network := range inspect.NetworkSettings.Networks {
		if network.IPAddress.IsValid() {
			return network.IPAddress.String(), nil
		}
	}
	return "", nil
}

func portSet(ports mobynetwork.PortMap) mobynetwork.PortSet {
	portSet := mobynetwork.PortSet{}
	for port := range ports {
		portSet[port] = struct{}{}
	}
	return portSet
}

func waitHTTP(url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 500 {
			_ = resp.Body.Close()
			return
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

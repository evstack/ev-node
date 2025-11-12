package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// RaftClusterClient is simple http based client for raft cluster management
type RaftClusterClient struct {
	peers      []string
	httpClient *http.Client
}

func (r RaftClusterClient) AddPeer(ctx context.Context, id, addr string) error {
	val := struct {
		NodeID  string `json:"id"`
		Address string `json:"address"`
	}{
		NodeID:  id,
		Address: addr,
	}
	return r.broadcast(ctx, "/raft/join", val)
}

func (r RaftClusterClient) RemovePeer(ctx context.Context, id string) error {
	val := struct {
		NodeID string `json:"id"`
	}{
		NodeID: id,
	}
	return r.broadcast(ctx, "/raft/remove", val)
}

func (r RaftClusterClient) broadcast(ctx context.Context, path string, obj any) error {
	bz, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	var peerErrs []error
	for _, peer := range r.peers {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, peer+path, bytes.NewBuffer(bz))
		if err != nil {
			peerErrs = append(peerErrs, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := r.httpClient.Do(req)
		if err != nil {
			peerErrs = append(peerErrs, err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			peerErrs = append(peerErrs, fmt.Errorf("unexpected status: %s", resp.Status))
			continue
		}
	}
	if len(peerErrs) == len(r.peers) {
		return errors.Join(peerErrs...)
	}
	return nil
}

func NewRaftClusterClient(peers ...string) (RaftClusterClient, error) {
	return RaftClusterClient{
		peers:      peers,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}, nil
}

package raft

import (
	"errors"
	"testing"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitPeerAddr(t *testing.T) {
	specs := map[string]struct {
		in     string
		exp    raft.Server
		expErr error
	}{
		"valid": {
			in:  "node1@127.0.0.1:1234",
			exp: raft.Server{ID: raft.ServerID("node1"), Address: raft.ServerAddress("127.0.0.1:1234")},
		},
		"trims whitespace": {
			in:  "  node2  @  10.0.0.2:9000  ",
			exp: raft.Server{ID: raft.ServerID("node2"), Address: raft.ServerAddress("10.0.0.2:9000")},
		},
		"missing at": {
			in:     "node1",
			expErr: errors.New("expecting nodeID@address for peer"),
		},
		"empty node id": {
			in:     "@127.0.0.1:1234",
			expErr: errors.New("nodeID cannot be empty"),
		},
		"empty address": {
			in:     "node1@",
			expErr: errors.New("address cannot be empty"),
		},
		"multiple ats": {
			in:     "a@b@c",
			expErr: errors.New("expecting nodeID@address for peer"),
		},
		"only spaces": {
			in:     "   @   ",
			expErr: errors.New("nodeID cannot be empty"),
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			_ = ctx // keep to follow guideline to prefer t.Context; function under test doesn't use context

			got, err := splitPeerAddr(spec.in)
			if spec.expErr != nil {
				require.Error(t, err)
				assert.Equal(t, spec.expErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestDeduplicateServers(t *testing.T) {

	specs := map[string]struct {
		in  []raft.Server
		exp []raft.Server
	}{
		"empty": {
			in:  nil,
			exp: []raft.Server{},
		},
		"no duplicates": {
			in: []raft.Server{
				{ID: raft.ServerID("n1"), Address: raft.ServerAddress("a1")},
				{ID: raft.ServerID("n2"), Address: raft.ServerAddress("a2")},
			},
			exp: []raft.Server{
				{ID: raft.ServerID("n1"), Address: raft.ServerAddress("a1")},
				{ID: raft.ServerID("n2"), Address: raft.ServerAddress("a2")},
			},
		},
		"duplicates keep first": {
			in: []raft.Server{
				{ID: raft.ServerID("n1"), Address: raft.ServerAddress("a1")},
				{ID: raft.ServerID("n2"), Address: raft.ServerAddress("a2")},
				{ID: raft.ServerID("n1"), Address: raft.ServerAddress("a3")},
				{ID: raft.ServerID("n3"), Address: raft.ServerAddress("a4")},
				{ID: raft.ServerID("n2"), Address: raft.ServerAddress("a5")},
			},
			exp: []raft.Server{
				{ID: raft.ServerID("n1"), Address: raft.ServerAddress("a1")},
				{ID: raft.ServerID("n2"), Address: raft.ServerAddress("a2")},
				{ID: raft.ServerID("n3"), Address: raft.ServerAddress("a4")},
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			_ = ctx

			got := deduplicateServers(spec.in)
			assert.Equal(t, spec.exp, got)
		})
	}
}

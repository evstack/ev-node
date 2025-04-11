package config

import (
	"strings"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestTranslateAddresses(t *testing.T) {
	t.Parallel()

	invalidCosmos := "foobar"
	validCosmos := "127.0.0.1:1234"
	validRollkit := "/ip4/127.0.0.1/tcp/1234"

	cases := []struct {
		name        string
		input       Config
		expected    Config
		expectedErr string
	}{
		{"empty", Config{}, Config{}, ""},
		{
			"valid listen address",
			Config{P2P: P2PConfig{ListenAddress: validCosmos}},
			Config{P2P: P2PConfig{ListenAddress: validRollkit}},
			"",
		},
		{
			"valid seed address",
			Config{P2P: P2PConfig{Peers: validCosmos + "," + validCosmos}},
			Config{P2P: P2PConfig{Peers: validRollkit + "," + validRollkit}},
			"",
		},
		{
			"invalid listen address",
			Config{P2P: P2PConfig{ListenAddress: invalidCosmos}},
			Config{},
			errInvalidAddress.Error(),
		},
		{
			"invalid seed address",
			Config{P2P: P2PConfig{Peers: validCosmos + "," + invalidCosmos}},
			Config{},
			errInvalidAddress.Error(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			// input is changed in place
			input := c.input
			err := TranslateAddresses(&input)
			if c.expectedErr != "" {
				assert.Error(err)
				assert.True(strings.HasPrefix(err.Error(), c.expectedErr), "invalid error message")
			} else {
				assert.NoError(err)
				assert.Equal(c.expected, input)
			}
		})
	}
}

func TestGetMultiaddr(t *testing.T) {
	t.Parallel()

	valid := mustGetMultiaddr(t, "/ip4/127.0.0.1/tcp/1234")
	withID := mustGetMultiaddr(t, "/ip4/127.0.0.1/tcp/1234/p2p/k2k4r8oqamigqdo6o7hsbfwd45y70oyynp98usk7zmyfrzpqxh1pohl7")
	udpWithID := mustGetMultiaddr(t, "/ip4/127.0.0.1/udp/1234/p2p/k2k4r8oqamigqdo6o7hsbfwd45y70oyynp98usk7zmyfrzpqxh1pohl7")

	cases := []struct {
		name        string
		input       string
		expected    multiaddr.Multiaddr
		expectedErr string
	}{
		{"empty", "", nil, errInvalidAddress.Error()},
		{"no port", "127.0.0.1:", nil, "failed to parse multiaddr"},
		{"ip only", "127.0.0.1", nil, errInvalidAddress.Error()},
		{"with invalid id", "deadbeef@127.0.0.1:1234", nil, "failed to parse multiaddr"},
		{"valid", "127.0.0.1:1234", valid, ""},
		{"valid with id", "k2k4r8oqamigqdo6o7hsbfwd45y70oyynp98usk7zmyfrzpqxh1pohl7@127.0.0.1:1234", withID, ""},
		{"valid with id and proto", "udp://k2k4r8oqamigqdo6o7hsbfwd45y70oyynp98usk7zmyfrzpqxh1pohl7@127.0.0.1:1234", udpWithID, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := GetMultiAddr(c.input)
			if c.expectedErr != "" {
				assert.Error(err)
				assert.Nil(actual)
				assert.True(strings.HasPrefix(err.Error(), c.expectedErr), "invalid error message")
			} else {
				assert.NoError(err)
				assert.Equal(c.expected, actual)
			}
		})
	}
}

func mustGetMultiaddr(t *testing.T, addr string) multiaddr.Multiaddr {
	t.Helper()
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		t.Fatal(err)
	}
	return maddr
}

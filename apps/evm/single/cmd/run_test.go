package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
)

func TestFindNamespaceForHeight(t *testing.T) {
	// Create test namespaces
	currentNamespace := da.NamespaceFromString("current-ns")
	currentDataNamespace := da.NamespaceFromString("current-data-ns")

	migration1 := namespaces{
		namespace:     "migration1-ns",
		dataNamespace: "migration1-data-ns",
	}
	migration2 := namespaces{
		namespace:     "migration2-ns",
		dataNamespace: "migration2-data-ns",
	}
	migration3 := namespaces{
		namespace:     "migration3-ns",
		dataNamespace: "", // Empty data namespace should fall back to namespace
	}

	tests := []struct {
		name            string
		migrations      map[uint64]namespaces
		height          uint64
		isDataNamespace bool
		expectedNS      []byte
		description     string
	}{
		{
			name:            "no migrations - regular namespace",
			migrations:      map[uint64]namespaces{},
			height:          100,
			isDataNamespace: false,
			expectedNS:      currentNamespace.Bytes(),
			description:     "Should return current namespace when no migrations exist",
		},
		{
			name:            "no migrations - data namespace",
			migrations:      map[uint64]namespaces{},
			height:          100,
			isDataNamespace: true,
			expectedNS:      currentDataNamespace.Bytes(),
			description:     "Should return current data namespace when no migrations exist",
		},
		{
			name: "height within migration range - regular namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
			},
			height:          50,
			isDataNamespace: false,
			expectedNS:      da.NamespaceFromString(migration1.GetNamespace()).Bytes(),
			description:     "Should return migration namespace when height is within migration range (0-100)",
		},
		{
			name: "height within migration range - data namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
			},
			height:          50,
			isDataNamespace: true,
			expectedNS:      da.NamespaceFromString(migration1.GetDataNamespace()).Bytes(),
			description:     "Should return migration data namespace when height is within migration range (0-100)",
		},
		{
			name: "height at migration untilHeight - regular namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
			},
			height:          100,
			isDataNamespace: false,
			expectedNS:      da.NamespaceFromString(migration1.GetNamespace()).Bytes(),
			description:     "Should return migration namespace when height equals untilHeight (inclusive)",
		},
		{
			name: "height at migration untilHeight - data namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
			},
			height:          100,
			isDataNamespace: true,
			expectedNS:      da.NamespaceFromString(migration1.GetDataNamespace()).Bytes(),
			description:     "Should return migration data namespace when height equals untilHeight (inclusive)",
		},
		{
			name: "height after migration range - regular namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
			},
			height:          150,
			isDataNamespace: false,
			expectedNS:      currentNamespace.Bytes(),
			description:     "Should return current namespace when height is after all migration ranges",
		},
		{
			name: "height after migration range - data namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
			},
			height:          150,
			isDataNamespace: true,
			expectedNS:      currentDataNamespace.Bytes(),
			description:     "Should return current data namespace when height is after all migration ranges",
		},
		{
			name: "multiple migrations - height in middle range",
			migrations: map[uint64]namespaces{
				100: migration1,
				200: migration2,
				300: migration3,
			},
			height:          150,
			isDataNamespace: false,
			expectedNS:      da.NamespaceFromString(migration2.GetNamespace()).Bytes(),
			description:     "Should use migration2 namespace for height 150 (within range 101-200)",
		},
		{
			name: "multiple migrations - height in middle range data namespace",
			migrations: map[uint64]namespaces{
				100: migration1,
				200: migration2,
				300: migration3,
			},
			height:          150,
			isDataNamespace: true,
			expectedNS:      da.NamespaceFromString(migration2.GetDataNamespace()).Bytes(),
			description:     "Should use migration2 data namespace for height 150 (within range 101-200)",
		},
		{
			name: "multiple migrations - height in first range",
			migrations: map[uint64]namespaces{
				100: migration1,
				200: migration2,
				300: migration3,
			},
			height:          50,
			isDataNamespace: false,
			expectedNS:      da.NamespaceFromString(migration1.GetNamespace()).Bytes(),
			description:     "Should use migration1 namespace for height 50 (within range 0-100)",
		},
		{
			name: "multiple migrations - height at exact untilHeight",
			migrations: map[uint64]namespaces{
				100: migration1,
				200: migration2,
				300: migration3,
			},
			height:          200,
			isDataNamespace: false,
			expectedNS:      da.NamespaceFromString(migration2.GetNamespace()).Bytes(),
			description:     "Should use migration2 namespace for height 200 (at untilHeight boundary)",
		},
		{
			name: "empty data namespace falls back to namespace",
			migrations: map[uint64]namespaces{
				100: migration3, // Has empty dataNamespace
			},
			height:          50,
			isDataNamespace: true,
			expectedNS:      da.NamespaceFromString(migration3.GetNamespace()).Bytes(),
			description:     "Should fall back to namespace when dataNamespace is empty",
		},
		{
			name: "edge case - height 0 with untilHeight 0",
			migrations: map[uint64]namespaces{
				0:   migration1,
				100: migration2,
			},
			height:          0,
			isDataNamespace: false,
			expectedNS:      da.NamespaceFromString("migration1-ns").Bytes(),
			description:     "Should handle height 0 with untilHeight 0 correctly",
		},
		{
			name: "edge case - height after all migrations",
			migrations: map[uint64]namespaces{
				100: migration1,
				200: migration2,
			},
			height:          ^uint64(0), // Maximum uint64
			isDataNamespace: false,
			expectedNS:      currentNamespace.Bytes(),
			description:     "Should return current namespace when height exceeds all migration ranges",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the API wrapper directly with known current namespaces
			api := &namespaceMigrationDAAPI{
				API:                  jsonrpc.API{},
				migrations:           tt.migrations,
				currentNamespace:     currentNamespace.Bytes(),
				currentDataNamespace: currentDataNamespace.Bytes(),
			}

			// Call the method under test
			result := api.findNamespaceForHeight(tt.height, tt.isDataNamespace)

			// Assert the result
			require.Equal(t, tt.expectedNS, result, tt.description)
		})
	}
}

func TestParseMigrations(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   string
		expected    map[uint64]namespaces
		expectError bool
		description string
	}{
		{
			name:        "empty flag",
			flagValue:   "",
			expected:    map[uint64]namespaces{},
			description: "Should return empty map when flag is empty",
		},
		{
			name:      "single migration with data namespace",
			flagValue: "100:ns1:dns1",
			expected: map[uint64]namespaces{
				100: {namespace: "ns1", dataNamespace: "dns1"},
			},
			description: "Should parse single migration with data namespace (untilHeight=100)",
		},
		{
			name:      "single migration without data namespace",
			flagValue: "100:ns1",
			expected: map[uint64]namespaces{
				100: {namespace: "ns1", dataNamespace: ""},
			},
			description: "Should parse single migration without data namespace (untilHeight=100)",
		},
		{
			name:      "multiple migrations",
			flagValue: "100:ns1:dns1,200:ns2:dns2,300:ns3",
			expected: map[uint64]namespaces{
				100: {namespace: "ns1", dataNamespace: "dns1"},
				200: {namespace: "ns2", dataNamespace: "dns2"},
				300: {namespace: "ns3", dataNamespace: ""},
			},
			description: "Should parse multiple migrations correctly with untilHeights",
		},
		{
			name:        "invalid format - missing namespace",
			flagValue:   "100:",
			expectError: true,
			description: "Should fail when namespace is missing",
		},
		{
			name:        "invalid format - missing untilHeight",
			flagValue:   ":ns1",
			expectError: true,
			description: "Should fail when untilHeight is missing",
		},
		{
			name:        "invalid format - no colon",
			flagValue:   "100ns1",
			expectError: true,
			description: "Should fail when format is completely wrong",
		},
		{
			name:        "invalid format - too many parts",
			flagValue:   "100:ns1:dns1:extra",
			expectError: true,
			description: "Should fail when there are too many parts",
		},
		{
			name:        "invalid untilHeight - not a number",
			flagValue:   "abc:ns1",
			expectError: true,
			description: "Should fail when untilHeight is not a valid number",
		},
		{
			name:        "invalid untilHeight - negative",
			flagValue:   "-100:ns1",
			expectError: true,
			description: "Should fail when untilHeight is negative",
		},
		{
			name:      "whitespace handling",
			flagValue: " 100:ns1:dns1 , 200:ns2 ",
			expected: map[uint64]namespaces{
				100: {namespace: "ns1", dataNamespace: "dns1"},
				200: {namespace: "ns2", dataNamespace: ""},
			},
			description: "Should handle whitespace correctly",
		},
		{
			name:      "untilHeight zero",
			flagValue: "0:genesis-ns",
			expected: map[uint64]namespaces{
				0: {namespace: "genesis-ns", dataNamespace: ""},
			},
			description: "Should handle untilHeight 0 correctly",
		},
		{
			name:      "very high untilHeight",
			flagValue: "18446744073709551615:max-ns", // Max uint64
			expected: map[uint64]namespaces{
				18446744073709551615: {namespace: "max-ns", dataNamespace: ""},
			},
			description: "Should handle maximum uint64 untilHeight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock command with the flag value
			cmd := &cobra.Command{}
			cmd.Flags().String("migrations", "", "test flag")
			err := cmd.Flags().Set("migrations", tt.flagValue)
			require.NoError(t, err)

			// Call the function under test
			result, err := parseMigrations(cmd)

			if tt.expectError {
				require.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}

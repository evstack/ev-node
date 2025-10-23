package submitting

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSubmitOptions_NoSigningAddress(t *testing.T) {
	baseOptions := []byte(`{"key":"value"}`)

	result, err := mergeSubmitOptions(baseOptions, "")
	require.NoError(t, err)
	assert.Equal(t, baseOptions, result, "should return unchanged options when no signing address")
}

func TestMergeSubmitOptions_EmptyBaseOptions(t *testing.T) {
	signingAddress := "celestia1abc123"

	result, err := mergeSubmitOptions([]byte{}, signingAddress)
	require.NoError(t, err)

	var resultMap map[string]interface{}
	err = json.Unmarshal(result, &resultMap)
	require.NoError(t, err)

	assert.Equal(t, signingAddress, resultMap["signer_address"])
}

func TestMergeSubmitOptions_ValidJSON(t *testing.T) {
	baseOptions := []byte(`{"existing":"option","number":42}`)
	signingAddress := "celestia1def456"

	result, err := mergeSubmitOptions(baseOptions, signingAddress)
	require.NoError(t, err)

	var resultMap map[string]interface{}
	err = json.Unmarshal(result, &resultMap)
	require.NoError(t, err)

	assert.Equal(t, "option", resultMap["existing"])
	assert.Equal(t, float64(42), resultMap["number"]) // JSON numbers are float64
	assert.Equal(t, signingAddress, resultMap["signer_address"])
}

func TestMergeSubmitOptions_InvalidJSON(t *testing.T) {
	baseOptions := []byte(`not-json-content`)
	signingAddress := "celestia1ghi789"

	result, err := mergeSubmitOptions(baseOptions, signingAddress)
	require.NoError(t, err)

	var resultMap map[string]interface{}
	err = json.Unmarshal(result, &resultMap)
	require.NoError(t, err)

	// Should create new JSON object with just the signing address
	assert.Equal(t, signingAddress, resultMap["signer_address"])
	assert.Len(t, resultMap, 1, "should only contain signing address when base options are invalid JSON")
}

func TestMergeSubmitOptions_OverrideExistingAddress(t *testing.T) {
	baseOptions := []byte(`{"signer_address":"old-address","other":"data"}`)
	newAddress := "celestia1new456"

	result, err := mergeSubmitOptions(baseOptions, newAddress)
	require.NoError(t, err)

	var resultMap map[string]interface{}
	err = json.Unmarshal(result, &resultMap)
	require.NoError(t, err)

	assert.Equal(t, newAddress, resultMap["signer_address"], "should override existing signing address")
	assert.Equal(t, "data", resultMap["other"])
}

func TestMergeSubmitOptions_NilBaseOptions(t *testing.T) {
	signingAddress := "celestia1jkl012"

	result, err := mergeSubmitOptions(nil, signingAddress)
	require.NoError(t, err)

	var resultMap map[string]interface{}
	err = json.Unmarshal(result, &resultMap)
	require.NoError(t, err)

	assert.Equal(t, signingAddress, resultMap["signer_address"])
}

func TestMergeSubmitOptions_ComplexJSON(t *testing.T) {
	baseOptions := []byte(`{
		"nested": {
			"key": "value"
		},
		"array": [1, 2, 3],
		"bool": true
	}`)
	signingAddress := "celestia1complex"

	result, err := mergeSubmitOptions(baseOptions, signingAddress)
	require.NoError(t, err)

	var resultMap map[string]interface{}
	err = json.Unmarshal(result, &resultMap)
	require.NoError(t, err)

	// Check nested structure is preserved
	nested, ok := resultMap["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", nested["key"])

	// Check array is preserved
	array, ok := resultMap["array"].([]interface{})
	require.True(t, ok)
	assert.Len(t, array, 3)

	// Check bool is preserved
	assert.Equal(t, true, resultMap["bool"])

	// Check signing address was added
	assert.Equal(t, signingAddress, resultMap["signer_address"])
}

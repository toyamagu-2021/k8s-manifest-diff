package e2e

import (
	"testing"
)

func TestSecretMasking(t *testing.T) {
	baseFile := getFixturePath("basic", "secret-with-data-base.yaml")
	headFile := getFixturePath("basic", "secret-with-data-head.yaml")

	result := runDiffCommand("diff", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header in the expected format
	assertDiffOutput(t, result, []string{
		"===== /Secret default/test-secret ======",
	})

	// Verify that actual secret values are masked in the diff output
	assertNotInOutput(t, result, []string{
		"bXlwYXNzd29yZA==", // base64 encoded "mypassword"
		"bmV3cGFzc3dvcmQ=", // base64 encoded "newpassword"
		"dGVzdC1hcGkta2V5", // base64 encoded "test-api-key"
		"YWJjZGVmZ2hpams=", // base64 encoded "abcdefghijk"
		"bmV3c2VjcmV0",     // base64 encoded "newsecret"
	})

	// Should show that the data field has changes but with masked values (+ symbols)
	assertDiffOutput(t, result, []string{
		"data:",
		"++++++++++++++++",
	})
}

func TestSecretMaskingWithStringData(t *testing.T) {
	// Create test files with stringData
	baseFile := getFixturePath("basic", "secret-with-stringdata-base.yaml")
	headFile := getFixturePath("basic", "secret-with-stringdata-head.yaml")

	result := runDiffCommand("diff", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header in the expected format
	assertDiffOutput(t, result, []string{
		"===== /Secret default/test-secret-string ======",
	})

	// Verify that actual stringData values are masked
	assertNotInOutput(t, result, []string{
		"plaintext-password",
		"new-plaintext-password",
		"my-api-key",
		"another-secret-value",
	})

	// Should show that the stringData field has changes but with masked values (+ symbols)
	assertDiffOutput(t, result, []string{
		"stringData:",
		"++++++++++++++++",
	})
}

func TestSecretMaskingMixedDataTypes(t *testing.T) {
	// Test with both data and stringData fields
	baseFile := getFixturePath("basic", "secret-mixed-base.yaml")
	headFile := getFixturePath("basic", "secret-mixed-head.yaml")

	result := runDiffCommand("diff", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header in the expected format
	assertDiffOutput(t, result, []string{
		"===== /Secret default/mixed-secret ======",
	})

	// Verify that both data and stringData values are masked
	assertNotInOutput(t, result, []string{
		"bXlwYXNzd29yZA==",  // base64 encoded "mypassword"
		"plaintext-api-key", // stringData value
		"bmV3cGFzc3dvcmQ=",  // base64 encoded "newpassword"
		"new-plaintext-key", // stringData value
	})

	// Should show both fields with masked values (+ symbols)
	assertDiffOutput(t, result, []string{
		"data:",
		"stringData:",
		"++++++++++++++++",
	})
}

func TestDisableMaskingSecretFlag(t *testing.T) {
	baseFile := getFixturePath("basic", "secret-with-data-base.yaml")
	headFile := getFixturePath("basic", "secret-with-data-head.yaml")

	// Test with --disable-masking-secret flag
	result := runDiffCommand("diff", "--disable-masking-secret", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header in the expected format
	assertDiffOutput(t, result, []string{
		"===== /Secret default/test-secret ======",
	})

	// Verify that actual secret values are NOT masked (should show actual base64 values)
	assertDiffOutput(t, result, []string{
		"bXlwYXNzd29yZA==", // base64 encoded "mypassword" (from base)
		"bmV3cGFzc3dvcmQ=", // base64 encoded "newpassword" (from head)
		"dGVzdC1hcGkta2V5", // base64 encoded "test-api-key"
		"YWJjZGVmZ2hpams=", // base64 encoded "abcdefghijk"
		"bmV3c2VjcmV0",     // base64 encoded "newsecret"
	})

	// Should NOT contain masked values (+ symbols)
	assertNotInOutput(t, result, []string{
		"++++++++++++++++",
		"+++++++++++++++++",
		"++++++++++++++++++",
	})
}

func TestDisableMaskingSecretWithStringData(t *testing.T) {
	baseFile := getFixturePath("basic", "secret-with-stringdata-base.yaml")
	headFile := getFixturePath("basic", "secret-with-stringdata-head.yaml")

	// Test with --disable-masking-secret flag
	result := runDiffCommand("diff", "--disable-masking-secret", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header in the expected format
	assertDiffOutput(t, result, []string{
		"===== /Secret default/test-secret-string ======",
	})

	// Verify that actual stringData values are NOT masked (should show plain text values)
	assertDiffOutput(t, result, []string{
		"plaintext-password",
		"new-plaintext-password",
		"my-api-key",
		"another-secret-value",
	})

	// Should NOT contain masked values (+ symbols)
	assertNotInOutput(t, result, []string{
		"++++++++++++++++",
		"+++++++++++++++++",
		"++++++++++++++++++",
	})
}

func TestDisableMaskingSecretMixedDataTypes(t *testing.T) {
	baseFile := getFixturePath("basic", "secret-mixed-base.yaml")
	headFile := getFixturePath("basic", "secret-mixed-head.yaml")

	// Test with --disable-masking-secret flag
	result := runDiffCommand("diff", "--disable-masking-secret", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header in the expected format
	assertDiffOutput(t, result, []string{
		"===== /Secret default/mixed-secret ======",
	})

	// Verify that both data and stringData values are NOT masked
	assertDiffOutput(t, result, []string{
		"bXlwYXNzd29yZA==",  // base64 encoded "mypassword"
		"plaintext-api-key", // stringData value
		"bmV3cGFzc3dvcmQ=",  // base64 encoded "newpassword"
		"new-plaintext-key", // stringData value
	})

	// Should NOT contain masked values (+ symbols)
	assertNotInOutput(t, result, []string{
		"++++++++++++++++",
		"+++++++++++++++++",
		"++++++++++++++++++",
	})
}

func TestMaskingSecretEnabledByDefault(t *testing.T) {
	baseFile := getFixturePath("basic", "secret-with-data-base.yaml")
	headFile := getFixturePath("basic", "secret-with-data-head.yaml")

	// Test without --disable-masking-secret flag (should be masked by default)
	result := runDiffCommand("diff", baseFile, headFile)

	// Should have differences (exit code 1)
	assertHasDiff(t, result)

	// Should contain the Secret resource header
	assertDiffOutput(t, result, []string{
		"===== /Secret default/test-secret ======",
	})

	// Verify that secret values are masked by default
	assertNotInOutput(t, result, []string{
		"bXlwYXNzd29yZA==", // base64 encoded "mypassword"
		"bmV3cGFzc3dvcmQ=", // base64 encoded "newpassword"
		"dGVzdC1hcGkta2V5", // base64 encoded "test-api-key"
		"YWJjZGVmZ2hpams=", // base64 encoded "abcdefghijk"
		"bmV3c2VjcmV0",     // base64 encoded "newsecret"
	})

	// Should contain masked values (+ symbols)
	assertDiffOutput(t, result, []string{
		"++++++++++++++++",
	})
}

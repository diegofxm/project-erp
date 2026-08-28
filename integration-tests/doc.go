// Package integrationtest holds cofacture's real-DIAN integration tests — the ones that build,
// sign and submit an actual document to DIAN's certification (habilitación) environment,
// instead of asserting against a mocked server or a golden file.
//
// This module lives outside github.com/diegofxm/cofacture on purpose and is never published
// alongside it: these tests need a real DIAN certificate, real NIT/software credentials, and in
// several cases embed real personal/business data from the certification account they were
// developed against. None of that belongs in the public library.
//
// It depends on cofacture only through its public API (see the "replace" directive in go.mod
// pointing at ../cofacture) — exactly like any other consumer of the library would.
//
// Every test here is skipped unless the COFACTURE_TEST_FIXTURES_DIR environment variable points
// at a local, git-ignored directory containing:
//
//	certificado_cert.pem   the certification-environment certificate
//	certificado_key.pem    its private key
//	credenciales.txt       KEY=VALUE pairs (see setup() in helpers_test.go for the full list)
//
// Run the full suite with:
//
//	COFACTURE_TEST_FIXTURES_DIR=/path/to/your/fixtures go test ./...
package integrationtest

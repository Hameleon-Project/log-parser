package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferPortLinks_SampleCSV(t *testing.T) {
	b, err := os.ReadFile("data/ibdiagnet2.db_csv")
	if err != nil {
		t.Skip("no sample data")
	}
	parsed, err := parseIBDiagExport(b)
	assert.NoError(t, err)
	links := InferPortLinks(parsed)
	assert.GreaterOrEqual(t, len(links), 5, "expect host↔switch latency link + 4-switch ISL ring")

	kinds := map[string]int{}
	for _, l := range links {
		kinds[l.Kind]++
	}
	assert.GreaterOrEqual(t, kinds["link_round_trip_latency:270"], 1)
	assert.GreaterOrEqual(t, kinds["switch_isl_ring_port65"], 4)
}

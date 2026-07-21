package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Run(
		"Должен найти все нарушения в пакете a", func(t *testing.T) {
			testdata := analysistest.TestData()
			results := analysistest.Run(t, testdata, Analyzer, "a")

			require.NotEmpty(t, results)
			assert.Len(t, results, 1)
		},
	)

	t.Run(
		"Должен найти нарушения в main-пакете, кроме вызовов в main()", func(t *testing.T) {
			testdata := analysistest.TestData()
			results := analysistest.Run(t, testdata, Analyzer, "b")

			require.NotEmpty(t, results)
			assert.Len(t, results, 1)
		},
	)

	t.Run(
		"Должен пропустить _test.go файлы", func(t *testing.T) {
			testdata := analysistest.TestData()
			results := analysistest.Run(t, testdata, Analyzer, "c")

			for _, r := range results {
				assert.Empty(t, r.Diagnostics)
			}
		},
	)

	t.Run(
		"Не должен находить нарушения в чистом коде", func(t *testing.T) {
			testdata := analysistest.TestData()
			results := analysistest.Run(t, testdata, Analyzer, "d")

			for _, r := range results {
				assert.Empty(t, r.Diagnostics)
			}
		},
	)
}

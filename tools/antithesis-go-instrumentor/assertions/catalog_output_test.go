package assertions

import (
	"os"
	"path/filepath"
	"testing"

	qt "github.com/go-quicktest/qt"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

func TestCatalogContents(t *testing.T) {
	outputDir := t.TempDir()

	expects := []*AntExpect{
		{
			AssertionFuncInfo: &AssertionFuncInfo{
				TargetFunc: "Always",
				AssertType: "always",
				MustHit:    true,
				Condition:  false,
				MessageArg: 1,
			},
			Assertion: "Always",
			Message:   "test always",
			Classname: "example.com/mymod",
			Funcname:  "main",
			Filename:  "main.go",
			Line:      10,
		},
		{
			AssertionFuncInfo: &AssertionFuncInfo{
				TargetFunc: "Reachable",
				AssertType: "reachability",
				MustHit:    true,
				Condition:  true,
				MessageArg: 0,
			},
			Assertion: "Reachable",
			Message:   "test reachable",
			Classname: "example.com/mymod",
			Funcname:  "init",
			Filename:  "main.go",
			Line:      5,
		},
	}

	genInfo := GenInfo{
		ExpectedVals:        expects,
		NumericGuidanceVals: nil,
		BooleanGuidanceVals: nil,
		AssertPackageName:   common.AssertPackageName(),
		VersionText:         "test version",
		CreateDate:          "Mon Jan 1 00:00:00 UTC 2025",
		HasAssertions:       true,
		HasNumericGuidance:  false,
		HasBooleanGuidance:  false,
		ConstMap:            getConstMap(expects),
		logWriter:           common.GetLogWriter(),
	}

	GenerateAssertionsCatalog(outputDir, &genInfo)

	// Verify the file was created
	outputPath := filepath.Join(outputDir, common.GENERATED_CATALOG_FILE)
	content, err := os.ReadFile(outputPath)
	qt.Assert(t, qt.IsNil(err))

	text := string(content)

	qt.Check(t, qt.StringContains(text, "package main"))
	qt.Check(t, qt.StringContains(text, `import "github.com/antithesishq/antithesis-sdk-go/assert"`))
	qt.Check(t, qt.StringContains(text, `assert.AssertRaw(`))
	qt.Check(t, qt.StringContains(text, `"test always"`))
	qt.Check(t, qt.StringContains(text, `"test reachable"`))
	qt.Check(t, qt.StringContains(text, "test version"))
}

func TestCatalogNumericGuidance(t *testing.T) {
	outputDir := t.TempDir()

	numericGuidance := []*AntGuidance{
		{
			GuidanceFuncInfo: &GuidanceFuncInfo{
				AssertionFuncInfo: AssertionFuncInfo{
					TargetFunc: "AlwaysGreaterThan",
					AssertType: "always",
					MustHit:    true,
					Condition:  false,
					MessageArg: 2,
				},
				GuidanceFn: GuidanceFnMinimize,
			},
			Assertion: "AlwaysGreaterThan",
			Message:   "x > y",
			Classname: "example.com/mymod",
			Funcname:  "compute",
			Filename:  "compute.go",
			Line:      42,
		},
	}

	genInfo := GenInfo{
		ExpectedVals:        nil,
		NumericGuidanceVals: numericGuidance,
		BooleanGuidanceVals: nil,
		AssertPackageName:   common.AssertPackageName(),
		VersionText:         "test",
		CreateDate:          "now",
		HasAssertions:       false,
		HasNumericGuidance:  true,
		HasBooleanGuidance:  false,
		ConstMap:            make(map[string]bool),
		logWriter:           common.GetLogWriter(),
	}

	GenerateAssertionsCatalog(outputDir, &genInfo)

	outputPath := filepath.Join(outputDir, common.GENERATED_CATALOG_FILE)
	content, err := os.ReadFile(outputPath)
	qt.Assert(t, qt.IsNil(err))

	text := string(content)
	qt.Check(t, qt.StringContains(text, "assert.NumericGuidanceRaw("))
	qt.Check(t, qt.StringContains(text, `"x > y"`))
}

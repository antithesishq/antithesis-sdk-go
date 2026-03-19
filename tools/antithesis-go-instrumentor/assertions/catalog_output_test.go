package assertions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if err != nil {
		t.Fatalf("failed to read generated catalog: %v", err)
	}

	text := string(content)

	// Verify package declaration
	if !strings.Contains(text, "package main") {
		t.Error("catalog should contain 'package main'")
	}

	// Verify import
	if !strings.Contains(text, `import "github.com/antithesishq/antithesis-sdk-go/assert"`) {
		t.Error("catalog should import assert package")
	}

	// Verify assertion calls
	if !strings.Contains(text, `assert.AssertRaw(`) {
		t.Error("catalog should contain AssertRaw calls")
	}

	if !strings.Contains(text, `"test always"`) {
		t.Error("catalog should contain the 'test always' message")
	}

	if !strings.Contains(text, `"test reachable"`) {
		t.Error("catalog should contain the 'test reachable' message")
	}

	// Verify version text is included
	if !strings.Contains(text, "test version") {
		t.Error("catalog should contain version text")
	}
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
	if err != nil {
		t.Fatalf("failed to read generated catalog: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "assert.NumericGuidanceRaw(") {
		t.Error("catalog should contain NumericGuidanceRaw call")
	}
	if !strings.Contains(text, `"x > y"`) {
		t.Error("catalog should contain guidance message")
	}
}

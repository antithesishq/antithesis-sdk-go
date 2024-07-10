package assertions

import (
	"github.com/antithesishq/antithesis-sdk-go/assert"
)

// --------------------------------------------------------------------------------
// Assertion Hints
// --------------------------------------------------------------------------------
type AssertionFuncInfo struct {
	TargetFunc string
	AssertType string
	MustHit    bool
	Condition  bool
	MessageArg int
}

type AssertionHints map[string]*AssertionFuncInfo

func SetupHintMap() AssertionHints {
	hintMap := make(AssertionHints)

	hintMap["Always"] = &AssertionFuncInfo{
		TargetFunc: "Always",
		MustHit:    true,
		AssertType: "always",
		Condition:  false,
		MessageArg: 1,
	}

	hintMap["AlwaysOrUnreachable"] = &AssertionFuncInfo{
		TargetFunc: "AlwaysOrUnreachable",
		MustHit:    false,
		AssertType: "always",
		Condition:  false,
		MessageArg: 1,
	}

	hintMap["Sometimes"] = &AssertionFuncInfo{
		TargetFunc: "Sometimes",
		MustHit:    true,
		AssertType: "sometimes",
		Condition:  false,
		MessageArg: 1,
	}

	hintMap["Unreachable"] = &AssertionFuncInfo{
		TargetFunc: "Unreachable",
		MustHit:    false,
		AssertType: "reachability",
		Condition:  false,
		MessageArg: 0,
	}

	hintMap["Reachable"] = &AssertionFuncInfo{
		TargetFunc: "Reachable",
		MustHit:    true,
		AssertType: "reachability",
		Condition:  true,
		MessageArg: 0,
	}

	return hintMap
}

// --------------------------------------------------------------------------------
// Guidance Hints
// --------------------------------------------------------------------------------
type GuidanceFuncInfo struct {
	AssertionFuncInfo
	Guidepost assert.GuidepostType
}

type GuidanceHints map[string]*GuidanceFuncInfo

func SetupGuidanceHintMap() GuidanceHints {
	hintMap := make(GuidanceHints)

	hintMap["AlwaysGreaterThan"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "AlwaysGreaterThan",
			AssertType: "always",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMinimize,
	}

	hintMap["AlwaysGreaterThanOrEqualTo"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "AlwaysGreaterThanOrEqualTo",
			AssertType: "always",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMinimize,
	}

	hintMap["SometimesGreaterThan"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "SometimesGreaterThan",
			AssertType: "sometimes",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMaximize,
	}

	hintMap["SometimesGreaterThanOrEqualTo"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "SometimesGreaterThanOrEqualTo",
			AssertType: "sometimes",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMaximize,
	}

	hintMap["AlwaysLessThan"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "AlwaysLessThan",
			AssertType: "always",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMaximize,
	}

	hintMap["AlwaysLessThanOrEqualTo"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "AlwaysLessThanOrEqualTo",
			AssertType: "always",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMaximize,
	}

	hintMap["SometimesLessThan"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "SometimesLessThan",
			AssertType: "sometimes",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMinimize,
	}

	hintMap["SometimesLessThanOrEqualTo"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "SometimesLessThanOrEqualTo",
			AssertType: "sometimes",
			MustHit:    true,
			Condition:  false,
			MessageArg: 2,
		},
		Guidepost: assert.GuidepostMinimize,
	}

	hintMap["AlwaysSome"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "AlwaysSome",
			AssertType: "always",
			MustHit:    true,
			Condition:  false,
			MessageArg: 1,
		},
		Guidepost: assert.GuidepostNone,
	}

	hintMap["SometimesAll"] = &GuidanceFuncInfo{
		AssertionFuncInfo: AssertionFuncInfo{
			TargetFunc: "SometimesAll",
			AssertType: "sometimes",
			MustHit:    true,
			Condition:  false,
			MessageArg: 1,
		},
		Guidepost: assert.GuidepostAll,
	}

	return hintMap
}

func (m AssertionHints) HintsForName(name string) *AssertionFuncInfo {
	if v, ok := m[name]; ok {
		return v
	}
	return nil
}

func (m GuidanceHints) GuidanceHintsForName(name string) *GuidanceFuncInfo {
	if v, ok := m[name]; ok {
		return v
	}
	return nil
}

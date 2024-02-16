package main

type AssertionFuncInfo struct {
	TargetFunc string
	MustHit    bool
	Expecting  bool
	AssertType string
	Condition  bool
}

type AssertionHints map[string]*AssertionFuncInfo

func SetupHintMap() AssertionHints {
	hintMap := make(AssertionHints)

	hintMap["Always"] = &AssertionFuncInfo{
		TargetFunc: "Always",
		MustHit:    true,
		Expecting:  true,
		AssertType: "every",
		Condition:  false,
	}

	hintMap["AlwaysOrUnreachable"] = &AssertionFuncInfo{
		TargetFunc: "AlwaysOrUnreachable",
		MustHit:    false,
		Expecting:  true,
		AssertType: "every",
		Condition:  false,
	}

	hintMap["Sometimes"] = &AssertionFuncInfo{
		TargetFunc: "Sometimes",
		MustHit:    true,
		Expecting:  true,
		AssertType: "some",
		Condition:  false,
	}

	hintMap["Unreachable"] = &AssertionFuncInfo{
		TargetFunc: "Unreachable",
		MustHit:    false,
		Expecting:  true,
		AssertType: "none",
		Condition:  true,
	}

	hintMap["Reachable"] = &AssertionFuncInfo{
		TargetFunc: "Reachable",
		MustHit:    true,
		Expecting:  true,
		AssertType: "none",
		Condition:  true,
	}
	return hintMap
}

func (m AssertionHints) HintsForName(name string) *AssertionFuncInfo {
	if v, ok := m[name]; ok {
		return v
	}
	return nil
}

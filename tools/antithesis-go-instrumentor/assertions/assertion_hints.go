package assertions

type AssertionFuncInfo struct {
	TargetFunc string
	MustHit    bool
	Expecting  bool
	AssertType string
	Condition  bool
	MessageArg int
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
		MessageArg: 1,
	}

	hintMap["AlwaysOrUnreachable"] = &AssertionFuncInfo{
		TargetFunc: "AlwaysOrUnreachable",
		MustHit:    false,
		Expecting:  true,
		AssertType: "every",
		Condition:  false,
		MessageArg: 1,
	}

	hintMap["Sometimes"] = &AssertionFuncInfo{
		TargetFunc: "Sometimes",
		MustHit:    true,
		Expecting:  true,
		AssertType: "some",
		Condition:  false,
		MessageArg: 1,
	}

	hintMap["Unreachable"] = &AssertionFuncInfo{
		TargetFunc: "Unreachable",
		MustHit:    false,
		Expecting:  true,
		AssertType: "none",
		Condition:  true,
		MessageArg: 0,
	}

	hintMap["Reachable"] = &AssertionFuncInfo{
		TargetFunc: "Reachable",
		MustHit:    true,
		Expecting:  true,
		AssertType: "none",
		Condition:  true,
		MessageArg: 0,
	}
	return hintMap
}

func (m AssertionHints) HintsForName(name string) *AssertionFuncInfo {
	if v, ok := m[name]; ok {
		return v
	}
	return nil
}
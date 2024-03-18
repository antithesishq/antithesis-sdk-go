package assertions

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

func (m AssertionHints) HintsForName(name string) *AssertionFuncInfo {
	if v, ok := m[name]; ok {
		return v
	}
	return nil
}

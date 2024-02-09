package main

type AssertionFuncInfo struct {
	TargetFunc string
	MustHit    bool
	Expecting  bool
	AssertType string
	Condition  bool
}

type AssertionHints map[string]*AssertionFuncInfo

func setup_hint_map() AssertionHints {
	hint_map := make(AssertionHints)

	hint_map["Always"] = &AssertionFuncInfo{
		TargetFunc: "Always",
		MustHit:    true,
		Expecting:  true,
		AssertType: "every",
		Condition:  false,
	}

	hint_map["AlwaysOrUnreachable"] = &AssertionFuncInfo{
		TargetFunc: "AlwaysOrUnreachable",
		MustHit:    false,
		Expecting:  true,
		AssertType: "every",
		Condition:  false,
	}

	hint_map["Sometimes"] = &AssertionFuncInfo{
		TargetFunc: "Sometimes",
		MustHit:    true,
		Expecting:  true,
		AssertType: "some",
		Condition:  false,
	}

	hint_map["Unreachable"] = &AssertionFuncInfo{
		TargetFunc: "Unreachable",
		MustHit:    false,
		Expecting:  true,
		AssertType: "none",
		Condition:  true,
	}

	hint_map["Reachable"] = &AssertionFuncInfo{
		TargetFunc: "Reachable",
		MustHit:    true,
		Expecting:  true,
		AssertType: "none",
		Condition:  true,
	}
	return hint_map
}

func (m AssertionHints) hints_for_name(name string) *AssertionFuncInfo {
	if v, ok := m[name]; ok {
		return v
	}
	return nil
}

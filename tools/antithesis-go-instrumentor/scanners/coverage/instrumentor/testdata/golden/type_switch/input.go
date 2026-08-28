package sample

func Kind(x any) string {
	switch x.(type) {
	case int:
		return "int"
	case string:
		return "string"
	}
	return "other"
}

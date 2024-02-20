package common

func Pluralize(val int, singularText string) string {
	if val == 1 {
		return singularText
	}
	return singularText + "s"
}

package rules

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

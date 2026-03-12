package util

func RequireNotEmptyString(s string) {
	if s == "" {
		panic("Missing value")
	}
}

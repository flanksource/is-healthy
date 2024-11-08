package health

func GetAWSResourceHealth(_, status string) (health HealthStatus) {
	return GetHealthFromStatusName(status)
}

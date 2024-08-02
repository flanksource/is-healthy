package health

func GetMongoHealth(obj map[string]any) (health HealthStatus) {
	hr := HealthStatus{
		Status: HealthStatusUnknown,
		Health: HealthUnknown,
		Ready:  false,
	}

	if v, ok := obj["clusterType"]; ok && v.(string) == "REPLICASET" {
		if v, ok := obj["stateName"]; ok {
			hr.Status = HealthStatusCode(v.(string))
			hr.Ready = true
		}
	}

	return hr
}

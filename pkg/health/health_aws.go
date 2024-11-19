package health

import (
	"fmt"
	"strings"
)

func GetAWSResourceHealth(_, status string) (health HealthStatus) {
	return GetHealthFromStatusName(status)
}

func getAWSHealthByConfigType(configType string, obj map[string]any, states ...string) HealthStatus {
	switch configType {
	case "AWS::ECS::Task":
		return GetECSTaskHealth(obj)
	case "AWS::Cloudformation::Stack":
		return GetHealthFromStatusName(get(obj, "StackStatus"), get(obj, "StackStatusReason"))
	case "AWS::EC2::Instance":
		return GetHealthFromStatusName(get(obj, "State"))
	case "AWS::RDS::DBInstance":
		return GetHealthFromStatusName(get(obj, "DBInstanceStatus"))
	case "AWS::ElasticLoadBalancing::LoadBalancer":
		return GetHealthFromStatusName(get(obj, "State", "Code"))
	case "AWS::AutoScaling::AutoScalingGroup":
		return GetHealthFromStatusName(get(obj, "Status"))
	case "AWS::Lambda::Function":
		return GetHealthFromStatusName(get(obj, "State"), get(obj, "StateReasonCode"))
	case "AWS::DynamoDB::Table":
		return GetHealthFromStatusName(get(obj, "TableStatus"))
	case "AWS::ElastiCache::CacheCluster":
		return GetHealthFromStatusName(get(obj, "CacheClusterStatus"))
	}

	if len(states) > 0 {
		return GetHealthFromStatusName(states[0])
	} else {
		for k, v := range obj {
			_k := strings.ToLower(k)
			_v := fmt.Sprintf("%s", v)
			if _k == "status" || _k == "state" ||
				strings.HasSuffix(_k, "status") {
				return GetHealthFromStatusName(_v)
			}
		}
	}
	return HealthStatus{
		Health: HealthUnknown,
	}

}

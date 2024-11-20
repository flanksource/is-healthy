package health

import (
	"fmt"
	"strings"
)

func GetAWSResourceHealth(_, status string) (health HealthStatus) {
	return GetHealthFromStatusName(status)
}

func getAWSHealthByConfigType(configType string, obj map[string]any, states ...string) HealthStatus {
	switch strings.ToLower(configType) {
	case "aws::ecs::task":
		return GetECSTaskHealth(obj)
	case "aws::cloudformation::stack":
		return GetHealthFromStatusName(get(obj, "StackStatus"), get(obj, "StackStatusReason"))
	case "aws::ec2::instance":
		return GetHealthFromStatusName(get(obj, "State"))
	case "aws::rds::dbinstance":
		return GetHealthFromStatusName(get(obj, "DBInstanceStatus"))
	case "aws::elasticloadbalancing::loadbalancer":
		return GetHealthFromStatusName(get(obj, "State", "Code"))
	case "aws::autoscaling::autoscalinggroup":
		return GetHealthFromStatusName(get(obj, "Status"))
	case "aws::lambda::function":
		return GetHealthFromStatusName(get(obj, "State"), get(obj, "StateReasonCode"))
	case "aws::dynamodb::table":
		return GetHealthFromStatusName(get(obj, "TableStatus"))
	case "aws::elasticache::cachecluster":
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

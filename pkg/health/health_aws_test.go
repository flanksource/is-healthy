package health

import "testing"

func TestMapAWSStatus(t *testing.T) {
	type args struct {
		status       string
		resourceType string
	}
	tests := []struct {
		name string
		args args
		want HealthStatusCode
	}{
		{
			name: "ec2",
			args: args{status: "shutting-down", resourceType: AWSResourceTypeEC2},
			want: HealthStatusDeleting,
		},
		{
			name: "unknown resource",
			args: args{status: "shutting-down", resourceType: "blob"},
			want: HealthStatusUnknown,
		},
		{
			name: "unknown type",
			args: args{status: "wakingup", resourceType: AWSResourceTypeEC2},
			want: HealthStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAWSResourceHealth(tt.args.resourceType, tt.args.status); got.Status != tt.want {
				t.Errorf("MapAWSStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

package telemetry

import (
	"reflect"
	"testing"
)

func Test_newCommonAttributes(t *testing.T) {
	type args struct {
		attributes map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *commonAttributes
		wantErr bool
	}{
		{
			name: "Nil Attributes",
			args: args{
				attributes: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Length 0 Attributes",
			args: args{
				attributes: map[string]interface{}{},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newCommonAttributes(tt.args.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("newCommonAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newCommonAttributes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

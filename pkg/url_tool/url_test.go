package url_tool

import "testing"

func TestBasePath(t *testing.T) {
	type args struct {
		orginUrl string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "test", args: args{"www.baidu.com/gsf"}, want: "gsf", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BasePath(tt.args.orginUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("BasePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BasePath() got = %v, want %v", got, tt.want)
			}
		})
	}
}

package build

import (
	"reflect"
	"testing"
)

// TestCap 复刻 package.sh 的 cap()：仅首字母大写、其余原样。
func TestCap(t *testing.T) {
	cases := map[string]string{
		"ap01018":      "Ap01018",
		"bpom3410":     "Bpom3410",
		"gpgzmkk042":   "Gpgzmkk042",
		"johnc1010991": "Johnc1010991",
		"":             "",
		"A":            "A",
		"a":            "A",
		// 已大写首字母保持不变；内部字符不被改动。
		"Ab":  "Ab",
		"aBc": "ABc",
	}
	for in, want := range cases {
		if got := Cap(in); got != want {
			t.Errorf("Cap(%q)=%q want %q", in, got, want)
		}
	}
}

// TestTaskName 验证 assemble<Cap>Release。
func TestTaskName(t *testing.T) {
	if got := TaskName("ap01018"); got != "assembleAp01018Release" {
		t.Errorf("TaskName=%q", got)
	}
}

// TestArgs 验证组装的 gradlew 参数与 package.sh 行为一致：
// 多个 task + -PtestEvents=<bool>（+ 透传参数）。
func TestArgs(t *testing.T) {
	o := Options{Flavors: []string{"ap01018", "ap01034"}, TestEvents: true, ExtraArgs: []string{"--offline"}}
	got := o.Args()
	want := []string{"assembleAp01018Release", "assembleAp01034Release", "-PtestEvents=true", "--offline"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Args=%v want %v", got, want)
	}

	o2 := Options{Flavors: []string{"gpgzmkk042"}}
	if got := o2.Args(); !reflect.DeepEqual(got, []string{"assembleGpgzmkk042Release", "-PtestEvents=false"}) {
		t.Errorf("Args(default)=%v", got)
	}
}

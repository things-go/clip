package signature

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	jsonStr := `{
  "name": "jjl",
  "data": {
    "m": {
      "t": 123,
      "arr": [
        678,
        123
      ]
    },
    "contact": [
      {
        "phone": [
          "13705970110",
          "13705970189"
        ],
        "name": "zhangsan"
      },
      {
        "name": "lisi",
        "phone": [
          "13705970181",
          "13705970182"
        ]
      }
    ]
  }
}`
	mp := make(map[string]any)
	err := json.Unmarshal([]byte(jsonStr), &mp)
	if err != nil {
		t.Fatal(err)
	}
	got := Sign(mp, "a74db8b7-3b97-4653-8e80-ae90ba0e81b3", HexSha256)
	want := "01a85d60c6a67d7ab58408e99109779f075825275f470104018a939b1a196917"
	if got != want {
		t.Errorf("SignHexSha256() = %v, want %v", got, want)
	}
}

var testMp = map[string]any{
	"name":   "jjl",
	"phone":  "13705970181",
	"phone1": "13705970181",
}

func BenchmarkSign(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sign(testMp, "123456", HexSha256)
	}
}

func TestIat(t *testing.T) {
	iat := Iat()
	require.True(t, VerifyIat(iat, time.Second))

	time.Sleep(time.Second)
	require.False(t, VerifyIat(iat, time.Millisecond*500))
}

func TestIatSign(t *testing.T) {
	str := "1888888888"
	iat, sign := IatSign(str)
	require.True(t, VerifyIatSign(iat, sign, str, time.Second))
	require.False(t, VerifyIatSign(iat, "sign", str, time.Millisecond*500))

	time.Sleep(time.Second)
	require.False(t, VerifyIatSign(iat, sign, str, time.Millisecond*500))
}

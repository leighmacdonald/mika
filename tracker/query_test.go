package tracker

import "testing"

//noinspection GoUnusedGlobalVariable
var result *query

func benchmarkQuery(b *testing.B) {
	q := "http://localhost:34000/announce?info_hash=%ac%c3%b2%e43%d7%c7GZ%bbYA%b5h%1c%b7%a1%ea%26%e2" +
		"&peer_id=ABCDEFGHIJKLMNOPQRST&ip=12.34.56.78&port=6881&downloaded=0&left=970"
	for i := 0; i < b.N; i++ {
		result, _ = queryStringParser(q)
	}
}

func BenchmarkQuery1000(b *testing.B) {
	benchmarkQuery(b)
}

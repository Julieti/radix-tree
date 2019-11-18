package radix

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

//func TestLongestPrefix(t *testing.T) {
//	r := New()
//
//	keys := []string{
//		"",
//		"foo",
//		"foobar",
//		"foobarbaz",
//		"foobarbazzip",
//		"foozip",
//	}
//	//for _, k := range keys {
//	//	r.Insert(k, nil)
//	//}
//	if r.Len() != len(keys) {
//		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
//	}
//
//	type exp struct {
//		inp string
//		out string
//	}
//	cases := []exp{
//		{"a", ""},
//		{"abc", ""},
//		{"fo", ""},
//		{"foo", "foo"},
//		{"foob", "foo"},
//		{"foobar", "foobar"},
//		{"foobarba", "foobar"},
//		{"foobarbaz", "foobarbaz"},
//		{"foobarbazzi", "foobarbaz"},
//		{"foobarbazzip", "foobarbazzip"},
//		{"foozi", "foo"},
//		{"foozip", "foozip"},
//		{"foozipzap", "foozip"},
//	}
//	for _, test := range cases {
//		m, _, ok := r.LongestPrefix(test.inp)
//		if !ok {
//			t.Fatalf("no match: %v", test)
//		}
//		if m != test.out {
//			t.Fatalf("mis-match: %v %v", m, test)
//		}
//	}
//}
//
//func TestWalkPath(t *testing.T) {
//	r := New()
//
//	keys := []string{
//		"foo",
//		"foo/bar",
//		"foo/bar/baz",
//		"foo/baz/bar",
//		"foo/zip/zap",
//		"zipzap",
//	}
//	//for _, k := range keys {
//	//	r.Insert(k, nil)
//	//}
//	if r.Len() != len(keys) {
//		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
//	}
//
//	type exp struct {
//		inp string
//		out []string
//	}
//	cases := []exp{
//		{
//			"f",
//			[]string{},
//		},
//		{
//			"foo",
//			[]string{"foo"},
//		},
//		{
//			"foo/",
//			[]string{"foo"},
//		},
//		{
//			"foo/ba",
//			[]string{"foo"},
//		},
//		{
//			"foo/bar",
//			[]string{"foo", "foo/bar"},
//		},
//		{
//			"foo/bar/baz",
//			[]string{"foo", "foo/bar", "foo/bar/baz"},
//		},
//		{
//			"foo/bar/bazoo",
//			[]string{"foo", "foo/bar", "foo/bar/baz"},
//		},
//		{
//			"z",
//			[]string{},
//		},
//	}
//
//	for _, test := range cases {
//		out := []string{}
//		fn := func(s string, v interface{}) bool {
//			out = append(out, s)
//			return false
//		}
//		r.WalkPath(test.inp, fn)
//		sort.Strings(out)
//		sort.Strings(test.out)
//		if !reflect.DeepEqual(out, test.out) {
//			t.Fatalf("mis-match: %v %v", out, test.out)
//		}
//	}
//}

func TestCreate(t *testing.T)  {
	words := loadTestFile("/Users/leah/Downloads/words.txt")

	nowTime := time.Now()
	r := NewFromMap(words)
	fmt.Println(time.Now().Sub(nowTime))

	//r.Walk(func(k string, v interface{}) bool {
	//	println(k)
	//	return false
	//})
	//r.WalkFirst()

	//r.WalkSecond()

	r.WalkThird()
}

func loadTestFile(path string) map[string]string {
	words := make(map[string]string, 0)
	file, err := os.Open(path)
	if err != nil {
		panic("Couldn't open " + path)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		if line, err := reader.ReadBytes(byte('\n')); err != nil {
			break
		} else {
			if len(line) > 0 {
				parts := strings.Split(string(line)," ")
				words[parts[0]] = parts[1]
			}
		}
	}
	return words
}

func TestCore(t *testing.T)  {
	num := runtime.NumCPU()
	fmt.Println(num)
	inner := []int {0,-1,0,-1}
	i := sort.Search(len(inner), func(i int) bool {
		return  inner[i] == -1
	})

	fmt.Println(i)
}

func TestKeyWordFind(t *testing.T)  {
	//cid := "QmZU23bWvhgME4PaookgpEVHX22NxtLer2WgQptuuFQ48g"
	//keyWord := "will"
	//cid := "QmRsCEFptyjAftBWJYEiZdy6Prv1cQKnzCwYw3pooMGu97"// 300词成功的cid

	cid := "QmTwYLezBRVoztEuWntZ2BWE14hjcH9KU53rQgrTpeGHD4"
	keyWord := "well"

	_, result, _ := Get(cid, keyWord)
	//fmt.Println(logo)
	fmt.Println(result)
	//fmt.Println(pathList)
}

func TestUpdate(t *testing.T)  {
	//newWords := loadTestFile("/Users/leah/Downloads/words.txt")
	cid := "QmWcwgUhiscW8JmdbTGTAgeVRkxUopTpiDXzUbCvfU1cxt"
	//newWords := loadTestFile("/Users/leah/Downloads/words.txt")
	//cid := "QmedgDTx1buF58LuPLkQjRkEfW441YhjZkLZEubuFTmMQ4"
	newWords := map[string]string {
		"well":"QmYKwLHLqbMwK5qgK6srL79qRxjd8GiQp3gdYAkL9yZV7Y,0.0031847135",
	}
	Update(newWords, cid)
}
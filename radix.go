package radix

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"zly.ecnu.edu.cn/go-radix/ipfs"
)

// WalkFn is used when walking the tree. Takes a
// key and value, returning if iteration should
// be terminated.
type WalkFn func(s string, v interface{}) bool

// leafNode is used to represent a value
type leafNode struct {
	key string
	val string
}

// edge is used to represent an edge node
type edge struct {
	label byte
	node  *node
}

type node struct {
	// leaf is used to store possible leaf
	leaf *leafNode

	// prefix is the common prefix we ignore
	prefix string

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	edges edges
}


type uploadNode struct {
	Type int `json:"type"`
	Key string `json:"key"`
	Value string `json:"value"`
	Prefix string `json:"prefix"`
	Label []string `json:"label"`
	Edges []string `json:"edges"`
	position []int
	Pos []int `json:"pos"`
}

type leaf struct {
	Type int `json:"type"`
	Key []string `json:"key"`
	Value []string `json:"value"`
	Prefix []string `json:"prefix"`
}


func (n *node) isLeaf() bool {
	return n.leaf != nil
}

func (n *node) addEdge(e edge) {
	n.edges = append(n.edges, e)
	n.edges.Sort()
}

func (n *node) updateEdge(label byte, node *node) {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		n.edges[idx].node = node
		return
	}
	panic("replacing missing edge")
}

func (n *node) getEdge(label byte) *node {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		return n.edges[idx].node
	}
	return nil
}


type edges []edge

func (e edges) Len() int {
	return len(e)
}

func (e edges) Less(i, j int) bool {
	return e[i].label < e[j].label
}

func (e edges) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e edges) Sort() {
	sort.Sort(e)
}

// Tree implements a radix tree. This can be treated as a
// Dictionary abstract data type. The main advantage over
// a standard hash map is prefix-based lookups and
// ordered iteration,
type Tree struct {
	root *node
	size int
}

//// New returns an empty Tree
//func New() *Tree {
//	return NewFromMap(nil)
//}

// NewFromMap returns a new tree containing the keys
// from an existing map
func NewFromMap(m map[string]string) *Tree {
	t := &Tree{root: &node{}}
	for k, v := range m {
		t.Insert(k, v)
	}
	return t
}

// Len is used to return the number of elements in the tree
func (t *Tree) Len() int {
	return t.size
}

// longestPrefix finds the length of the shared prefix
// of two strings
func longestPrefix(k1, k2 string) int {
	max := len(k1)
	if l := len(k2); l < max {
		max = l
	}
	var i int
	for i = 0; i < max; i++ {
		if k1[i] != k2[i] {
			break
		}
	}
	return i
}

// Insert is used to add a newentry or update
// an existing entry. Returns if updated.
func (t *Tree) Insert(s string, v string) (interface{}, bool) {
	var parent *node
	n := t.root
	search := s
	for {
		// Handle key exhaution
		if len(search) == 0 {
			if n.isLeaf() {
				old := n.leaf.val
				n.leaf.val = v
				return old, true
			}

			n.leaf = &leafNode{
				key: s,
				val: v,
			}
			t.size++
			return nil, false
		}

		// Look for the edge
		parent = n
		n = n.getEdge(search[0])

		// No edge, create one
		if n == nil {
			e := edge{
				label: search[0],
				node: &node{
					leaf: &leafNode{
						key: s,
						val: v,
					},
					prefix: search,
				},
			}
			parent.addEdge(e)
			t.size++
			return nil, false
		}

		// Determine longest prefix of the search key on match
		commonPrefix := longestPrefix(search, n.prefix)
		if commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
			continue
		}

		// Split the node
		t.size++
		child := &node{
			prefix: search[:commonPrefix],
		}
		parent.updateEdge(search[0], child)

		// Restore the existing node
		child.addEdge(edge{
			label: n.prefix[commonPrefix],
			node:  n,
		})
		n.prefix = n.prefix[commonPrefix:]

		// Create a new leaf node
		leaf := &leafNode{
			key: s,
			val: v,
		}

		// If the new key is a subset, add to to this node
		search = search[commonPrefix:]
		if len(search) == 0 {
			child.leaf = leaf
			return nil, false
		}

		// Create a new edge for the node
		child.addEdge(edge{
			label: search[0],
			node: &node{
				leaf:   leaf,
				prefix: search,
			},
		})
		return nil, false
	}
}

// LongestPrefix is like Get, but instead of an
// exact match, it will return the longest prefix match.
func (t *Tree) LongestPrefix(s string) (string, interface{}, bool) {
	var last *leafNode
	n := t.root
	search := s
	for {
		// Look for a leaf node
		if n.isLeaf() {
			last = n.leaf
		}

		// Check for key exhaution
		if len(search) == 0 {
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if strings.HasPrefix(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
	if last != nil {
		return last.key, last.val, true
	}
	return "", nil, false
}


// Walk is used to walk the tree
func (t *Tree) Walk(fn WalkFn) {
	recursiveWalk(t.root, fn)
}

// WalkPrefix is used to walk the tree under a prefix
func (t *Tree) WalkPrefix(prefix string, fn WalkFn) {
	n := t.root
	search := prefix
	for {
		// Check for key exhaution
		if len(search) == 0 {
			recursiveWalk(n, fn)
			return
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if strings.HasPrefix(search, n.prefix) {
			search = search[len(n.prefix):]

		} else if strings.HasPrefix(n.prefix, search) {
			// Child may be under our search prefix
			recursiveWalk(n, fn)
			return
		} else {
			break
		}
	}

}

// WalkPath is used to walk the tree, but only visiting nodes
// from the root down to a given leaf. Where WalkPrefix walks
// all the entries *under* the given prefix, this walks the
// entries *above* the given prefix.
func (t *Tree) WalkPath(path string, fn WalkFn) {
	n := t.root
	search := path
	for {
		// Visit the leaf values if any
		if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
			return
		}

		// Check for key exhaution
		if len(search) == 0 {
			return
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			return
		}

		// Consume the search prefix
		if strings.HasPrefix(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
}

// recursiveWalk is used to do a pre-order walk of a node
// recursively. Returns true if the walk should be aborted
func recursiveWalk(n *node, fn WalkFn) bool {
	// Visit the leaf values if any
	if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
		return true
	}

	// Recurse on the children
	for _, e := range n.edges {
		if recursiveWalk(e.node, fn) {
			return true
		}
	}
	return false
}

// ToMap is used to walk the tree and convert it into a map
func (t *Tree) ToMap() map[string]interface{} {
	out := make(map[string]interface{}, t.size)
	t.Walk(func(k string, v interface{}) bool {
		out[k] = v
		return false
	})
	return out
}

/**
	单线程
 */
func (t *Tree) WalkFirst() {
	depth := 0
	fmt.Println(recursiveWalkFirst(t.root, depth))
}

func recursiveWalkFirst(n *node, depth int) string {
	result := make([]string, 0)
	labels := make([]string, 0)
	var value string
	key := ""
	hash := ""
	// Recurse on the children
	for _, e := range n.edges {
		hash = recursiveWalkFirst(e.node, depth+1)
		result = append(result,hash)
	}

	// Visit the leaf values if any
	if len(n.edges) == 0 && n.leaf != nil {

		newLeaf := &uploadNode{
			Type: 0,
			Key: n.leaf.key,
			Value: n.leaf.val,
			Prefix: n.prefix,
		}
		l, _ := json.Marshal(newLeaf)

		return ipfs.UploadIndex(string(l))
	}



	for _, e := range n.edges {
		labels = append(labels, string(e.label))
	}

	if n.leaf != nil {
		key = n.leaf.key
		value = n.leaf.val
	}

	newNode := &uploadNode{
		Type: 1,
		Key: key,
		Value: value,
		Prefix: n.prefix,
		Label: labels,
		Edges: result,
	}

	in, _ := json.Marshal(newNode)

	return ipfs.UploadIndex(string(in))
}


/**
	多线程
 */

func (t *Tree) WalkSecond() {
	depth := 0

	ch := make(chan string)
	result := make([]string, 0)
	label := make([]string, 0)

	if len(t.root.edges) > 0 {
		for  _, e := range t.root.edges {
			go recursiveWalkSecond(e.node, depth, ch, e.label)
		}

	}

	for{
		 hash := <-ch
		 label = append(label, string(hash[0]))
		 result = append(result, hash[1:])

		 if len(result) == len(t.root.edges) {
		 	 close(ch)
			 newRoot := &uploadNode{
				 Label: label,
				 Edges: result,
			 }

			 r, _ := json.Marshal(newRoot)

			 fmt.Println(ipfs.UploadIndex(string(r)))
			 return
		 }
	}

}

func recursiveWalkSecond(n *node, depth int, ch chan string, label byte) string {
	result := make([]string, 0)
	labels := make([]string, 0)
	var value string
	key := ""
	hash := ""
	// Recurse on the children
	for _, e := range n.edges {
		hash = recursiveWalkSecond(e.node, depth+1, ch, label)
		result = append(result,hash)
	}


	// Visit the leaf values if any
	if len(n.edges) == 0 && n.leaf != nil {

		newLeaf := &uploadNode{
			Type: 0,
			Key: n.leaf.key,
			Value: n.leaf.val,
			Prefix: n.prefix,
		}
		l, _ := json.Marshal(newLeaf)

		if depth == 0 {
			ch <- ipfs.UploadIndex(string(l))
		}

		return ipfs.UploadIndex(string(l))
	}

	for _, e := range n.edges {
		labels = append(labels, string(e.label))
	}

	if n.leaf != nil {
		key = n.leaf.key
		value = n.leaf.val
	}

	newNode := &uploadNode{
		Type: 1,
		Key: key,
		Value: value,
		Prefix: n.prefix,
		Label: labels,
		Edges: result,
	}

	in, _ := json.Marshal(newNode)

	if depth == 0 {
		ch <- string(label) + ipfs.UploadIndex(string(in))
	}

	return ipfs.UploadIndex(string(in))
}

/**
	合并叶子节点  单线程
 */
func (t *Tree) WalkThird() {
	depth := 0
	fmt.Println(recursiveWalkThird(t.root, depth))
}

func recursiveWalkThird(n *node, depth int) string {
	result := make([]string, 0)
	labels := make([]string, 0)
	position := make([]int, 0)
	pos := make([]int, 0)
	length := 0
	start := 0
	var value string
	key := ""
	hash := ""
	nLeaf := &leaf{}
	// Recurse on the children
	for index, e := range n.edges {
		if e.node.leaf != nil && len(e.node.edges) == 0 {
			nLeaf.Type = 0
			nLeaf.Key = append(nLeaf.Key, e.node.leaf.key)
			nLeaf.Value = append(nLeaf.Value, e.node.leaf.val)
			nLeaf.Prefix = append(nLeaf.Prefix,e.node.prefix)
			position = append(position, index)
			result = append(result,"")
			pos = append(pos, -1)
			continue
		}
		pos = append(pos, 0)
		hash = recursiveWalkThird(e.node, depth+1)
		result = append(result,hash)
	}

	for i := 0; i < len(nLeaf.Key); i++ {
		nowLength := len(nLeaf.Key[i]) + len(nLeaf.Value[i]) + len(nLeaf.Prefix[i])
		if length + nowLength > 4096 {
			tempLeaf := &leaf {
				Type: 0,
				Key: nLeaf.Key[:i],
				Value: nLeaf.Value[:i],
				Prefix: nLeaf.Prefix[:i],
			}

			in, _ := json.Marshal(tempLeaf)
			cid := ipfs.UploadIndex(string(in))

			for j := start; j < i; j++ {
				result[position[j]] = cid
			}
			nLeaf.Key = nLeaf.Key[i:]
			nLeaf.Value = nLeaf.Value[i:]
			nLeaf.Prefix = nLeaf.Prefix[i:]
			position = position[i:]
			start = i
			length = 0
		}

		length += nowLength
	}

	in, _ := json.Marshal(nLeaf)
	cid := ipfs.UploadIndex(string(in))

	for j := 0; j < len(nLeaf.Key); j++ {
		result[position[j]] = cid
	}

	for _, e := range n.edges {
		labels = append(labels, string(e.label))
	}

	if n.leaf != nil {
		key = n.leaf.key
		value = n.leaf.val
	}

	newNode := &uploadNode{
		Type: 1,
		Key: key,
		Value: value,
		Prefix: n.prefix,
		Label: labels,
		Edges: result,
		Pos: pos,
	}

	in, _ = json.Marshal(newNode)

	return ipfs.UploadIndex(string(in))
}

/**
	TODO 合并叶子节点  多线程
 */
func (t *Tree) WalkFourth() {
	depth := 0
	fmt.Println(recursiveWalkFourth(t.root, depth))
}

func recursiveWalkFourth(n *node, depth int) string {
	result := make([]string, 0)
	labels := make([]string, 0)
	position := make([]int, 0)
	length := 0
	start := 0
	var value string
	key := ""
	hash := ""
	nLeaf := &leaf{}
	// Recurse on the children
	for index, e := range n.edges {
		if e.node.leaf != nil && len(e.node.edges) == 0 {
			nLeaf.Key = append(nLeaf.Key, e.node.leaf.key)
			nLeaf.Value = append(nLeaf.Value, e.node.leaf.val)
			nLeaf.Prefix = append(nLeaf.Prefix,e.node.prefix)
			position = append(position, index)
			result = append(result,"")
			continue
		}
		hash = recursiveWalkFourth(e.node, depth+1)
		result = append(result,hash)
	}

	for i := 0; i < len(nLeaf.Key); i++ {
		nowLength := len(nLeaf.Key[i]) + len(nLeaf.Value[i])+ len(nLeaf.Prefix[i])
		if length + nowLength > 4096 {
			tempLeaf := &leaf {
				Key: nLeaf.Key[:i],
				Value: nLeaf.Value[:i],
				Prefix: nLeaf.Prefix[:i],
			}

			in, _ := json.Marshal(tempLeaf)
			cid := ipfs.UploadIndex(string(in))

			for j := start; j < i; j++ {
				result[position[j]] = cid
			}
			nLeaf.Key = nLeaf.Key[i:]
			nLeaf.Value = nLeaf.Value[i:]
			nLeaf.Prefix = nLeaf.Prefix[i:]
			position = position[i:]
			start = i
			length = 0
		}

		length += nowLength
	}

	in, _ := json.Marshal(nLeaf)
	cid := ipfs.UploadIndex(string(in))

	for j := 0; j < len(nLeaf.Key); j++ {
		result[position[j]] = cid
	}

	for _, e := range n.edges {
		labels = append(labels, string(e.label))
	}

	if n.leaf != nil {
		key = n.leaf.key
		value = n.leaf.val
	}

	newNode := &uploadNode{
		Type: 1,
		Key: key,
		Value: value,
		Prefix: n.prefix,
		Label: labels,
		Edges: result,
	}

	in, _ = json.Marshal(newNode)

	return ipfs.UploadIndex(string(in))
}

func Get(cid string, keyWord string) (string, string, []string) {
	pathList := make([]string, 0)
	pathList = append(pathList, cid)
	// 判断根节点
	content := ipfs.CatIndex(cid)
	n := &uploadNode{}
	err := json.Unmarshal([]byte(content), &n)
	search := keyWord

	i := sort.Search(len(n.Label), func(i int) bool {
		return string(int(search[0])) <= n.Label[i]
	})

	if string(int(search[0])) > n.Label[len(n.Label) - 1] {
		i = len(n.Label) - 1
	}

	if i == -1 || n.Label[i] != string(int(search[0])) {
		return string(int(search[0])), "no label", pathList
	}
	content = ipfs.CatIndex(n.Edges[i])
	pathList = append(pathList, n.Edges[i])

	for n.Pos[i] == 0 {
		err = json.Unmarshal([]byte(content), &n)
		if err != nil {
			log.Printf("Find index error: unmarshal error: %v", err)
		}

		if strings.HasPrefix(search, n.Prefix) {
			search = search[len(n.Prefix):]
		} else {
			return search, "inner split", pathList
		}

		if search == "" {
			if n.Key == keyWord {
				return "", n.Value, pathList
			} else {
				return "", "no inner key", pathList
			}
		}

		i = sort.Search(len(n.Label), func(i int) bool {
			return string(int(search[0])) <= n.Label[i]
		})

		if string(int(search[0])) > n.Label[len(n.Label) - 1] {
			i = len(n.Label) - 1
		}

		if i < len(n.Label) && i >= 0 && n.Label[i] == string(int(search[0]))  {
			content = ipfs.CatIndex(n.Edges[i])
			pathList = append(pathList, n.Edges[i])
		} else {
			return string(int(search[0])), "no label", pathList
		}
	}

	l := &leaf{}
	//pathList = append(pathList, n.Edges[i])
	err = json.Unmarshal([]byte(content), &l)
	i = sort.Search(len(l.Key), func(i int) bool {
		return keyWord <= l.Key[i]
	})

	if keyWord > l.Key[len(l.Key) - 1] {
		i = len(l.Key) - 1
	}

	if i < len(l.Key) && l.Key[i] == keyWord && i > 0 {
		return "", l.Value[i], pathList
	} else {
		return search, "no leaf key", pathList
	}


}

func Update(newWords map[string]string, cid string)  {
	for word := range newWords {
		cid = update(word, newWords, cid)
	}

	fmt.Println(cid)
}

func update(word string, newWords map[string]string, cid string) string {
	fmt.Println("word is", word)
	fmt.Println("cid is", cid)
	info, value, pathList := Get(cid, word)
	fmt.Println("value is", value)
	switch value {
	case "no label":
		lf := &leaf{}
		lf.Type = 0
		cur := -1
		inner := &uploadNode{}
		content := ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[: len(pathList) - 1]
		err := json.Unmarshal([]byte(content), &inner)
		if err != nil {
			log.Printf("no label index error: unmarshal error: %v", err)
			return ""
		}

		//添加Label
		i := sort.Search(len(inner.Label), func(i int) bool {
			return info <= inner.Label[i]
		})

		if info > inner.Label[len(inner.Label) - 1] {
			i = len(inner.Label) - 1
		}

		nLabel := append([]string{}, inner.Label[i:]...)
		nEdges := append([]string{},inner.Edges[i:]...)
		nPos := append([]int{}, inner.Pos[i:]...)
		inner.Label = append(inner.Label[0:i], info)
		inner.Edges = append(inner.Edges[0:i], "")
		inner.Pos = append(inner.Pos[0:i], -1)
		inner.Label = append(inner.Label, nLabel...)
		inner.Edges = append(inner.Edges, nEdges...)
		inner.Pos = append(inner.Pos, nPos...)

		// 寻找叶子节点
		for j := i + 1; j < len(inner.Label); j++ {
			if inner.Pos[j] == -1{
				cur = j

				content := ipfs.CatIndex(inner.Edges[cur])
				err = json.Unmarshal([]byte(content), &lf)
				break
			}
		}

		if cur != -1 {
			commonPrefix := longestPrefix(word, lf.Key[0])
			i = sort.Search(len(lf.Key), func(i int) bool {
				return word <= lf.Key[i]
			})

			if word > lf.Key[len(lf.Key) - 1] {
				i = len(lf.Key) - 1
			}
			nKey := append([]string{}, lf.Key[i:]...)
			nValue := append([]string{}, lf.Value[i:]...)
			nPrefix := append([]string{}, lf.Prefix[i:]...)

			lf.Key = append(lf.Key[0:i], word)
			lf.Value = append(lf.Value[0:i], newWords[word])
			lf.Prefix = append(lf.Prefix[0:i], word[commonPrefix:])

			lf.Key = append(lf.Key, nKey...)
			lf.Value = append(lf.Value, nValue...)
			lf.Prefix = append(lf.Prefix, nPrefix...)
			inner = judgeLeafSplit(lf, inner, word, )
		} else {
			if inner.Prefix != "" {
				pres := strings.Split(word, inner.Prefix)
				lf.Prefix = append(lf.Prefix, pres[len(pres) - 1])
			} else {
				lf.Prefix = append(lf.Prefix, word)
			}
			lf.Key = append(lf.Key, word)
			lf.Value = append(lf.Value, newWords[word])

			in, _ := json.Marshal(lf)
			cid := ipfs.UploadIndex(string(in))
			inner.Edges[i] = cid
		}
		cid = uploadInner(inner, pathList)
	case "no inner key":
		inner := &uploadNode{}
		content := ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[: len(pathList) - 1]
		err := json.Unmarshal([]byte(content), &inner)
		if err != nil {
			log.Printf("no inner key index error: unmarshal error: %v", err)
			return ""
		}
		inner.Key = word
		inner.Value = newWords[word]
		cid = uploadInner(inner, pathList)
	case "inner split":
		inner := &uploadNode{}
		content := ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[: len(pathList) - 1]
		err := json.Unmarshal([]byte(content), &inner)
		if err != nil {
			log.Printf("inner split index error: unmarshal error: %v", err)
			return ""
		}
		commonPrefix := longestPrefix(info, inner.Prefix)
		pre := info[commonPrefix:]
		//新创建内部结点
		nInner := &uploadNode{
			Type: 1,
			Prefix: info[:commonPrefix],
		}
		//修改原节点
		inner.Prefix = inner.Prefix[commonPrefix:]
		in, _ := json.Marshal(inner)
		cid1 := ipfs.UploadIndex(string(in))

		if len(pre) > 0 {
			// 新叶子节点
			lf := &leaf{
				Type: 0,
				Key: []string{word},
				Value: []string{newWords[word]},
				Prefix: []string{pre},
			}
			in, _ = json.Marshal(lf)
			cid2 := ipfs.UploadIndex(string(in))

			if inner.Prefix[0] > pre[0] {
				nInner.Label = append(nInner.Label, string(int(pre[0])), string(int(inner.Prefix[0])))
				nInner.Edges = append(nInner.Edges,cid2, cid1)
				nInner.Pos = append(nInner.Pos,-1, 0)
			} else {
				nInner.Label = append(nInner.Label, string(int(inner.Prefix[0])), string(int(pre[0])))
				nInner.Edges = append(nInner.Edges,cid1, cid2)
				nInner.Pos = append(nInner.Pos,0,-1)
			}
		} else {
			nInner.Key = word
			nInner.Value = newWords[word]
			nInner.Label = []string{string(int(inner.Prefix[0]))}
			nInner.Edges = []string{cid1}
			nInner.Pos = []int{0}
		}

		cid = uploadInner(nInner, pathList)
	case "no leaf key":
		var p uint8
		lf := &leaf{}
		nL := &leaf{}
		inner := &uploadNode{}
		lf.Type = 0
		nL.Type = 0
		inner.Type = 1
		//取出原leaf
		content := ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[:len(pathList) - 1]
		err := json.Unmarshal([]byte(content), &lf)
		if err != nil {
			log.Printf("no leaf key index error: unmarshal error: %v", err)
			return ""
		}
		i := sort.Search(len(lf.Prefix), func(i int) bool {
			return info[0] <= lf.Prefix[i][0]
		})

		if info[0] > lf.Prefix[len(lf.Prefix) - 1][0] {
			i = len(lf.Prefix) - 1
		}

		commonPrefix := longestPrefix(word, lf.Key[i])
		info = word[commonPrefix:]
		nPre := lf.Key[i][commonPrefix:]

		inner1 := &uploadNode{}
		content = ipfs.CatIndex(pathList[len(pathList) - 1])
		err = json.Unmarshal([]byte(content), &inner1)
		inner.Prefix =  word[len(inner1.Prefix): commonPrefix]

		if len(info) > 0 && len(nPre) > 0 {//产生两个叶子节点
			if info[0] < nPre[0] {
				nL.Key = append(nL.Key, word, lf.Key[i])
				nL.Value = append(nL.Value, newWords[word], lf.Value[i])
				nL.Prefix = append(nL.Prefix, info, nPre)
				inner.Label = append(inner.Label, string(int(info[0])), string(int(nPre[0])))

			} else {
				nL.Key = append(nL.Key, lf.Key[i], word)
				nL.Value = append(nL.Value, lf.Value[i], newWords[word])
				nL.Prefix = append(nL.Prefix, nPre, info)
				inner.Label = append(inner.Label, string(int(nPre[0])), string(int(info[0])))
			}
			in, _ := json.Marshal(nL)
			cid = ipfs.UploadIndex(string(in))
			inner.Edges = append(inner.Edges, cid, cid)
			inner.Pos = append(inner.Pos, -1, -1)
		} else if len(info) > 0 { // 新插入的word产生叶子节点
			nL.Key = append(nL.Key, word)
			nL.Value = append(nL.Value, newWords[word])
			nL.Prefix = append(nL.Prefix, info)

			in, _ := json.Marshal(nL)
			cid = ipfs.UploadIndex(string(in))

			//原叶子节点变为内部结点
			inner.Key = lf.Key[i]
			inner.Value = lf.Value[i]
			inner.Label = append(inner.Label, string(int(info[0])))
			inner.Edges = append(inner.Edges, cid)
			inner.Pos = append(inner.Pos, -1)

			p = inner.Prefix[0]

			//新内部结点cid
			in, _ = json.Marshal(inner)
			cid = ipfs.UploadIndex(string(in))

			if len(lf.Key) > 1 {
				// 原叶子节点重整
				lf.Key = append(lf.Key[0:i], lf.Key[i+1:]...)
				lf.Prefix = append(lf.Prefix[0:i], lf.Prefix[i+1:]...)
				lf.Value = append(lf.Value[0:i], lf.Value[i+1:]...)
				in, _ = json.Marshal(lf)
				cid1 := ipfs.UploadIndex(string(in))


				content = ipfs.CatIndex(pathList[len(pathList) - 1])
				pathList = pathList[: len(pathList) - 1]
				err = json.Unmarshal([]byte(content), &inner)

				i = sort.Search(len(inner.Label), func(i int) bool {
					return string(int(lf.Prefix[0][0])) <= inner.Label[i]
				})

				if string(int(lf.Prefix[0][0])) > inner.Label[len(inner.Label) - 1] {
					i = len(inner.Label) - 1
				}

				oldCid := inner.Edges[i]
				for j := 0; j < len(inner.Edges); j++ {
					if inner.Pos[j] == -1 && inner.Edges[j] == oldCid {
						if inner.Label[j] == string(int(p)) {
							inner.Edges[j] = cid
							inner.Pos[j] = 0
						} else {
							inner.Edges[j] = cid1
						}
					}
				}

			} else {
				content = ipfs.CatIndex(pathList[len(pathList) - 1])
				pathList = pathList[:len(pathList) - 1]
				err = json.Unmarshal([]byte(content), &inner)

				i = sort.Search(len(inner.Label), func(i int) bool {
					return string(int(p)) <= inner.Label[i]
				})

				if string(int(p)) > inner.Label[len(inner.Label) - 1] {
					i = len(inner.Label) - 1
				}

				inner.Edges[i] = cid
				inner.Pos[i] = 0
			}
		} else if len(nPre) > 0 { //新插入的word变为内部结点
			nL.Key = append(nL.Key, lf.Key[i])
			nL.Value = append(nL.Value, lf.Value[i])
			nL.Prefix = append(nL.Prefix, nPre)

			in, _ := json.Marshal(nL)
			cid = ipfs.UploadIndex(string(in))
			inner.Key = word
			inner.Value = newWords[word]
			inner.Label = append(inner.Label, string(int(nPre[0])))
			inner.Edges = append(inner.Edges, cid)
			inner.Pos = append(inner.Pos, -1)

			p = inner.Prefix[0]

			//新内部结点cid
			in, _ = json.Marshal(inner)
			cid = ipfs.UploadIndex(string(in))

			if len(lf.Key) > 1 {
				// 原叶子节点重整
				lf.Key = append(lf.Key[0:i], lf.Key[i+1:]...)
				lf.Prefix = append(lf.Prefix[0:i], lf.Prefix[i+1:]...)
				lf.Value = append(lf.Value[0:i], lf.Value[i+1:]...)
				in, _ = json.Marshal(lf)
				cid1 := ipfs.UploadIndex(string(in))


				content = ipfs.CatIndex(pathList[len(pathList) - 1])
				pathList = pathList[: len(pathList) - 1]
				err = json.Unmarshal([]byte(content), &inner)

				i = sort.Search(len(inner.Label), func(i int) bool {
					return string(int(lf.Prefix[0][0])) <= inner.Label[i]
				})

				if string(int(lf.Prefix[0][0])) > inner.Label[len(inner.Label) - 1] {
					i = len(inner.Label) - 1
				}

				oldCid := inner.Edges[i]
				for j := 0; j < len(inner.Edges); j++ {
					if inner.Pos[j] == -1 && inner.Edges[j] == oldCid {
						if inner.Label[j] == string(int(p)) {
							inner.Edges[j] = cid
							inner.Pos[j] = 0
						} else {
							inner.Edges[j] = cid1
						}
					}
				}

			} else {
				content = ipfs.CatIndex(pathList[len(pathList) - 1])
				pathList = pathList[:len(pathList) - 1]
				err = json.Unmarshal([]byte(content), &inner)

				i = sort.Search(len(inner.Label), func(i int) bool {
					return string(int(p)) <= inner.Label[i]
				})

				if string(int(p)) > inner.Label[len(inner.Label) - 1] {
					i = len(inner.Label) - 1
				}

				inner.Edges[i] = cid
				inner.Pos[i] = 0
			}
		}

		in, _ := json.Marshal(inner)
		cid = ipfs.UploadIndex(string(in))

		if inner.Prefix != "" {
			p = inner.Prefix[0]
		} else {
			return cid
		}

		content = ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[: len(pathList) - 1]
		err = json.Unmarshal([]byte(content), &inner)

		i = sort.Search(len(inner.Label), func(i int) bool {
			return  string(int(p)) <= inner.Label[i]
		})

		if string(int(p)) > inner.Label[len(inner.Label) - 1] {
			i = len(inner.Label) - 1
		}

		inner.Pos[i] = 0
		inner.Edges[i] = cid
		cid = uploadInner(inner, pathList)
	default: //word存在
		lf := &leaf{} //取出叶子节点
		inner := &uploadNode{}
		lf.Type = 0
		content := ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[:len(pathList) - 1]
		err := json.Unmarshal([]byte(content), &lf)

		if err != nil {
			err = json.Unmarshal([]byte(content), &inner)
			inner.Value = inner.Value + newWords[word]
		} else {
			content = ipfs.CatIndex(pathList[len(pathList) - 1])
			err = json.Unmarshal([]byte(content), &inner)
			pathList = pathList[:len(pathList) - 1]
			if err != nil {
				log.Printf("key exist index error: unmarshal error: %v", err)
				return ""
			}

			i := sort.Search(len(lf.Key), func(i int) bool {
				return word <= lf.Key[i]
			})

			if word > lf.Key[len(lf.Key) - 1] {
				i = len(lf.Key) - 1
			}
			lf.Value[i] = lf.Value[i] + newWords[word]

			inner = judgeLeafSplit(lf, inner, word)
		}
		cid = uploadInner(inner, pathList)
	}

	return cid
}

func judgeLeafSplit(lf *leaf, inner *uploadNode, word string) *uploadNode {
	length := 0
	for _, v := range lf.Value {
		length += len(v)
	}

	if length > 4096 { // 叶子节点超过4096bytes
		nL := &leaf{}
		nL.Type = 0
		l := len(lf.Key)/2
		nL.Key = lf.Key[l:]
		nL.Value = lf.Value[l:]
		nL.Prefix = lf.Prefix[l:]
		in, _ := json.Marshal(nL)
		cid2 := ipfs.UploadIndex(string(in))

		//修改原节点
		lf.Key = lf.Key[:l]
		lf.Value = lf.Value[:l]
		lf.Prefix = lf.Prefix[:l]
		in, _ = json.Marshal(lf)
		cid1 := ipfs.UploadIndex(string(in))


		p := nL.Prefix[0][0]
		i := sort.Search(len(inner.Label), func(i int) bool {
			return string(int(p)) <= inner.Label[i]
		})

		//修改叶子节点上层内部节点的 Edges
		for m := 0;m < len(inner.Edges); m++ {
			if inner.Pos[m] != 0 {
				if m < i {
					inner.Edges[m] = cid1
				} else {
					inner.Edges[m] = cid2
				}
			}
		}

	} else {
		in, _ := json.Marshal(lf)
		cid := ipfs.UploadIndex(string(in))
		commonPrefix := strings.LastIndex(lf.Key[0], lf.Prefix[0])

		i := sort.Search(len(inner.Label), func(i int) bool {
			return string(int(lf.Key[0][commonPrefix])) <= inner.Label[i]
		})

		if string(int(lf.Key[0][commonPrefix])) > inner.Label[len(inner.Label) - 1] {
			i = len(inner.Label) - 1
		}

		oldCid := inner.Edges[i]
		for j := i; j < len(inner.Label); j++ {
			if inner.Edges[j] ==  oldCid  || inner.Edges[j] == ""{
				inner.Edges[j] = cid
			}
		}

	}

	return inner
}

func uploadInner(inner *uploadNode, pathList []string) string {
	var p uint8
	in, _ := json.Marshal(inner)
	cid := ipfs.UploadIndex(string(in))
	if inner.Prefix != "" {
		p = inner.Prefix[0]
	}

	for len(pathList) > 0  {
		inner := &uploadNode{}
		content := ipfs.CatIndex(pathList[len(pathList) - 1])
		pathList = pathList[: len(pathList) - 1]
		err := json.Unmarshal([]byte(content), &inner)

		if err != nil {
			log.Printf("uploadInner error : unmarshal error: %v", err)
			return ""
		}
		i := sort.Search(len(inner.Label), func(i int) bool {
			return string(int(p)) <= inner.Label[i]
		})
		if string(int(p)) > inner.Label[len(inner.Label) - 1] {
			i = len(inner.Label) - 1
		}

		inner.Edges[i] = cid
		if inner.Prefix != "" {
			p = inner.Prefix[0]
		}

		in, _ := json.Marshal(inner)
		cid = ipfs.UploadIndex(string(in))
	}

	return cid
}
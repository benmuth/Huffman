package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

type tree struct {
	id int
	c  rune
	w  int
	l  *tree
	r  *tree
}

type treeInfo struct {
	id int
	c  rune
	l  int
	r  int
}

func combineTrees(t1 tree, t2 tree) tree {
	var t3 tree
	t3.w = t1.w + t2.w
	t3.l = &t1
	t3.r = &t2
	return t3
}

func compressTreeBytes(root *tree, b []byte) []byte {
	if root.l != nil {
		b = compressTreeBytes(root.l, b)
	}
	b = append(b, byte(root.id), byte(root.c))
	if root.l != nil {
		b = append(b, byte(root.l.id))
	} else {
		b = append(b, byte(0))
	}
	if root.r != nil {
		b = append(b, byte(root.r.id))
	} else {
		b = append(b, byte(0))
	}
	if root.r != nil {
		b = compressTreeBytes(root.r, b)
	}
	return b
}

// make a map of each character's path through the tree
func runePaths(t *tree, path string) map[rune]string {
	if t == nil {
		return nil
	}
	result := merge(runePaths(t.l, path+"0"), runePaths(t.r, path+"1"))
	if t.c != 0 {
		result[t.c] = path
	}
	return result
}

func merge(maps ...map[rune]string) map[rune]string {
	result := make(map[rune]string)
	for _, m := range maps {
		for char, path := range m {
			result[char] = path
		}
	}
	return result
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

//use the map to turn the data being compressed into 1's and 0's representing
func stringToBits(s string, m map[rune]string) []byte {
	b := make([]byte, 1)
	//eof := m['ൾ']
	var offset, i int
	for _, runeValue := range s {
		bitSequence := m[runeValue]
		for _, bit := range bitSequence {
			if offset == 8 {
				offset = 0
				var new byte
				b = append(b, new)
				i++
			}
			if bit == '0' {
				b[i] <<= 1
				offset++
			} else {
				b[i] <<= 1
				b[i] |= 1
				offset++
			}
		}
	}

	b[i] <<= (8 - offset)
	return b
}

func compress(data []byte) []byte {
	fullData := string(data) + "Þ"

	//countChars
	counts := make(map[rune]int)
	forest := make([]tree, len(fullData))
	var j int
	for _, c := range fullData {
		counts[c]++
		var charInForest bool
		for _, t := range forest {
			if c == t.c {
				charInForest = true
				break
			}
		}
		if !charInForest {
			forest[j].c = c
			j++
		}
	}
	forest = forest[:len(counts)]
	for i := 0; i < len(forest); i++ {
		forest[i].w = counts[forest[i].c]
		forest[i].id = i + 1
	}
	//makeTree
	length := len(forest)
	var tree tree
	for i := 0; len(forest) > 1; i++ {
		sort.Slice(forest, func(k, l int) bool { return forest[k].w < forest[l].w })
		tree = combineTrees(forest[0], forest[1])
		tree.id = length + i + 1
		forest[1] = tree
		forest = forest[1:]
	}
	compressedTree := make([]byte, 0, len(counts)*4)
	compressedTree = compressTreeBytes(&tree, compressedTree)
	treeEnd := []byte{1, 1, 1, 1, 0, 0, 0, 0}
	compressedTree = append(compressedTree, treeEnd...) //for decompression to know where end of tree is
	var path string
	table := runePaths(&tree, path)
	compressedData := append(compressedTree, stringToBits(fullData, table)...)
	return compressedData
}

func uncompressTree(b []byte) []*tree {
	treeFields := make([]treeInfo, 0)
	m := make(map[int]*tree)
	for i := 0; i < len(b); i += 4 {
		var ti treeInfo
		var t tree
		ti.id = int(b[i])
		t.id = int(b[i])
		ti.c = int32(b[i+1])
		t.c = int32(b[i+1])
		ti.l = int(b[i+2])
		ti.r = int(b[i+3])
		treeFields = append(treeFields, ti)
		m[t.id] = &t
	}
	newTree := make([]*tree, 0)
	for i := 0; i < len(treeFields); i++ {
		t := m[treeFields[i].id]
		if tl, ok := m[treeFields[i].l]; ok {
			t.l = tl
		}
		if tr, ok := m[treeFields[i].r]; ok {
			t.r = tr
		}
		newTree = append(newTree, t)
	}
	return newTree
}

func findRoot(newTree []*tree) *tree {
	root := newTree[0]
	for _, t := range newTree {
		if t.l == root || t.r == root {
			root = t
		}
	}
	return root
}

func unhuff(data []byte, root *tree) string {
	var unhuffedData string
	trueRoot := root
	for _, b := range data {
		comp := byte(128) //10000000
		i := 0
		for i < 8 {
			if root.l != nil || root.r != nil {
				if b&comp == comp {
					root = root.r
				} else {
					root = root.l
				}
				i++
				comp >>= 1
			} else if root.c == 'Þ' {
				break
			} else {
				unhuffedData += string(root.c)
				root = trueRoot
			}
		}
	}
	return unhuffedData
}

func findTreeEnd(b []byte) int {
	var oneByteCount int
	var oneSequenceDone bool
	var zeroByteCount int
	var treeEnd int
	for i, elem := range b {
		if elem == 1 {
			oneByteCount++
		} else if elem != 0 {
			oneByteCount = 0
		}
		if oneByteCount == 4 {
			oneSequenceDone = true
		} else {
			oneSequenceDone = false
		}
		if oneSequenceDone {
			if elem == 0 {
				zeroByteCount++
			} else {
				zeroByteCount = 0
			}
			if zeroByteCount == 4 {
				treeEnd = i - 7
			}
		}
	}
	return treeEnd
}

func decompress(data []byte) string {
	treeEnd := findTreeEnd(data)
	treeData := data[:treeEnd]
	newTree := uncompressTree(treeData)
	root := findRoot(newTree)
	content := data[treeEnd+8:]
	unhuffedData := unhuff(content, root)
	return unhuffedData
}

func main() {
	compressFile("testFile2")
	decompressFile("testFile2.huff")
}

func compressFile(fileName string) {
	f, err := os.Open(fileName)
	check(err)
	defer f.Close()
	data, err := ioutil.ReadFile(fileName)
	check(err)

	compressedData := compress(data)

	fileName = strings.TrimSuffix(fileName, ".unhuff")
	compressedFileName := fmt.Sprintf("%s.huff", fileName)
	cF, err := os.Create(compressedFileName)
	check(err)
	defer cF.Close()
	n, err := cF.Write(compressedData)
	fmt.Printf("%v bytes written\n", n)
	check(err)
}

func decompressFile(fileName string) {
	if strings.HasSuffix(fileName, ".huff") {
		f, err := os.Open(fileName)
		check(err)
		defer f.Close()
		b, err := ioutil.ReadFile(fileName)
		check(err)

		unhuffedData := decompress(b)

		uncompressedFileName := strings.Replace(fileName, ".huff", ".unhuff", -1)
		uCF, err := os.Create(uncompressedFileName)
		check(err)
		defer uCF.Close()
		n, err := uCF.WriteString(unhuffedData)
		fmt.Printf("%v bytes written\n", n)
		check(err)
	} else {
		fmt.Printf("File is not in the correct format (.huff)\n")
	}
}

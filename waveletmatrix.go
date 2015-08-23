/*
Package waveletmatrix is implementation of Wavelet-Matrix in Go.

Synopsis

	package main

	import (
		"fmt"

		"github.com/hideo55/go-waveletmatrix"
	)

	func main() {
		src := []uint64{1, 3, 1, 4, 2, 1, 10}
		wm, err := waveletmatrix.NewWM(src)
		if err != nil {
			// Failed to build wavelet-matrix
		}
		val, _ := wm.Lookup(3)
		fmt.Println(val) // 4 ... src[3]
		rank, _ := wm.Rank(2, 6)
		fmt.Println(rank) // 1 ... The number of 2 in src[0..5]
		pos, _ := wm.Select(1, 3) // = 5 ... The third 1 appeared in src[5]
		fmt.Println(pos)
	}

*/
package waveletmatrix

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"

	"github.com/hideo55/go-pq"
	"github.com/hideo55/go-sbvector"
)

// WMData holds information of Wavelet-Matrix
type WMData struct {
	size           uint64
	alphabetNum    uint64
	alphabetBitNum uint64
	bv             []*sbvector.BitVectorData
	nodePos        [][]uint64
	seps           []uint64
}

// ListResult is result of list* API
type ListResult struct {
	// The character
	C uint64
	// The frequency of c in the array
	Freq uint64
}

type queryOnNode struct {
	begNode    uint64
	endNode    uint64
	begPos     uint64
	endPos     uint64
	depth      uint64
	prefixChar uint64
}

// WaveletMatrix is interface of Wavelet-Matrix
type WaveletMatrix interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Size() uint64
	Lookup(pos uint64) (uint64, bool)
	Rank(c, pos uint64) (uint64, bool)
	RankAll(c, beginPos, endPos uint64) (rank, rankLessThan, rankMoreThan uint64)
	RankLessThan(c, pos uint64) uint64
	RankMoreThan(c, pos uint64) uint64
	Select(c, rank uint64) (uint64, bool)
	SelectFromPos(c, pos, rank uint64) (uint64, bool)
	Freq(c uint64) uint64
	FreqSum(minC, maxC uint64) uint64
	FreqRange(minC, maxC, begPos, endPos uint64) uint64
	QuantileRange(begPos, endPos, k uint64) (pos, val uint64)
	MaxRange(begPos, endPos uint64) (pos, val uint64)
	MinRange(begPos, endPos uint64) (pos, val uint64)
	ListModeRange(minC, maxC, begPos, endPos, num uint64) []ListResult
	ListMinRange(minC, maxC, begPos, endPos, num uint64) []ListResult
	ListMaxRange(minC, maxC, begPos, endPos, num uint64) []ListResult
}

const (
	// NotFound indicates `value is not found`
	NotFound uint64 = 0xFFFFFFFFFFFFFFFF

	sizeOfInt32 uint64 = 4
	sizeOfInt64 uint64 = 8
)

var (
	// ErrorInvalidFormat indicates that binary format is invalid.
	ErrorInvalidFormat = errors.New("UnmarshalBinary: invalid binary format")
)

func NewWMFromBinary(data []byte) (WaveletMatrix, error) {
	wm := new(WMData)
	err := wm.UnmarshalBinary(data)
	return wm, err
}

// Size returns size of wavelet-matrix
func (wm *WMData) Size() uint64 {
	return wm.size
}

// Lookup element by pos.
// This function returns value of pos-th element of wavelet-matrix.
// if pos >= (size of wavelet-matrix),  value of second result parameter is false.
func (wm *WMData) Lookup(pos uint64) (uint64, bool) {
	if pos >= wm.size {
		return NotFound, false
	}
	index := pos
	c := uint64(0)

	for i := 0; i < len(wm.bv); i++ {
		b, _ := wm.bv[i].Get(index)
		bit := uint64(0)
		if b {
			bit = uint64(1)
		}
		c <<= 1
		c |= bit
		index, _ = wm.bv[i].Rank(index, b)
		if b {
			index += wm.nodePos[i][1]
		}
	}
	return c, true
}

// Rank returns the frequency of a character 'c' in the prefix of the array A[0...pos)
func (wm *WMData) Rank(c, pos uint64) (uint64, bool) {
	if c >= wm.alphabetNum || pos > wm.size {
		return NotFound, false
	}

	if pos == 0 {
		return 0, false
	}

	beginPos := wm.nodePos[wm.alphabetBitNum-uint64(1)][c]
	endPos := pos

	for i := uint64(0); i < wm.alphabetBitNum; i++ {
		bv := wm.bv[i]
		bit := (c >> (wm.alphabetBitNum - i - uint64(1))) & uint64(1)
		b := toBool(bit)
		endPos, _ = bv.Rank(endPos, b)
		if b {
			endPos += wm.nodePos[i][1]
		}
	}

	return endPos - beginPos, true
}

// RankAll returns the frequency of characters c' < c, c'=c, and c' > c, in the subarray A[begPos...endPos)
func (wm *WMData) RankAll(c, beginPos, endPos uint64) (rank, rankLessThan, rankMoreThan uint64) {
	if c >= wm.alphabetNum || beginPos >= wm.size || endPos > wm.size {
		rank = NotFound
		rankLessThan = NotFound
		rankMoreThan = NotFound
		return
	}
	rank, rankLessThan, rankMoreThan = uint64(0), uint64(0), uint64(0)

	if beginPos >= endPos {
		return
	}

	begNode := uint64(0)
	endNode := wm.size
	pos := endPos

	for i := uint64(0); i < wm.alphabetBitNum && beginPos < endPos; i++ {
		bv := wm.bv[i]
		bit := (c >> (wm.alphabetBitNum - i - uint64(1))) & 1
		b := toBool(bit)
		begNodeZero, _ := bv.Rank0(begNode)
		begNodeOne := begNode - begNodeZero
		endNodeZero, _ := bv.Rank0(endNode)
		boundary := begNode + endNodeZero - begNodeZero

		rankZero, _ := bv.Rank0(pos)
		rankOne, _ := bv.Rank1(pos)

		if b {
			rankLessThan += rankZero - begNodeZero
			pos = boundary + rankOne - (begNode - begNodeZero)
			begNode = boundary
		} else {
			rankMoreThan += rankOne - begNodeOne
			pos = begNode + rankZero - begNodeZero
			endNode = boundary
		}
	}
	rank = pos - begNode
	return
}

// RankLessThan returns the frequency of characters c' < c in the subarray A[0...pos)
func (wm *WMData) RankLessThan(c, pos uint64) uint64 {
	_, rank, _ := wm.RankAll(c, 0, pos)
	return rank
}

// RankMoreThan returns the frequency of characters c' > c in the subarray A[0...pos)
func (wm *WMData) RankMoreThan(c, pos uint64) uint64 {
	_, _, rank := wm.RankAll(c, 0, pos)
	return rank
}

// Select returns the position of the (rank+1)-th occurrence of `c` in the array.
func (wm *WMData) Select(c, rank uint64) (uint64, bool) {
	return wm.SelectFromPos(c, 0, rank)
}

// SelectFromPos returns the position of the (rank+1)-th occurrence of `c` in the suffix of the array starting from 'pos'
func (wm *WMData) SelectFromPos(c, pos, rank uint64) (uint64, bool) {
	if c >= wm.alphabetNum || pos >= wm.size || rank > wm.Freq(c) {
		return NotFound, false
	}

	index := uint64(0)
	if pos == 0 {
		index = wm.nodePos[wm.alphabetBitNum-uint64(1)][c]
	} else {
		index = pos
		for i := uint64(0); i < wm.alphabetBitNum; i++ {
			bit := (c >> (wm.alphabetBitNum - i + uint64(1))) & 1
			b := toBool(bit)
			index, _ = wm.bv[i].Rank(index, b)
			if b {
				index += wm.nodePos[i][1]
			}
		}
	}

	index += rank

	for i := int(wm.alphabetBitNum) - 1; i >= 0; i-- {
		bit := (c >> (wm.alphabetBitNum - uint64(i) - uint64(1))) & 1
		b := toBool(bit)
		if b {
			index -= wm.nodePos[i][1]
		}
		var err error
		index, err = wm.bv[i].Select(index-uint64(1), b)
		if err != nil {
			return NotFound, false
		}
		index++
	}
	return index - uint64(1), true
}

// Freq returns the frequency of the character `c`.
func (wm *WMData) Freq(c uint64) uint64 {
	rank, _ := wm.Rank(c, wm.size)
	return rank
}

// FreqSum returns frequency of the characters(minC <= c' < maxC)
func (wm *WMData) FreqSum(minC, maxC uint64) uint64 {
	sum := uint64(0)
	for i := minC; i < maxC; i++ {
		sum += wm.Freq(i)
	}
	return sum
}

// FreqRange returns the frequency of characters minC <= c' < maxC in the subarray A[begPos ... endPos)
func (wm *WMData) FreqRange(minC, maxC, begPos, endPos uint64) uint64 {
	if minC >= wm.alphabetNum {
		return uint64(0)
	}
	if maxC <= minC {
		return uint64(0)
	}
	if endPos > wm.size || begPos >= endPos {
		return uint64(0)
	}
	_, maxLess, _ := wm.RankAll(maxC, begPos, endPos)
	_, minLess, _ := wm.RankAll(minC, begPos, endPos)
	return maxLess - minLess
}

// QuantileRange returns the K-th smallest value( and position) in the subarray A[begPos ... endPos)
func (wm *WMData) QuantileRange(begPos, endPos, k uint64) (pos, val uint64) {
	if endPos >= wm.size || begPos >= endPos || k >= (endPos-begPos) {
		pos = NotFound
		val = NotFound
		return
	}

	val = 0

	nodeNum := uint64(0)
	begZero := uint64(0)
	endZero := uint64(0)

	fromZero := (begPos == 0)
	toEnd := (endPos == wm.size)

	for i := uint64(0); i < wm.alphabetBitNum; i++ {
		bv := wm.bv[i]

		if fromZero {
			begZero = wm.nodePos[i][nodeNum]
		} else {
			begZero, _ = bv.Rank0(begPos)
		}

		if toEnd {
			endZero = wm.nodePos[i][nodeNum+uint64(1)]
		} else {
			endZero, _ = bv.Rank0(endPos)
		}

		zeroBits := endZero - begZero
		bit := uint64(1)
		if k < zeroBits {
			bit = uint64(0)
		}
		if bit == uint64(1) {
			k -= zeroBits
			begPos += wm.nodePos[i][1] - begZero
			endPos += wm.nodePos[i][1] - endZero
		} else {
			begPos = begZero
			endPos = endZero
		}

		nodeNum |= bit << i
		val <<= 1
		val |= bit
	}
	pos, _ = wm.Select(val, begPos+k-wm.nodePos[wm.alphabetBitNum-uint64(1)][val]+uint64(1))
	return
}

// MaxRange returns maximum value(and position) in the subarray A[begPos .. endPos]
func (wm *WMData) MaxRange(begPos, endPos uint64) (pos, val uint64) {
	pos, val = wm.QuantileRange(begPos, endPos, endPos-begPos-uint64(1))
	return
}

// MinRange returns minimum value(and position) in the subarray A[begPos .. endPos]
func (wm *WMData) MinRange(begPos, endPos uint64) (pos, val uint64) {
	pos, val = wm.QuantileRange(begPos, endPos, 0)
	return
}

func (wm *WMData) listRange(minC, maxC, begPos, endPos, num uint64, comparator pq.CmpFunc) []ListResult {
	var res []ListResult
	if endPos > wm.size || begPos >= endPos {
		return res
	}

	q := pq.NewPriorityQueue(comparator)
	q.Push(&queryOnNode{0, wm.size, begPos, endPos, 0, 0})
	for uint64(len(res)) < num && !q.Empty() {
		qon := q.Pop().(*queryOnNode)
		if qon.depth >= wm.alphabetBitNum {
			res = append(res, ListResult{qon.prefixChar, qon.endPos - qon.begPos})
		} else {
			next := wm.expandNode(minC, maxC, qon)
			for _, n := range next {
				q.Push(n)
			}
		}
	}

	return res
}

// ListModeRange returns list of the distinct characters appeared in A[begPos ... endPos) from most frequent ones.
func (wm *WMData) ListModeRange(minC, maxC, begPos, endPos, num uint64) []ListResult {
	return wm.listRange(minC, maxC, begPos, endPos, num, modeComparator)
}

// ListMinRange returns list of the distinct characters in A[begPos ... endPos) minC <= c < maxC  from smallest ones.
func (wm *WMData) ListMinRange(minC, maxC, begPos, endPos, num uint64) []ListResult {
	return wm.listRange(minC, maxC, begPos, endPos, num, minComparator)
}

// ListMaxRange returns list of the distinct characters appeared in A[begPos ... endPos) from largest ones.
func (wm *WMData) ListMaxRange(minC, maxC, begPos, endPos, num uint64) []ListResult {
	return wm.listRange(minC, maxC, begPos, endPos, num, maxComparator)
}

func (wm *WMData) expandNode(minC, maxC uint64, qon *queryOnNode) []*queryOnNode {
	bv := wm.bv[qon.depth]
	begNodeZero, _ := bv.Rank0(qon.begNode)
	endNodeZero, _ := bv.Rank0(qon.endNode)
	begNodeOne := qon.begNode - begNodeZero
	begZero, _ := bv.Rank0(qon.begPos)
	endZero, _ := bv.Rank0(qon.endPos)
	begOne := qon.begPos - begZero
	endOne := qon.endPos - endZero
	boundary := qon.begNode + endNodeZero - begNodeZero
	var next []*queryOnNode
	if (endZero - begZero) > 0 {
		nextPrefix := qon.prefixChar << 1
		if wm.checkPrefix(nextPrefix, qon.depth+1, minC, maxC) {
			next = append(next, &queryOnNode{qon.begNode, boundary, qon.begNode + begZero - begNodeZero, qon.begNode + endZero - begNodeZero, qon.depth + 1, nextPrefix})
		}

	}
	if (endOne - begOne) > 0 {
		nextPrefix := (qon.prefixChar << 1) + uint64(1)
		if wm.checkPrefix(nextPrefix, qon.depth+1, minC, maxC) {
			next = append(next, &queryOnNode{boundary, qon.endNode, boundary + begOne - begNodeOne, boundary + endOne - begNodeOne, qon.depth + 1, nextPrefix})
		}
	}
	return next
}

func prefixCode(x, size, bitNum uint64) uint64 {
	return x >> (bitNum - size)
}

func (wm *WMData) checkPrefix(prefix, depth, minC, maxC uint64) bool {
	if prefixCode(minC, depth, wm.alphabetBitNum) <= prefix && prefixCode(maxC-uint64(1), depth, wm.alphabetBitNum) >= prefix {
		return true
	}
	return false
}

func (wm *WMData) MarshalBinary() ([]byte, error) {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, &wm.size)
	binary.Write(buffer, binary.LittleEndian, &wm.alphabetNum)
	binary.Write(buffer, binary.LittleEndian, &wm.alphabetBitNum)
	bvSize := uint64(len(wm.bv))
	binary.Write(buffer, binary.LittleEndian, &bvSize)
	for i := uint64(0); i < bvSize; i++ {
		buf, _ := wm.bv[i].MarshalBinary()
		vsize := uint64(len(buf))
		binary.Write(buffer, binary.LittleEndian, &vsize)
		binary.Write(buffer, binary.LittleEndian, buf)
	}
	npSize := uint64(len(wm.nodePos))
	binary.Write(buffer, binary.LittleEndian, &npSize)
	for i := uint64(0); i < npSize; i++ {
		arraySize := uint64(len(wm.nodePos[i]))
		binary.Write(buffer, binary.LittleEndian, &arraySize)
		for j := uint64(0); j < arraySize; j++ {
			binary.Write(buffer, binary.LittleEndian, &(wm.nodePos[i][j]))
		}
	}
	sepSize := uint64(len(wm.seps))
	binary.Write(buffer, binary.LittleEndian, &sepSize)
	for i := uint64(0); i < sepSize; i++ {
		binary.Write(buffer, binary.LittleEndian, &(wm.seps[i]))
	}

	return buffer.Bytes(), nil
}

func (wm *WMData) UnmarshalBinary(data []byte) error {
	dataLen := uint64(len(data))
	offset := uint64(0)
	if dataLen < offset+sizeOfInt64 {
		return ErrorInvalidFormat
	}
	buf := data[offset : offset+sizeOfInt64]
	offset += sizeOfInt64
	wm.size = binary.LittleEndian.Uint64(buf)

	if dataLen < offset+sizeOfInt64 {
		return ErrorInvalidFormat
	}
	buf = data[offset : offset+sizeOfInt64]
	offset += sizeOfInt64
	wm.alphabetNum = binary.LittleEndian.Uint64(buf)

	if dataLen < offset+sizeOfInt64 {
		return ErrorInvalidFormat
	}
	buf = data[offset : offset+sizeOfInt64]
	offset += sizeOfInt64
	wm.alphabetBitNum = binary.LittleEndian.Uint64(buf)

	if dataLen < offset+sizeOfInt64 {
		return ErrorInvalidFormat
	}
	buf = data[offset : offset+sizeOfInt64]
	offset += sizeOfInt64
	bvSize := binary.LittleEndian.Uint64(buf)
	wm.bv = make([]*sbvector.BitVectorData, bvSize)

	for i := uint64(0); i < bvSize; i++ {
		if dataLen < offset+sizeOfInt64 {
			return ErrorInvalidFormat
		}
		buf := data[offset : offset+sizeOfInt64]
		offset += sizeOfInt64
		vsize := binary.LittleEndian.Uint64(buf)

		if dataLen < offset+vsize {
			return ErrorInvalidFormat
		}
		buf = data[offset : offset+vsize]
		bv, err := sbvector.NewVectorFromBinary(buf)
		if err != nil {
			return ErrorInvalidFormat
		}
		wm.bv[i] = bv.(*sbvector.BitVectorData)
		offset += vsize
	}
	if dataLen < offset+sizeOfInt64 {
		return ErrorInvalidFormat
	}
	buf = data[offset : offset+sizeOfInt64]
	offset += sizeOfInt64
	npSize := binary.LittleEndian.Uint64(buf)
	wm.nodePos = make([][]uint64, npSize)
	for i := uint64(0); i < npSize; i++ {
		if dataLen < offset+sizeOfInt64 {
			return ErrorInvalidFormat
		}
		buf := data[offset : offset+sizeOfInt64]
		offset += sizeOfInt64
		arrayLen := binary.LittleEndian.Uint64(buf)
		if dataLen < offset+sizeOfInt64*arrayLen {
			return ErrorInvalidFormat
		}
		wm.nodePos[i] = make([]uint64, arrayLen)
		for j := uint64(0); j < arrayLen; j++ {
			buf := data[offset : offset+sizeOfInt64]
			wm.nodePos[i][j] = binary.LittleEndian.Uint64(buf)
			offset += sizeOfInt64
		}
	}
	if dataLen < offset+sizeOfInt64 {
		return ErrorInvalidFormat
	}
	buf = data[offset : offset+sizeOfInt64]
	offset += sizeOfInt64
	sepSize := binary.LittleEndian.Uint64(buf)
	if dataLen < offset+sepSize*sizeOfInt64 {
		return ErrorInvalidFormat
	}
	wm.seps = make([]uint64, sepSize)
	for i := uint64(0); i < sepSize; i++ {
		buf := data[offset : offset+sizeOfInt64]
		wm.seps[i] = binary.LittleEndian.Uint64(buf)
		offset += sizeOfInt64
	}

	return nil
}

func toBool(bit uint64) bool {
	if bit == 0 {
		return false
	}
	return true
}

func modeComparator(a, b interface{}) bool {
	lhs := a.(*queryOnNode)
	rhs := b.(*queryOnNode)
	if lhs.endPos-lhs.begPos != rhs.endPos-rhs.begPos {
		return lhs.endPos-lhs.begPos < rhs.endPos-rhs.begPos
	} else if lhs.depth != rhs.depth {
		return lhs.depth < rhs.depth
	} else {
		return lhs.begPos > rhs.begPos
	}
}

func minComparator(a, b interface{}) bool {
	lhs := a.(*queryOnNode)
	rhs := b.(*queryOnNode)
	if lhs.depth != rhs.depth {
		return lhs.depth < rhs.depth
	}
	return lhs.begPos > rhs.begPos

}

func maxComparator(a, b interface{}) bool {
	lhs := a.(*queryOnNode)
	rhs := b.(*queryOnNode)
	if lhs.depth != rhs.depth {
		return lhs.depth < rhs.depth
	}
	return lhs.begPos < rhs.begPos
}

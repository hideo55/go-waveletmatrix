/*
Package waveletmatrix is implementation of Wavelet-Matrix in Go.
*/
package waveletmatrix

import (
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

type WaveletMatrix interface {
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
)

func (wm *WMData) Size() uint64 {
	return wm.size
}

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

func (wm *WMData) Rank(c, pos uint64) (uint64, bool) {
	if c >= wm.alphabetNum || pos > wm.size {
		return NotFound, false
	}

	if pos == 0 {
		return 0, true
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

func (wm *WMData) RankLessThan(c, pos uint64) uint64 {
	_, rank, _ := wm.RankAll(c, 0, pos)
	return rank
}

func (wm *WMData) RankMoreThan(c, pos uint64) uint64 {
	_, _, rank := wm.RankAll(c, 0, pos)
	return rank
}

func (wm *WMData) Select(c, rank uint64) (uint64, bool) {
	return wm.SelectFromPos(c, 0, rank)
}

func (wm *WMData) SelectFromPos(c, pos, rank uint64) (uint64, bool) {
	if c >= wm.alphabetNum || pos >= wm.size {
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

func (wm *WMData) Freq(c uint64) uint64 {
	rank, _ := wm.Rank(c, wm.size)
	return rank
}

func (wm *WMData) FreqSum(minC, maxC uint64) uint64 {
	sum := uint64(0)
	for i := minC; i < maxC; i++ {
		sum += wm.Freq(i)
	}
	return sum
}

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

func (wm *WMData) MaxRange(begPos, endPos uint64) (pos, val uint64) {
	pos, val = wm.QuantileRange(begPos, endPos, endPos-begPos-uint64(1))
	return
}

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

func (wm *WMData) ListModeRange(minC, maxC, begPos, endPos, num uint64) []ListResult {
	return wm.listRange(minC, maxC, begPos, endPos, num, modeComparator)
}

func (wm *WMData) ListMinRange(minC, maxC, begPos, endPos, num uint64) []ListResult {
	return wm.listRange(minC, maxC, begPos, endPos, num, minComparator)
}

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
	if prefixCode(minC, depth, wm.alphabetBitNum) <= prefix && prefixCode(maxC, depth, wm.alphabetBitNum) >= prefix {
		return true
	}
	return false
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

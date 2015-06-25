package waveletmatrix

import (
	"github.com/hideo55/go-sbvector"
)

type WMData struct {
	size           uint64
	alphabetNum    uint64
	alphabetBitNum uint64
	bv             []*sbvector.BitVectorData
	nodePos        [][]uint64
	seps           []uint64
}

type WaveletMatrix interface {
	Lookup(pos uint64) (uint64, bool)
}

const (
	// NotFound indicates `value is not found`
	NotFound uint64 = 0xFFFFFFFFFFFFFFFF
)

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

	beginPos := wm.nodePos[wm.alphabetBitNum - uint64(1)][c]
	endPos := pos

	for i := uint64(0); i < wm.alphabetBitNum; i++ {
		bv := wm.bv[i]
		bit := (c >> (wm.alphabetBitNum - i - uint64(1))) & uint64(1)
		b := toBool(bit)
		endPos, _ := bv.Rank(endPos, b)
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
		index = wm.nodePos[wm.alphabetBitNum - uint64(1)][c]
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

	for i := wm.alphabetBitNum - uint64(1); i >= 0; i-- {
		bit := (c >> (wm.alphabetBitNum - i - uint64(1))) & 1
		b := toBool(bit)
		if b {
			index -= wm.nodePos[i][1]
		}
		var err error
		index, err = wm.bv[i].Select(index - uint64(1), b)
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

func toBool(bit uint64) bool {
	if bit == 0{
		return false
	}
	return true
}

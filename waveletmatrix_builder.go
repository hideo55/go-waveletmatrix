package waveletmatrix

import (
	"errors"
	"github.com/hideo55/go-sbvector"
)

type wmBuilderData struct {
	wm *WMData
}

type wmBuilder interface {
	Build(src []uint64) (WaveletMatrix, error)
}

var (
	ErrorEmpty = errors.New("Argument is empty.")
)

func NewWM(src []uint64) (WaveletMatrix, error) {
	builder := &wmBuilderData{}
	builder.wm = &WMData{}
	return builder.Build(src)
}

func (builder *wmBuilderData) Build(src []uint64) (WaveletMatrix, error) {
	builder.wm = &WMData{}
	wm := builder.wm
	alphabetNum := getAlphabetNum(src)
	wm.alphabetNum = alphabetNum
	if alphabetNum == 0 {
		return nil, ErrorEmpty
	}

	alphabetBitNum := log2(alphabetNum)
	wm.alphabetBitNum = alphabetBitNum

	wm.size = uint64(len(src))
	wm.nodePos = make([][]uint64, alphabetBitNum)

	bvBuilders := make([]sbvector.SuccinctBitVectorBuilder, alphabetBitNum)
	for i := uint64(0); i < alphabetBitNum; i++ {
		bvBuilders[i] = sbvector.NewVectorBuilder()
	}
	wm.bv = make([]*sbvector.BitVectorData, alphabetBitNum)

	dummy := make([]uint64, 2)
	dummy[0] = uint64(0)
	dummy[1] = uint64(wm.size)
	prev_begin_pos := &dummy
	for i := uint64(0); i < alphabetBitNum; i++ {
		wm.nodePos[i] = make([]uint64, (1 << (i + 1)))
		for j := uint64(0); j < wm.size; j++ {
			bit := (src[j] >> (alphabetBitNum - i - 1)) & 1
			subscript := src[j] >> (alphabetBitNum - i)
			bvElm := true
			if bit == 0 {
				bvElm = false
			}
			bvBuilders[i].Set((*prev_begin_pos)[subscript], bvElm)
			(*prev_begin_pos)[subscript]++
			wm.nodePos[i][(subscript<<1)|bit]++
		}

		cur_max := uint64(1) << i
		rev := cur_max - uint64(1)
		prev_rev := rev

		for j := rev; j > 0; j-- {
			rev ^= cur_max - (cur_max / 2 / (j & -j))
			(*prev_begin_pos)[prev_rev] = (*prev_begin_pos)[rev]
			prev_rev = rev
		}
		(*prev_begin_pos)[0] = 0

		cur_max <<= 1
		rev = uint64(0)
		sum := uint64(0)

		for j := uint64(0); j < cur_max; rev ^= cur_max - (cur_max / 2 / (j & -j)) {
			t := wm.nodePos[i][rev]
			wm.nodePos[i][rev] = sum
			sum += t
			j++
		}
		bv, _ := bvBuilders[i].Build(true, true)
		wm.bv[i] = bv.(*sbvector.BitVectorData)
		prev_begin_pos = &((*wm).nodePos[i])
	}
	return wm, nil
}

func getAlphabetNum(array []uint64) uint64 {
	alphabetNum := uint64(0)
	for i := 0; i < len(array); i++ {
		if array[i] >= alphabetNum {
			alphabetNum = array[i] + uint64(1)
		}
	}
	return alphabetNum
}

func log2(x uint64) uint64 {
	if x == 0 {
		return 0
	}
	x--
	bitNum := uint64(0)
	for (x >> bitNum) != 0 {
		bitNum++
	}
	return bitNum
}

package main

import (
	"errors"
	"fmt"
)

type BitField []byte

func getIdxOffset(pieceIndex int) (int, int) {
	idx := pieceIndex / 8
	offset := pieceIndex % 8
	return idx, offset
}

func (bf BitField) Has(pieceIndex int) (bool, error) {
	idx, offset := getIdxOffset(pieceIndex)
	if idx < 0 || idx >= len(bf) {
		return false, errors.New(fmt.Sprintf("Out Of Bound in BitField, idx=%v.", idx))
	}
	return ((bf[idx] >> (7 - offset)) & 0x1) != 0x0, nil
}

func (bf BitField) Set(pieceIndex int) {
	idx, offset := getIdxOffset(pieceIndex)
	val := bf[idx]
	var mask byte = (0x1 << (7 - offset))
	val |= mask
	bf[idx] = val
}

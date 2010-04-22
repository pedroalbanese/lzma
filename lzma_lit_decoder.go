package lzma

import "os"

//-------------------------- litCoder2 ----------------------------------

type litCoder2 struct {
	coders []uint16
}

func newLitCoder2() *litCoder2 {
	return &litCoder2{
		coders: initBitModels(0x300),
	}
}

//------------------------ litCoder2 --- decode -------------------------

func (lc2 *litCoder2) decodeNormal(rd *rangeDecoder) (b byte, err os.Error) {
	symbol := uint32(1)
	for symbol < 0x100 {
		i, err := rd.decodeBit(lc2.coders, symbol)
		if err != nil {
			return
		}
		symbol = symbol<<1 | i
	}
	return byte(symbol), nil
}

func (lc2 *litCoder2) decodeWithMatchByte(rd *rangeDecoder, matchByte byte) (b byte, err os.Error) {
	symbol := uint32(1)
	for symbol < 0x100 {
		matchBit := uint32((matchByte >> 7) & 1)
		matchByte = matchByte << 1
		bit, err := rd.decodeBit(lc2.coders, ((1+matchBit)<<8)+symbol)
		if err != nil {
			return
		}
		symbol = (symbol << 1) | bit
		if matchBit != bit {
			for symbol < 0x100 {
				i, err := rd.decodeBit(lc2.coders, symbol)
				if err != nil {
					return
				}
				symbol = (symbol << 1) | i
			}
			break
		}
	}
	return byte(symbol), nil
}

//------------------------- litCoder2 --- encode ------------------------

func (lc2 *litCoder2) encode(re *rangeEncoder, symbol byte) (err os.Error) {
	context := uint32(1)
	for i := 7; i >= 0; i-- {
		bit := (symbol >> uint8(i)) & 1
		err = re.encode(lc2.coders, context, uint32(bit))
		if err != nil {
			return
		}
		context = context<<1 | uint32(bit)
	}
	return
}

func (lc2 *litCoder2) encodeMatched(re *rangeEncoder, matchByte, symbol byte) (err os.Error) {
	context := uint32(1)
	same := true
	for i := 7; i >= 0; i-- {
		bit := (symbol >> uint8(i)) & 1
		state := context
		if same == true {
			matchBit := (matchByte >> uint8(i)) & 1
			state += (1 + uint32(matchBit)) << 8
			same = false
			if matchBit == bit {
				same = true
			}
		}
		err = re.encode(lc2.coders, state, uint32(bit))
		if err != nil {
			return
		}
		context = context<<1 | uint32(bit)
	}
	return
}

func (lc2 *litCoder2) getPrice(matchMode bool, matchByte, symbol byte) uint32 {
	price := uint32(0)
	context := uint32(1)
	i := 7
	if matchMode == true {
		for ; i >= 0; i-- {
			matchBit := (matchByte >> uint8(i)) & 1
			bit := (symbol >> uint8(i)) & 1
			price += getPrice(uint32(lc2.coders[uint32(1+matchBit)<<8+context]), uint32(bit))
			context = context<<1 | uint32(bit)
			if matchBit != bit {
				i--
				break
			}
		}
	}
	for ; i >= 0; i-- {
		bit := (symbol >> uint8(i)) & 1
		price += getPrice(uint32(lc2.coders[context]), uint32(bit))
		context = context<<1 | uint32(bit)
	}
	return price
}


//--------------------------- litCoder ----------------------------------

type litCoder struct {
	coders      []*litCoder2
	numPrevBits uint32
	numPosBits  uint32
	posMask     uint32
}

func newLitCoder(numPosBits, numPrevBits uint32) *litCoder {
	numStates := uint32(1 << (numPrevBits + numPosBits))
	lc := &litCoder{
		coders:      make([]*litCoder2, numStates),
		numPrevBits: numPrevBits,
		numPosBits:  numPosBits,
		posMask:     (1 << numPosBits) - 1,
	}
	for i := uint32(0); i < numStates; i++ {
		lc.coders[i] = newLitCoder2()
	}
	return lc
}

func (lc *litCoder) getCoder(pos uint32, prevByte byte) *litCoder2 {
	return lc.coders[((pos&lc.posMask)<<lc.numPrevBits)+uint32((prevByte&0xff)>>(8-lc.numPrevBits))]
}

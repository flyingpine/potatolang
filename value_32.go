//+build 386 arm mips mipsle

package potatolang

import (
	"crypto/sha1"
	"fmt"
	"reflect"
	"unsafe"
)

const SizeofValue = 12

const (
	// Constants for multiplication: four random odd 32-bit numbers.
	m1    = 3168982561
	m2    = 3339683297
	m3    = 832293441
	m4    = 2336365089
	m5    = 0xb9ea8671
	m6    = 0xae7f19fb
	iseed = 0x930731
)

// The following code is taken from src/runtime/hash32.go

// Note: in order to get the compiler to issue rotl instructions, we
// need to constant fold the shift amount by hand.
// TODO: convince the compiler to issue rotl instructions after inlining.
func rotl_15(x uint32) uint32 {
	return (x << 15) | (x >> (32 - 15))
}

type hashv struct {
	a, b, c uint32
}

func (v Value) Hash() hashv {
	var a hashv
	switch v.Type() {
	case Tnumber, Tnil, Tclosure, Tmap, Tgeneric:
		a = *(*hashv)(unsafe.Pointer(&v))
	case Tstring:
		if byte(v.num)>>4 > 0 {
			a = *(*hashv)(unsafe.Pointer(&v))
			break
		}

		hdr := (*reflect.StringHeader)(v.ptr)
		seed := uintptr(iseed)
		s := uintptr(hdr.Len)
		p := unsafe.Pointer(hdr.Data)
		h := uint32(seed + s*hashkey[0])
		h1 := uint32(seed>>1 + s*hashkey[0])
		h2 := uint32(seed>>2 + s*hashkey[0])

	tail:
		switch {
		case s == 0:
		case s < 4:
			h ^= uint32(*(*byte)(p))
			h ^= uint32(*(*byte)(add(p, s>>1))) << 8
			h ^= uint32(*(*byte)(add(p, s-1))) << 16
			h = rotl_15(h*m1) * m2
			h1 ^= h
		case s == 4:
			h ^= readUnaligned32(p)
			h = rotl_15(h*m1) * m2
			h1 ^= h
		case s <= 8:
			h ^= readUnaligned32(p)
			h = rotl_15(h*m1) * m2
			h ^= readUnaligned32(add(p, s-4))
			h = rotl_15(h*m1) * m2
			h2 ^= h
		case s <= 16:
			h ^= readUnaligned32(p)
			h = rotl_15(h*m1) * m2
			h ^= readUnaligned32(add(p, 4))
			h = rotl_15(h*m1) * m2
			h ^= readUnaligned32(add(p, s-8))
			h = rotl_15(h*m1) * m2
			h ^= readUnaligned32(add(p, s-4))
			h = rotl_15(h*m1) * m2
			h2 ^= h
		default:
			v1 := h
			v2 := uint32(seed * hashkey[1])
			v3 := uint32(seed * hashkey[2])
			v4 := uint32(seed * hashkey[3])
			for s >= 16 {
				v1 ^= readUnaligned32(p)
				v1 = rotl_15(v1*m1) * m2
				p = add(p, 4)
				v2 ^= readUnaligned32(p)
				v2 = rotl_15(v2*m2) * m3
				p = add(p, 4)
				v3 ^= readUnaligned32(p)
				v3 = rotl_15(v3*m3) * m4
				p = add(p, 4)
				v4 ^= readUnaligned32(p)
				v4 = rotl_15(v4*m4) * m1
				p = add(p, 4)
				s -= 16
			}
			h = v1 ^ v2 ^ v3 ^ v4
			h2 ^= h * m5
			h1 ^= h * m6
			goto tail
		}

		a = hashv{rt(h), rt(h1), rt(h2)}
	}
	return a
}

func rt(h uint32) uint32 {
	h ^= h >> 17
	h *= m3
	h ^= h >> 13
	h *= m4
	h ^= h >> 16
	return h
}

func (v Value) hashstr() string {
	h := v.Hash()
	return fmt.Sprintf("%x", *(*[12]byte)(unsafe.Pointer(&h)))
}

func (v Value) hash2() [2]uint64 {
	h := v.Hash()
	b := *(*[12]byte)(unsafe.Pointer(&h))
	s := sha1.Sum(append(b[:], hash2Salt...))
	return *(*[2]uint64)(unsafe.Pointer(&s)) // 20 > 16
}

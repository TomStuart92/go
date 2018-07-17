package bloomfilter

import (
	"math"

	"github.com/spaolacci/murmur3"
	"github.com/willf/bitset"
)

/*
A BloomFilter is a probabilistic data structure for tracking members of a set.
It may return false positives i.e. returns true when a member is not a member of the set,
but will never return false negatives.
*/

// BloomFilter main struct
type BloomFilter struct {
	Capacity         uint64
	ErrRate          float64
	NumBits          uint64
	NumHashFunctions uint64
	ByteArray        *bitset.BitSet
}

// NewBloomFilter creates a new optimised BloomFilter for a given number of items and ErrRate
func NewBloomFilter(Capacity uint64, ErrRate float64) *BloomFilter {
	NumBits := -float64(Capacity) * math.Log(ErrRate) / (math.Pow(math.Log(2.0), 2.0))
	NumHashFunctions := NumBits / float64(Capacity) * math.Log(2.0)
	return &BloomFilter{
		Capacity:         Capacity,
		ErrRate:          ErrRate,
		NumBits:          uint64(NumBits + 1),
		NumHashFunctions: uint64(NumHashFunctions + 1),
	}
}

// Add adds a string to the BloomFilter Set
func (filter *BloomFilter) Add(s string) *BloomFilter {
	if filter.ByteArray == nil {
		filter.ByteArray = bitset.New(uint(filter.NumBits))
	}
	hashes := baseHashes([]byte(s))
	for i := uint64(0); i < filter.NumHashFunctions; i++ {
		location := computeKMLocation(hashes, uint64(i))
		byteIndex := uint(location % uint64(filter.NumBits))
		filter.ByteArray.Set(byteIndex)
	}
	return filter
}

/*
Use the Kirsch-Mitzenmacher-Optimization to compute the kth hash.
Using the base hashes h1(x) and h2(x), we can calculate
gi(x) = h1(x)+ i * h2(x), where 0 < i <= k - 1
*/
func computeKMLocation(hashes [4]uint64, k uint64) uint64 {
	return hashes[k%2] + k*hashes[2+(((k+(k%2))%4)/2)]
}

/*
MayContain returns true if set may contain element,
but false if it definely does not contain element
*/
func (filter *BloomFilter) MayContain(s string) bool {
	if filter.ByteArray == nil { // not yet initialized. Cannot contain element.
		return false
	}
	hashes := baseHashes([]byte(s))
	for i := uint64(0); i < filter.NumHashFunctions; i++ {
		location := computeKMLocation(hashes, uint64(i))
		byteIndex := uint(location % uint64(filter.NumBits))
		if !filter.ByteArray.Test(byteIndex) {
			return false
		}
	}
	return true
}

func baseHashes(data []byte) [4]uint64 {
	a1 := []byte{1} // to grab another bit of data
	hasher := murmur3.New128()
	hasher.Write(data) // #nosec
	v1, v2 := hasher.Sum128()
	hasher.Write(a1) // #nosec
	v3, v4 := hasher.Sum128()
	return [4]uint64{
		v1, v2, v3, v4,
	}
}

package utils

import (
	"hash/fnv"
)

func Hash(signature []byte, input []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(input)
	hasher.Write(signature)
	return hasher.Sum32()
}

package utils

import "hash/fnv"

func Hash(input []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(input)
	return hasher.Sum32()
}
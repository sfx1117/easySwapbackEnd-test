package utils

type ChainIdMap map[int]string

var ChainIdToChain = ChainIdMap{
	1:        "eth",
	10:       "optimism",
	11155111: "sepolia",
}

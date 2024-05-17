package wzexplorer

type KVPair struct {
	Key   string
	Value Object
}

type Properties[T any] []T

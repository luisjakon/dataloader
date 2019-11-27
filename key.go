package dataloader

// Key is an interface each element identifier must implement in order to be stored and cached
// in the ResultsMap
type Key interface {
	// String should return a guaranteed unique string that can be used to identify
	// the element. It's purpose is to identify each record when storing the results.
	// Records which should be different but share the same key will be overwritten.
	String() string

	// Raw should return real value of the key.
	Raw() interface{}
}

// Keys wraps an array of keys and contains accessor methods
type Keys struct {
	keys []Key
}

type StringKey string

func (k StringKey) String() string {
	return string(k)
}

func (k StringKey) Raw() interface{} {
	return k
}

// NewKeys returns a new instance of the Keys array with the provided capacity.
func NewKeys(capacity int) Keys {
	return Keys{make([]Key, 0, capacity)}
}

// NewKeysWith is a helper method for returning a new keys array which includes the
// the provided keys
func NewKeysWith(key ...Key) Keys {
	return Keys{append([]Key{}, key...)}
}

// ================================== public methods ==================================

func (k *Keys) Append(keys ...Key) {
	ks := make([]Key, 0)
	for _, key := range keys {
		if key == nil && key.Raw() == nil { // don't track nil keys
			continue
		}
		for _, kk := range k.keys { // skip duplicates
			if kk == key {
				return
			}
		}
		ks = append(ks, key)
	}
	k.keys = append(k.keys, ks...)
}

func (k Keys) Capacity() int {
	return cap(k.keys)
}

func (k Keys) Length() int {
	return len(k.keys)
}

func (k Keys) ClearAll() {
	k.keys = make([]Key, 0, len(k.keys))
}

func (k Keys) Keys() []Key {
	return k.keys
}

func (k Keys) RawKeys() []interface{} {
	result := make([]interface{}, 0, k.Length())
	for _, key := range k.keys {
		result = append(result, key.Raw())
	}
	return result
}

func (k Keys) StringKeys() []string {
	result := make([]string, 0, k.Length())
	for _, key := range k.keys {
		result = append(result, key.String())
	}
	return result
}

func (k Keys) IsEmpty() bool {
	return len(k.keys) == 0
}

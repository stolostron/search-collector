package transforms

// Takes a map keyed by string and flattens it, adding the given prefix before each key.
// Used for labels, roles, etc.
func flattenStringMap(prefix string, m map[string]string) map[string]string {
	ret := make(map[string]string)
	for k, v := range m {
		ret[prefix+k] = v
	}
	return ret
}

package pkg

func mergeBytes(ks ...[]byte) []byte {
	var ret []byte
	for _, k := range ks {
		if k == nil {
			continue
		}
		ret = append(ret, k...)
	}
	return ret
}

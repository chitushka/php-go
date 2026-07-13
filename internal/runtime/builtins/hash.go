package builtins

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

func md5Fn(args []Value) (Value, error) {
	if err := checkArgCount("md5", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	sum := md5.Sum([]byte(args[0].String()))
	return StringValue(hex.EncodeToString(sum[:])), nil
}

func sha1Fn(args []Value) (Value, error) {
	if err := checkArgCount("sha1", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	sum := sha1.Sum([]byte(args[0].String()))
	return StringValue(hex.EncodeToString(sum[:])), nil
}

func hashFn(args []Value) (Value, error) {
	if err := checkArgCount("hash", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	h, err := hashAlgorithm(args[0].String())
	if err != nil {
		return NullValue(), err
	}
	h.Write([]byte(args[1].String()))
	return StringValue(hex.EncodeToString(h.Sum(nil))), nil
}

func hashAlgorithm(name string) (hash.Hash, error) {
	switch strings.ToLower(name) {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha224":
		return sha256.New224(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha384":
		return sha512.New384(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %s", name)
	}
}

package util

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Return a slice of range [0, n)
func Range(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		s[i] = i
	}
	return s
}

func ShuffleSlice(a []int) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
}

func CopyFile(srcPath, dstPath string) error {
	fin, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	return err
}

func LatestVersionTag(latestReleaseURL string) string {
	resp, err := http.Head(latestReleaseURL)
	if err != nil {
		log.Printf("failed to check for newest version: %s", err.Error())
		return ""
	}
	url := resp.Request.URL.String()
	url = strings.TrimSuffix(url, "/")
	idx := strings.LastIndex(url, "/")
	if idx >= len(url)-1 {
		return ""
	}
	return url[idx+1:]
}

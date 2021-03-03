package proxy

import (
	"bufio"
	"errors"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"
)

// LoadProxyFile will load the file, and
func LoadProxyFile(path string) (count int, err error) {
	if path == "" {
		return 0, errors.New("proxy file path cannot be blank")
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		ProxyURLs = append(ProxyURLs, s.Text())
	}

	ProxyCount = len(ProxyURLs)

	return ProxyCount, nil
}

func createDirIfNotExist(dirpath string) (err error) {
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		return os.MkdirAll(dirpath, os.ModeDir|0755)
	}
	return nil
}

// write the full filepath, also creates non existent directory if not exist
func writeFullFilePath(fullpath string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(fullpath)
	err = createDirIfNotExist(dir)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fullpath, data, perm)
}

func getRandom(s []string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	i := r.Intn(len(s))

	return s[i]
}

// dnsName returns the DNS name in addr, if any.
func dnsName(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	return host
}

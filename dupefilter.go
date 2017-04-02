package spy

import (
	"os"
	"bufio"
	"strings"
	"github.com/ridewindx/crumb/set"
	"github.com/Sirupsen/logrus"
)

type DupeFilter interface {
	Opener
	Closer
	SeenRequest(request *Request, spider ISpider) bool
}

type FingerprintDupeFilter struct {
	fingerprints *set.Set
	file *os.File
	*logrus.Logger
}

func NewFingerprintDupeFilter(logger *logrus.Logger, filename ...string) *FingerprintDupeFilter {
	fingerprints := set.NewSet()
	var file *os.File
	if len(filename) > 0 {
		var err error
		file, err = os.OpenFile(filename[0], os.O_RDWR | os.O_APPEND | os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fingerprints.Add(strings.TrimSpace(scanner.Text()))
		}
		err = scanner.Err()
		if err != nil {
			panic(err)
		}
	}

	return &FingerprintDupeFilter{
		fingerprints: fingerprints,
		file: file,
		Logger: logger,
	}
}

func (f *FingerprintDupeFilter) Open(spider ISpider) {
}

func (f *FingerprintDupeFilter) Close(spider ISpider) {
	if f.file != nil {
		f.file.Close()
	}
}

func (f *FingerprintDupeFilter) SeenRequest(request *Request, spider ISpider) bool {
	fp := request.Fingerprint()
	if f.fingerprints.Contains(fp) {
		if f.Logger != nil {
			f.WithFields(logrus.Fields{
				"spider": spider,
				"request": request,
			}).Debugf("Filtered duplicate request %s", request)
		}
		return true
	}

	f.fingerprints.Add(fp)
	if f.file != nil {
		f.file.WriteString(fp+"\n")
	}
	return false
}

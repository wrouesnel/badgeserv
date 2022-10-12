package assets

import (
	"github.com/pkg/errors"
	"io/ioutil"
)

func ReadFile(name string) ([]byte, error) {
	f, err := Assets().Open(name)
	if err != nil {
		return nil, errors.Wrapf(err, "assets.ReadFile Open Error (use-filesystem: %v)", useFileSystem)
	}
	content, err := ioutil.ReadAll(f)
	return content, errors.Wrapf(err, "assets.ReadFile ReadAll Error (use-filesystem: %v)", useFileSystem)
}

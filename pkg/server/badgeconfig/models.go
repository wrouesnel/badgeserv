package badgeconfig

import (
	"net/url"

	"github.com/pkg/errors"
)

// URL is a custom URL type that allows validation at configuration load time.
type URL struct {
	*url.URL
}

func NewURL(url string) (URL, error) {
	u := URL{nil}
	err := u.UnmarshalText([]byte(url))
	return u, err
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for URLs.
func (u *URL) UnmarshalText(text []byte) error {
	urlp, err := url.Parse(string(text))

	if err != nil {
		return errors.Wrap(err, "URL.UnmarshalText failed")
	}
	u.URL = urlp
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface for URLs.
func (u *URL) MarshalText() ([]byte, error) {
	if u.URL != nil {
		return []byte(u.String()), nil
	}
	return []byte(""), nil
}

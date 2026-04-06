package yum

import (
	"fmt"
	"net/url"
)

// validateHTTPSURL parses rawURL and returns an error if it does not use https://.
// kind is a human-readable label used in error messages (e.g. "URL", "key URL", "RPM URL").
// Returns the parsed URL on success so callers avoid re-parsing.
func validateHTTPSURL(rawURL, kind string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", kind, err)
	}
	switch u.Scheme {
	case "https":
		return u, nil
	case "http":
		return nil, fmt.Errorf("%s must use https://, not http:// (rejected for security)", kind)
	case "file":
		return nil, fmt.Errorf("file:// %ss are not allowed (rejected for security)", kind)
	case "":
		return nil, fmt.Errorf("invalid %s: missing scheme (use https://)", kind)
	default:
		return nil, fmt.Errorf("%s scheme %q not allowed; use https://", kind, u.Scheme)
	}
}

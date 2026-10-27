package httpClient

import (
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/v2rayA/v2rayA/conf"
)

func HttpGetUsingSpecificClient(c *http.Client, url string) (*http.Response, error) {
	return httpGet(c, url, nil)
}

// HttpGetWithHeaders is HttpGetUsingSpecificClient with extra request headers.
// User-Agent stays v2rayA's; a caller cannot replace it.
func HttpGetWithHeaders(c *http.Client, url string, headers http.Header) (*http.Response, error) {
	return httpGet(c, url, headers)
}

func httpGet(c *http.Client, url string, headers http.Header) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	//shadowrocket会有可能不清楚alterid的情况，影响aead是否启用的问题
	req.Header.Set("User-Agent", fmt.Sprintf("v2rayA/%v WebRequestHelper", conf.Version))
	for key, values := range headers {
		if strings.EqualFold(key, "User-Agent") {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if resp, err = c.Do(req); err != nil {
		return nil, fmt.Errorf("could not fetch through the selected proxy mode (%w); set the mode to direct or start the core", err)
	}
	if resp == nil {
		cause := err
		var urlErr *neturl.Error
		if errors.As(cause, &urlErr) {
			cause = urlErr.Err
		}
		if cause == nil {
			cause = errors.New("request returned no response")
		}
		return nil, fmt.Errorf("could not reach %s: %w", req.URL.Hostname(), cause)
	}
	return
}

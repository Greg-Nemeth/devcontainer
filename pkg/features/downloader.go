package features

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const OciManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
const DockerManifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"

type OCIManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Layers        []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"layers"`
}

type OCIClient struct {
	client *http.Client
	tokens map[string]string // maps scope -> token
}

func NewOCIClient(httpClient *http.Client) *OCIClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OCIClient{
		client: httpClient,
		tokens: make(map[string]string),
	}
}

func parseBearerChallenge(headerVal string) (realm, service, scope string, err error) {
	if !strings.HasPrefix(strings.ToLower(headerVal), "bearer ") {
		return "", "", "", fmt.Errorf("invalid auth challenge prefix")
	}

	re := regexp.MustCompile(`(\w+)="([^"]+)"`)
	matches := re.FindAllStringSubmatch(headerVal, -1)

	for _, m := range matches {
		switch m[1] {
		case "realm":
			realm = m[2]
		case "service":
			service = m[2]
		case "scope":
			scope = m[2]
		}
	}

	if realm == "" {
		return "", "", "", fmt.Errorf("missing realm in bearer challenge")
	}
	return realm, service, scope, nil
}

func (c *OCIClient) getAuthToken(realm, service, scope string) (string, error) {
	cacheKey := service + ":" + scope
	if token, ok := c.tokens[cacheKey]; ok {
		return token, nil
	}

	u, err := url.Parse(realm)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if service != "" {
		q.Set("service", service)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	u.RawQuery = q.Encode()

	resp, err := c.client.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to retrieve token: status %d", resp.StatusCode)
	}

	var data struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	token := data.Token
	if token == "" {
		token = data.AccessToken
	}

	if token == "" {
		return "", fmt.Errorf("no token returned in auth response")
	}

	c.tokens[cacheKey] = token
	return token, nil
}

func (c *OCIClient) doRequestWithAuth(req *http.Request, scope string) (*http.Response, error) {
	// Clone request so original is not modified
	cloned := req.Clone(req.Context())

	// Try without token first (to see if authorized or to get challenge details)
	resp, err := c.client.Do(cloned)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		challenge := resp.Header.Get("Www-Authenticate")
		realm, service, parsedScope, err := parseBearerChallenge(challenge)
		if err != nil {
			return nil, fmt.Errorf("failed to parse auth challenge: %w", err)
		}

		if parsedScope == "" {
			parsedScope = scope
		}

		token, err := c.getAuthToken(realm, service, parsedScope)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate: %w", err)
		}

		// Retry with auth header
		reqWithAuth := req.Clone(req.Context())
		reqWithAuth.Header.Set("Authorization", "Bearer "+token)
		return c.client.Do(reqWithAuth)
	}

	return resp, nil
}

func (c *OCIClient) FetchManifest(ref FeatureRef) (*OCIManifest, error) {
	// For OCI repositories, host protocol defaults to HTTPS
	scheme := "https"
	// Check if registry looks like a local mock server host (e.g. 127.0.0.1 or localhost)
	if strings.HasPrefix(ref.Registry, "127.0.0.1") || strings.HasPrefix(ref.Registry, "localhost") {
		scheme = "http"
	}

	manifestURL := fmt.Sprintf("%s://%s/v2/%s/%s/manifests/%s", scheme, ref.Registry, ref.Namespace, ref.ID, ref.Version)
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return nil, err
	}

	// Request standard OCI manifest media types
	req.Header.Set("Accept", OciManifestMediaType+", "+DockerManifestMediaType)

	scope := fmt.Sprintf("repository:%s/%s:pull", ref.Namespace, ref.ID)
	resp, err := c.doRequestWithAuth(req, scope)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: status %d", resp.StatusCode)
	}

	var manifest OCIManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (c *OCIClient) DownloadBlob(ref FeatureRef, digest string, w io.Writer) error {
	scheme := "https"
	if strings.HasPrefix(ref.Registry, "127.0.0.1") || strings.HasPrefix(ref.Registry, "localhost") {
		scheme = "http"
	}

	blobURL := fmt.Sprintf("%s://%s/v2/%s/%s/blobs/%s", scheme, ref.Registry, ref.Namespace, ref.ID, digest)
	req, err := http.NewRequest("GET", blobURL, nil)
	if err != nil {
		return err
	}

	scope := fmt.Sprintf("repository:%s/%s:pull", ref.Namespace, ref.ID)
	resp, err := c.doRequestWithAuth(req, scope)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch blob: status %d", resp.StatusCode)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func ExtractTarGz(r io.Reader, dst string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	cleanDst := filepath.Clean(dst)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Clean(filepath.Join(dst, header.Name))
		if !strings.HasPrefix(target, cleanDst) {
			return fmt.Errorf("illegal file path (directory traversal): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}

	return nil
}

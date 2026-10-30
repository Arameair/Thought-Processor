package idclient

import (
	"context"
	"net/url"
	"time"
)

// Typed views of Identity v1 answers (only the fields Processor uses).

type Status struct {
	State            string `json:"state"`
	API              string `json:"api"`
	IdentitySystemID string `json:"identity_system_id"`
	Build            struct {
		Version  string `json:"version"`
		Branch   string `json:"branch"`
		Revision string `json:"revision"`
	} `json:"build"`
	Lifecycle struct {
		PID       int    `json:"pid"`
		Port      int    `json:"port"`
		Ownership string `json:"ownership"`
		StartedBy string `json:"started_by"`
		RunID     string `json:"run_id"`
	} `json:"lifecycle"`
	GitHub         GitHub   `json:"github"`
	NotImplemented []string `json:"not_implemented"`
	Authority      struct {
		// UpgradeAPI is set only by an Identity that can authorize in-place
		// installation upgrades (identity.installation-upgrade/1).
		UpgradeAPI string `json:"upgrade_api"`
	} `json:"authority"`
}

type GitHub struct {
	State           string     `json:"state"`
	Mode            string     `json:"mode"`
	Message         string     `json:"message,omitempty"`
	NextStep        string     `json:"next_step,omitempty"`
	UserCode        string     `json:"user_code,omitempty"`
	VerificationURI string     `json:"verification_uri,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	Account         *struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"account,omitempty"`
}

type Entity struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Kind              string     `json:"kind"`
	CreatedAt         time.Time  `json:"created_at"`
	ArchivedAt        *time.Time `json:"archived_at,omitempty"`
	EnabledFeatures   []string   `json:"enabled_features"`
	AuthorizedSources []string   `json:"authorized_sources"`
}

type Eligibility struct {
	State    string `json:"state"`
	Rule     string `json:"rule,omitempty"`
	Reason   string `json:"reason,omitempty"`
	NextStep string `json:"next_step,omitempty"`
	Meaning  string `json:"meaning,omitempty"`
}

type FeatureState struct {
	Feature            string      `json:"feature"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	Visibility         string      `json:"visibility"`
	Eligible           Eligibility `json:"eligible_to_install"`
	ServiceRegistered  string      `json:"service_registered"`
	EnabledForIdentity string      `json:"enabled_for_identity"`
	AuthorizedSources  []string    `json:"authorized_sources"`
}

type FeatureList struct {
	Identity    *Entity        `json:"identity,omitempty"`
	Features    []FeatureState `json:"features"`
	CatalogNote string         `json:"catalog_note"`
}

// Contract is Identity's feature contract, schema thought.feature/1.
type Contract struct {
	Schema      string `json:"schema"`
	FeatureID   string `json:"feature_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Publisher   string `json:"publisher"`
	Visibility  string `json:"visibility"`
	Kind        string `json:"kind"`
	Source      *struct {
		Kind             string   `json:"kind"`
		Repo             string   `json:"repo"`
		TagPrefix        string   `json:"tag_prefix"`
		AssetTemplate    string   `json:"asset_template"`
		Checksums        string   `json:"checksums"`
		TrustedUploaders []string `json:"trusted_uploaders"`
		AllowPrerelease  bool     `json:"allow_prerelease"`
	} `json:"source,omitempty"`
	Install *struct {
		Kind     string `json:"kind"`
		MaxBytes int64  `json:"max_bytes"`
	} `json:"install,omitempty"`
	WouldAskFor []string `json:"would_ask_for,omitempty"`
}

type Release struct {
	Feature        string    `json:"feature"`
	ContractSchema string    `json:"contract_schema"`
	DecisionID     string    `json:"decision_id"`
	Tag            string    `json:"tag"`
	Name           string    `json:"name"`
	PublishedAt    time.Time `json:"published_at"`
	Platform       string    `json:"platform"`
	Asset          string    `json:"asset"`
	Size           int64     `json:"size"`
	SHA256         string    `json:"sha256"`
	Source         *struct {
		Repo             string   `json:"repo"`
		Tag              string   `json:"tag"`
		URL              string   `json:"url"`
		AssetUploader    string   `json:"asset_uploader"`
		ChecksumUploader string   `json:"checksum_uploader"`
		Prerelease       bool     `json:"prerelease"`
		Checks           []string `json:"checks"`
		NotChecked       string   `json:"not_checked"`
	} `json:"source"`
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var s Status
	err := c.Get(ctx, "/api/v1/status", &s)
	return s, err
}

func (c *Client) Identities(ctx context.Context) ([]Entity, error) {
	var r struct {
		Identities []Entity `json:"identities"`
	}
	err := c.Get(ctx, "/api/v1/identities", &r)
	return r.Identities, err
}

func (c *Client) CreateIdentity(ctx context.Context, name, kind string) (Entity, error) {
	var e Entity
	err := c.Post(ctx, "/api/v1/identities", map[string]string{"name": name, "kind": kind}, &e)
	return e, err
}

func (c *Client) Features(ctx context.Context, identityID string) (FeatureList, error) {
	var f FeatureList
	path := "/api/v1/features"
	if identityID != "" {
		path = "/api/v1/identities/" + url.PathEscape(identityID) + "/features"
	}
	err := c.Get(ctx, path, &f)
	return f, err
}

func (c *Client) Contract(ctx context.Context, feature string) (Contract, error) {
	var k Contract
	err := c.Get(ctx, "/api/v1/features/"+url.PathEscape(feature)+"/contract", &k)
	return k, err
}

func (c *Client) Release(ctx context.Context, feature, platform string) (Release, error) {
	var r Release
	err := c.Get(ctx, "/api/v1/features/"+url.PathEscape(feature)+"/release?platform="+url.QueryEscape(platform), &r)
	return r, err
}

func (c *Client) DownloadPath(feature, platform, tag string) string {
	return "/api/v1/features/" + url.PathEscape(feature) + "/release/download?platform=" + url.QueryEscape(platform) + "&tag=" + url.QueryEscape(tag)
}

func (c *Client) GitHub(ctx context.Context) (GitHub, error) {
	var g GitHub
	err := c.Get(ctx, "/api/v1/github", &g)
	return g, err
}

func (c *Client) GitHubConnect(ctx context.Context) (GitHub, error) {
	var g GitHub
	err := c.Post(ctx, "/api/v1/github/connect", nil, &g)
	return g, err
}

func (c *Client) GitHubCancel(ctx context.Context) error {
	return c.Post(ctx, "/api/v1/github/cancel", nil, nil)
}

func (c *Client) Unpair(ctx context.Context) error {
	return c.Post(ctx, "/api/v1/unpair", nil, nil)
}

func (c *Client) Shutdown(ctx context.Context, reason string) error {
	return c.Post(ctx, "/api/v1/shutdown", map[string]string{"confirm": "shut-down-identity", "reason": reason}, nil)
}

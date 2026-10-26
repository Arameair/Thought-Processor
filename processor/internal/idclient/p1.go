package idclient

// P1 Identity API: Admin Identity, approvals, installation identities,
// grants and tickets. Every answer goes through Call, so it is accepted only
// when it carries a valid proof from the paired Identity installation.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

type AdminBinding struct {
	ID         int64     `json:"id"`
	Login      string    `json:"login_label"`
	Mode       string    `json:"mode"`
	VerifiedAt time.Time `json:"verified_at"`
}

type AdminView struct {
	State   string `json:"state"`
	Meaning string `json:"meaning"`
	Error   string `json:"error,omitempty"`
	Admin   *struct {
		AdminID       string       `json:"admin_id"`
		EstablishedAt time.Time    `json:"established_at"`
		GitHub        AdminBinding `json:"github_binding"`
		Authenticator string       `json:"authenticator"`
	} `json:"admin,omitempty"`
	CurrentGitHub map[string]any      `json:"current_github"`
	Options       []map[string]string `json:"authenticator_options"`
	Recovery      string              `json:"recovery"`
}

type Approval struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	Action      json.RawMessage `json:"action"`
	Digest      string          `json:"action_sha256"`
	Summary     []string        `json:"summary"`
	RequestedBy string          `json:"requested_by"`
	CreatedAt   time.Time       `json:"created_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
	State       string          `json:"state"`
	DecidedBy   string          `json:"decided_by,omitempty"`
	Result      string          `json:"result,omitempty"`
	Outcome     string          `json:"outcome,omitempty"`
}

type ApprovalRequest struct {
	Approval    Approval `json:"approval"`
	ApprovalURL string   `json:"approval_url"`
}

type Capability struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	DataClass   string `json:"data_class"`
}

type InstallRequest struct {
	FeatureID      string       `json:"feature_id"`
	Name           string       `json:"name"`
	Version        string       `json:"version"`
	API            string       `json:"api"`
	PublisherClaim string       `json:"publisher_claim"`
	SourceKind     string       `json:"source_kind"`
	SourceRef      string       `json:"source_ref"`
	TrustClass     string       `json:"trust_class"`
	ArtifactSHA256 string       `json:"artifact_sha256"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	Capabilities   []Capability `json:"capabilities"`
	InstallKey     string       `json:"install_key"`
}

type Installation struct {
	ID             string       `json:"installation_id"`
	FeatureID      string       `json:"feature_id"`
	Name           string       `json:"name"`
	Version        string       `json:"version"`
	API            string       `json:"api"`
	PublisherClaim string       `json:"publisher_claim"`
	PublisherProof string       `json:"publisher_proof"`
	SourceKind     string       `json:"source_kind"`
	SourceRef      string       `json:"source_ref"`
	TrustClass     string       `json:"trust_class"`
	ArtifactSHA256 string       `json:"artifact_sha256"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	Capabilities   []Capability `json:"capabilities"`
	InstallKey     string       `json:"install_key"`
	ApprovalID     string       `json:"approval_id"`
	AdministeredBy string       `json:"administered_by"`
	State          string       `json:"state"`
	CreatedAt      time.Time    `json:"created_at"`
	ConfirmedAt    *time.Time   `json:"confirmed_at,omitempty"`
	EndedAt        *time.Time   `json:"ended_at,omitempty"`
	CredentialGen  int          `json:"credential_generation"`
}

type Grant struct {
	ID           string     `json:"grant_id"`
	Version      int        `json:"version"`
	Requester    string     `json:"requester"`
	Destination  string     `json:"destination_installation_id"`
	Subject      string     `json:"subject_identity_id"`
	Capability   string     `json:"capability"`
	SourceScope  string     `json:"source_scope"`
	ApprovedBy   string     `json:"approved_by_admin_id"`
	ApprovalID   string     `json:"approval_id"`
	RequestedBy  string     `json:"requested_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	UntilRevoked bool       `json:"until_revoked,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    string     `json:"revoked_by,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
}

type GrantView struct {
	Grant  Grant  `json:"grant"`
	Status string `json:"status"`
}

type InstallationView struct {
	Installation Installation     `json:"installation"`
	Grants       []GrantView      `json:"grants"`
	ActiveGrants int              `json:"active_grants"`
	Sources      []string         `json:"information_sources"`
	Credential   bool             `json:"credential_registered"`
	Decisions    []map[string]any `json:"decisions,omitempty"`
}

type Credential struct {
	InstallationID   string `json:"installation_id"`
	Secret           string `json:"secret"`
	IdentitySystemID string `json:"identity_system_id"`
	Generation       int    `json:"generation"`
}

type InstallCommit struct {
	Installation Installation `json:"installation"`
	Created      bool         `json:"created"`
	Credential   *Credential  `json:"credential"`
}

type Ticket struct {
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	DecisionID string    `json:"decision_id"`
	Ticket     string    `json:"ticket,omitempty"`
	TicketID   string    `json:"ticket_id,omitempty"`
	GrantID    string    `json:"grant_id,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

// CheckApprovalURL accepts only a link to the paired Identity's own approval
// page; Processor never opens anything else as an "approval".
func (c *Client) CheckApprovalURL(u string) error {
	if !strings.HasPrefix(u, c.p.BaseURL+"/approve#a=apr_") || strings.ContainsAny(u, " \"<>'") {
		return &Error{Kind: WrongService, Msg: "Identity returned an approval link that is not its own approval page; refused"}
	}
	return nil
}

// decodeSigned decodes an answer that is signed even when it is a refusal
// (Refused errors carry the verified raw body).
func decodeSigned(raw []byte, err error, out any) error {
	return decodeSignedFor(nil, raw, err, out)
}

func decodeSignedFor(c *Client, raw []byte, err error, out any) error {
	var e *Error
	if err != nil && !(errors.As(err, &e) && e.Kind == Refused && raw != nil) {
		return err
	}
	if raw != nil && out != nil {
		json.Unmarshal(raw, out)
	}
	if ar, ok := out.(*ApprovalRequest); ok && c != nil && err == nil {
		if e := c.CheckApprovalURL(ar.ApprovalURL); e != nil {
			*ar = ApprovalRequest{}
			return e
		}
	}
	return err
}

func (c *Client) Admin(ctx context.Context) (AdminView, error) {
	var v AdminView
	err := c.Get(ctx, "/api/v1/admin", &v)
	return v, err
}

func (c *Client) RequestAdminEnrollment(ctx context.Context) (ApprovalRequest, error) {
	var v ApprovalRequest
	_, raw, err := c.Call(ctx, "POST", "/api/v1/admin/enrollment", map[string]any{})
	return v, decodeSignedFor(c, raw, err, &v)
}

func (c *Client) Approval(ctx context.Context, id string) (Approval, error) {
	var v Approval
	err := c.Get(ctx, "/api/v1/approvals/"+url.PathEscape(id), &v)
	return v, err
}

func (c *Client) RequestInstall(ctx context.Context, r InstallRequest) (ApprovalRequest, error) {
	var v ApprovalRequest
	_, raw, err := c.Call(ctx, "POST", "/api/v1/installations/requests", r)
	return v, decodeSignedFor(c, raw, err, &v)
}

func (c *Client) CommitInstall(ctx context.Context, approvalID string) (InstallCommit, error) {
	var v InstallCommit
	_, raw, err := c.Call(ctx, "POST", "/api/v1/installations", map[string]string{"approval_id": approvalID})
	return v, decodeSigned(raw, err, &v)
}

func (c *Client) Installations(ctx context.Context) ([]InstallationView, error) {
	var v struct {
		Installations []InstallationView `json:"installations"`
	}
	err := c.Get(ctx, "/api/v1/installations", &v)
	return v.Installations, err
}

func (c *Client) Installation(ctx context.Context, id string) (InstallationView, error) {
	var v InstallationView
	err := c.Get(ctx, "/api/v1/installations/"+url.PathEscape(id), &v)
	return v, err
}

func (c *Client) AbandonInstall(ctx context.Context, id string) error {
	return c.Post(ctx, "/api/v1/installations/"+url.PathEscape(id)+"/abandon", nil, nil)
}

func (c *Client) RequestUninstall(ctx context.Context, id string) (ApprovalRequest, error) {
	var v ApprovalRequest
	_, raw, err := c.Call(ctx, "POST", "/api/v1/installations/"+url.PathEscape(id)+"/uninstall/requests", map[string]any{})
	return v, decodeSignedFor(c, raw, err, &v)
}

func (c *Client) CommitUninstall(ctx context.Context, id, approvalID string) error {
	return c.Post(ctx, "/api/v1/installations/"+url.PathEscape(id)+"/uninstall", map[string]string{"approval_id": approvalID}, nil)
}

// RequestGrant asks for a grant. expiresIn 0 = Identity's finite default
// (1 hour). untilRevoked asks for no expiry; the Admin must then confirm that
// separately in Identity's page.
func (c *Client) RequestGrant(ctx context.Context, dest, subject, capability string, expiresIn int64, untilRevoked bool) (ApprovalRequest, error) {
	var v ApprovalRequest
	body := map[string]any{"destination_installation_id": dest, "subject_identity_id": subject, "capability": capability}
	if untilRevoked {
		body["until_revoked"] = true
	} else if expiresIn > 0 {
		body["expires_in_seconds"] = expiresIn
	}
	_, raw, err := c.Call(ctx, "POST", "/api/v1/grants/requests", body)
	return v, decodeSignedFor(c, raw, err, &v)
}

// RequestAdminRevoke asks the Admin to revoke a grant this program may not
// revoke itself (e.g. one another paired program asked for).
func (c *Client) RequestAdminRevoke(ctx context.Context, id, reason string) (ApprovalRequest, error) {
	var v ApprovalRequest
	_, raw, err := c.Call(ctx, "POST", "/api/v1/grants/"+url.PathEscape(id)+"/revoke/requests", map[string]string{"reason": reason})
	return v, decodeSignedFor(c, raw, err, &v)
}

// CommitAdminRevoke carries out an Admin-approved revocation.
func (c *Client) CommitAdminRevoke(ctx context.Context, id, approvalID string) (GrantView, error) {
	var v GrantView
	_, raw, err := c.Call(ctx, "POST", "/api/v1/grants/"+url.PathEscape(id)+"/admin-revoke", map[string]string{"approval_id": approvalID})
	return v, decodeSigned(raw, err, &v)
}

func (c *Client) CommitGrant(ctx context.Context, approvalID string) (GrantView, error) {
	var v GrantView
	_, raw, err := c.Call(ctx, "POST", "/api/v1/grants", map[string]string{"approval_id": approvalID})
	return v, decodeSigned(raw, err, &v)
}

func (c *Client) RevokeGrant(ctx context.Context, id, reason string) error {
	return c.Post(ctx, "/api/v1/grants/"+url.PathEscape(id)+"/revoke", map[string]string{"reason": reason}, nil)
}

// RequestTicket asks for a one-request ticket. A DENIED answer is a signed
// refusal: the returned Ticket carries the reason and decision ID.
func (c *Client) RequestTicket(ctx context.Context, dest, subject, capability string, body []byte) (Ticket, error) {
	h := sha256.Sum256(body)
	var t Ticket
	_, raw, err := c.Call(ctx, "POST", "/api/v1/tickets", map[string]string{"destination_installation_id": dest,
		"subject_identity_id": subject, "capability": capability, "request_sha256": hex.EncodeToString(h[:])})
	err = decodeSigned(raw, err, &t)
	return t, err
}

func (c *Client) Decisions(ctx context.Context, installation string) ([]map[string]any, error) {
	var v struct {
		Decisions []map[string]any `json:"decisions"`
	}
	err := c.Get(ctx, "/api/v1/decisions?installation="+url.QueryEscape(installation), &v)
	return v.Decisions, err
}

// ConfirmInstallation proves to Identity that the installation credential was
// saved: the confirm request is signed WITH that credential (X-Installation-ID).
func ConfirmInstallation(ctx context.Context, baseURL string, cred Credential) (Installation, error) {
	var v struct {
		Installation Installation `json:"installation"`
	}
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/installation/confirm", bytes.NewReader([]byte("{}")))
	if err != nil {
		return Installation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	rk, sk := DeriveKeys(cred.Secret)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := store.RandHex(12)
	sum := sha256.Sum256([]byte("{}"))
	req.Header.Set("X-Installation-ID", cred.InstallationID)
	req.Header.Set(TimeHeader, ts)
	req.Header.Set(NonceHeader, nonce)
	req.Header.Set(AuthHeader, hmacHex(rk, []byte("identity/2 request\nPOST\n/api/v1/installation/confirm\n"+ts+"\n"+nonce+"\n"+hex.EncodeToString(sum[:]))))
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return Installation{}, &Error{Kind: Unavailable, Msg: "Identity didn't answer"}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == 401 {
		return Installation{}, &Error{Kind: TokenRefused, Status: 401, Msg: "Identity refused the installation credential"}
	}
	want := hmacHex(sk, []byte("identity/2 response\n"+nonce+"\n"+strconv.Itoa(resp.StatusCode)+"\n"), raw)
	if want != resp.Header.Get(ProofHeader) || resp.Header.Get(IDHeader) != cred.IdentitySystemID {
		return Installation{}, &Error{Kind: WrongService, Msg: "confirmation answer not signed by the right Identity"}
	}
	if resp.StatusCode != 200 {
		return Installation{}, &Error{Kind: Refused, Status: resp.StatusCode, Msg: string(raw)}
	}
	err = json.Unmarshal(raw, &v)
	return v.Installation, err
}

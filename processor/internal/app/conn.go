package app

// Conn keeps Processor's connection to Identity honest. It never assumes:
// every few seconds it asks the paired Identity for its signed status and
// sets one of these states. Protected actions call Require, which asks again
// right then — nothing is allowed on an old answer (fail closed).

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

const (
	CSearching    = "IDENTITY_NOT_FOUND"    // never connected; nothing answering
	CFound        = "FOUND_NOT_CONNECTED"   // Identity answering, not paired
	CCodeShown    = "WAITING_FOR_CODE"      // asked Identity to show a code
	CVerified     = "CONNECTED"             // signed status from the paired installation
	CUnavailable  = "AUTHORITY_UNAVAILABLE" // was connected; no answer now
	CRemoved      = "CONNECTION_REMOVED"    // 401: Identity forgot this Processor
	CWrong        = "WRONG_SERVICE"         // unsigned/badly signed/different installation
	CIncompatible = "INCOMPATIBLE"
	CBroken       = "PAIRING_UNREADABLE" // saved pairing can't be read (e.g. other Windows user)
)

type Conn struct {
	prof *store.Profile
	log  *Log

	mu          sync.Mutex
	pairing     *idclient.Pairing
	client      *idclient.Client
	state       string
	detail      string
	since       time.Time
	lastOK      time.Time
	status      *idclient.Status
	candidates  []idclient.Candidate
	chosen      string
	codeExpires time.Time
	identities  []idclient.Entity
	features    *idclient.FeatureList
	admin       *idclient.AdminView
	insts       []idclient.InstallationView
	featuresFor string
	dataAt      time.Time
	dataErr     string
	kick        chan struct{}
}

func NewConn(prof *store.Profile, log *Log) *Conn {
	c := &Conn{prof: prof, log: log, state: CSearching, since: time.Now(), kick: make(chan struct{}, 1)}
	p, err := idclient.LoadPairing(prof.Dir)
	switch {
	case err != nil:
		c.state, c.detail = CBroken, err.Error()
		log.Warn("identity", "pairing.unreadable", err.Error())
	case p != nil:
		c.pairing, c.client = p, idclient.New(*p)
		c.state, c.detail = CUnavailable, "checking the saved connection…"
		log.Info("identity", "pairing.loaded", "saved connection to "+p.IdentitySystemID+" (token protection: "+store.SecretScheme+")")
	}
	return c
}

func (c *Conn) setState(st, detail string) {
	if st != c.state {
		lvl := "info"
		if st == CUnavailable || st == CRemoved || st == CWrong || st == CIncompatible {
			lvl = "warn"
		}
		c.log.Add(lvl, "identity", "connection.state", c.state+" → "+st+": "+detail)
		c.since = time.Now()
	}
	c.state, c.detail = st, detail
}

// Kick asks the loop to refresh now.
func (c *Conn) Kick() {
	select {
	case c.kick <- struct{}{}:
	default:
	}
}

func (c *Conn) Run(ctx context.Context) {
	for {
		c.Tick(ctx)
		every := 3 * time.Second
		if c.State() == CVerified {
			every = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-c.kick:
		case <-time.After(every):
		}
	}
}

func (c *Conn) State() string { c.mu.Lock(); defer c.mu.Unlock(); return c.state }

// Tick refreshes the connection state (and, when connected, the identity
// list and the selected identity's features).
func (c *Conn) Tick(ctx context.Context) {
	c.mu.Lock()
	p := c.pairing
	st := c.state
	c.mu.Unlock()
	if st == CBroken {
		return
	}
	if p == nil {
		cands := idclient.Discover(ctx, c.prof.Settings().IdentityURL)
		c.mu.Lock()
		c.candidates = cands
		switch {
		case c.state == CCodeShown && time.Now().Before(c.codeExpires):
		case len(cands) == 0:
			c.setState(CSearching, "No Identity is answering on this computer.")
		default:
			c.setState(CFound, "Identity is running but this Processor isn't connected to it yet.")
		}
		c.mu.Unlock()
		return
	}
	c.check(ctx)
	if c.State() == CVerified {
		c.refreshData(ctx)
	}
}

// check asks for signed status; follows the installation if it moved port.
func (c *Conn) check(ctx context.Context) (*idclient.Status, error) {
	c.mu.Lock()
	cl := c.client
	c.mu.Unlock()
	if cl == nil {
		return nil, &idclient.Error{Kind: idclient.NotPaired, Msg: "not connected"}
	}
	s, err := cl.Status(ctx)
	if k := idclient.KindOf(err); k == idclient.Unavailable || k == idclient.WrongService || k == idclient.TokenRefused {
		// Maybe our Identity moved (new port) and something else is on the old one.
		if u := idclient.Relocate(ctx, cl.Pairing()); u != "" && u != cl.Pairing().BaseURL {
			cl.SetBase(u)
			if s2, err2 := cl.Status(ctx); err2 == nil {
				s, err = s2, nil
				c.mu.Lock()
				np := cl.Pairing()
				c.pairing = &np
				c.mu.Unlock()
				idclient.SavePairing(c.prof.Dir, np)
				c.log.Info("identity", "connection.relocated", "the connected Identity is now at "+u+" (found by its installation ID)")
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		switch idclient.KindOf(err) {
		case idclient.TokenRefused:
			c.setState(CRemoved, err.(*idclient.Error).Msg)
		case idclient.WrongService:
			c.setState(CWrong, err.(*idclient.Error).Msg)
		default:
			c.setState(CUnavailable, "Can't reach Identity. Protected actions are paused until it's back.")
		}
		return nil, err
	}
	if !idclient.Compatible(s.API) {
		c.setState(CIncompatible, "This Identity speaks "+s.API+"; this Processor needs "+idclient.APIMajor+".")
		return nil, &idclient.Error{Kind: idclient.Incompatible, Msg: c.detail}
	}
	c.status, c.lastOK = &s, time.Now()
	c.setState(CVerified, "Connected to Identity "+s.IdentitySystemID)
	return &s, nil
}

// Pairing returns the saved pairing (zero value if none).
func (c *Conn) Pairing() idclient.Pairing {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client.Pairing()
	}
	if c.pairing != nil {
		return *c.pairing
	}
	return idclient.Pairing{}
}

func (c *Conn) adminView() *idclient.AdminView {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != CVerified {
		return nil
	}
	return c.admin
}

func (c *Conn) installations() []idclient.InstallationView {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != CVerified {
		return []idclient.InstallationView{}
	}
	return append([]idclient.InstallationView{}, c.insts...)
}

// installation returns Identity's last signed view of one installation.
// It is nil while Identity can't be confirmed: stale authority facts are
// never presented (or used) as current.
func (c *Conn) installation(id string) *idclient.InstallationView {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != CVerified {
		return nil
	}
	for i := range c.insts {
		if c.insts[i].Installation.ID == id {
			v := c.insts[i]
			return &v
		}
	}
	return nil
}

func (c *Conn) refreshData(ctx context.Context) {
	c.mu.Lock()
	cl := c.client
	c.mu.Unlock()
	if cl == nil {
		return
	}
	if adm, err := cl.Admin(ctx); err == nil {
		c.mu.Lock()
		c.admin = &adm
		c.mu.Unlock()
	}
	if insts, err := cl.Installations(ctx); err == nil {
		c.mu.Lock()
		c.insts = insts
		c.mu.Unlock()
	}
	ids, err := cl.Identities(ctx)
	sel := c.prof.Settings().SelectedIdentity
	found := false
	for _, e := range ids {
		if e.ID == sel {
			found = true
		}
	}
	if err == nil && !found && sel != "" {
		c.prof.UpdateSettings(func(s *store.Settings) { s.SelectedIdentity = "" })
		sel = ""
	}
	var fl idclient.FeatureList
	var ferr error
	if err == nil {
		fl, ferr = cl.Features(ctx, sel)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err == nil && ferr == nil {
		c.identities, c.features, c.featuresFor, c.dataAt, c.dataErr = ids, &fl, sel, time.Now(), ""
		return
	}
	if err == nil {
		err = ferr
	}
	c.dataErr = err.Error()
}

// Require returns a client only if Identity confirms, right now, that this
// is the paired installation. Every protected action goes through here.
func (c *Conn) Require(ctx context.Context) (*idclient.Client, error) {
	if _, err := c.check(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client, nil
}

// ---- pairing ----

func (c *Conn) Choose(u string) error {
	u, err := idclient.LoopbackOnly(u)
	if err != nil {
		return err
	}
	c.prof.UpdateSettings(func(s *store.Settings) { s.IdentityURL = u })
	c.mu.Lock()
	c.chosen = u
	c.mu.Unlock()
	c.Kick()
	return nil
}

func (c *Conn) RequestCode(ctx context.Context, u string) error {
	if u == "" {
		c.mu.Lock()
		if len(c.candidates) == 1 {
			u = c.candidates[0].BaseURL
		} else {
			u = c.chosen
		}
		c.mu.Unlock()
	}
	u, err := idclient.LoopbackOnly(u)
	if err != nil {
		return errors.New("choose which Identity to connect to")
	}
	exp, err := idclient.RequestCode(ctx, u, c.prof.Self.ProcessorID)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.chosen, c.codeExpires = u, exp
	c.setState(CCodeShown, "Identity is showing a one-time code in its own window.")
	c.mu.Unlock()
	c.log.Info("identity", "pairing.code_requested", "asked "+u+" to show a pairing code (the code is never sent to Processor)")
	return nil
}

func (c *Conn) CompleteCode(ctx context.Context, code string) error {
	c.mu.Lock()
	u := c.chosen
	c.mu.Unlock()
	if u == "" {
		return errors.New("ask Identity for a code first")
	}
	p, err := idclient.CompletePairing(ctx, u, c.prof.Self.ProcessorID, code)
	if err != nil {
		c.log.Warn("identity", "pairing.refused", err.Error())
		return err
	}
	if err := idclient.SavePairing(c.prof.Dir, p); err != nil {
		return errors.New("connected, but the connection couldn't be saved: " + err.Error())
	}
	c.mu.Lock()
	c.pairing, c.client = &p, idclient.New(p)
	c.setState(CVerified, "Connected to Identity "+p.IdentitySystemID)
	c.mu.Unlock()
	c.log.Info("identity", "pairing.paired", "connected to Identity "+p.IdentitySystemID+" at "+u+" (token protection: "+store.SecretScheme+")")
	c.Kick()
	return nil
}

// Disconnect unpairs (if Identity is reachable) and forgets the pairing.
func (c *Conn) Disconnect(ctx context.Context) {
	c.mu.Lock()
	cl := c.client
	c.mu.Unlock()
	if cl != nil {
		if err := cl.Unpair(ctx); err != nil {
			c.log.Warn("identity", "pairing.unpair_unconfirmed", "Identity didn't confirm the disconnect ("+err.Error()+"); the local connection is removed anyway")
		}
	}
	idclient.DeletePairing(c.prof.Dir)
	c.mu.Lock()
	c.pairing, c.client, c.status, c.identities, c.features, c.admin, c.insts = nil, nil, nil, nil, nil, nil, nil
	c.setState(CSearching, "Disconnected.")
	c.mu.Unlock()
	c.log.Info("identity", "pairing.disconnected", "this Processor's connection to Identity was removed")
	c.Kick()
}

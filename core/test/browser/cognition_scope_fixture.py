"""Core-attested scope for Cognition browser fixtures (#106 lifecycle contract v2 §2).

Core answers every successful entity-scoped Cognition GET with three headers naming the Identity
entity, Engine entity and owner link it authorized the read under. The fixtures stand in for Core,
so they send the same headers. scoped() wraps a Playwright Route: a 2xx answer to an /api/ request
gets the attestation; other answers are unchanged. No assertion depends on this module.
"""
from urllib.parse import urlparse


def scope_of(lib):
    """The attestation Core sends with a library body (and with every read in that scope)."""
    link = ((lib.get("link") or {}).get("link") or {}).get("identity_decision_id") or "NONE"
    return {"X-Cognition-Identity": lib["identity_entity_id"], "X-Cognition-Engine": lib.get("engine_entity_id") or "UNLINKED",
            "X-Cognition-Link": link}


# Every pre-existing fixture uses this one Identity entity, Engine entity and no recorded link.
SCOPE = {"X-Cognition-Identity": "ent_0123", "X-Cognition-Engine": "eng_1", "X-Cognition-Link": "NONE"}


class _Scoped:
    def __init__(self, route, scope):
        self._route, self._scope = route, scope

    def __getattr__(self, name):
        return getattr(self._route, name)

    def fulfill(self, **kw):
        if urlparse(self._route.request.url).path.startswith("/api/") and 200 <= kw.get("status", 200) < 300:
            kw["headers"] = {**self._scope, **(kw.get("headers") or {})}
        return self._route.fulfill(**kw)


def scoped(route, scope=None):
    return _Scoped(route, scope or SCOPE)

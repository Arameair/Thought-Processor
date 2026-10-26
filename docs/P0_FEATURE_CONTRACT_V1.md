# P0 reusable feature install contract — `feature-install/1-alpha`

**Status:** Experimental Processor-local contract implemented on `experiment/p0-reusable-feature-onboarding-20261026`. This defines **Processor's** generic adapter boundary; it is **not** a newly invented Identity API or a publisher signature format. Owner approval and a second production feature are outstanding.

## Boundary

Processor never needs a hardcoded Engine installer to install an Identity-recognized feature. It requires:

1. **Feature ID:** lowercase URL/path-safe feature identifier supplied by authenticated Identity catalog.
2. **Eligibility:** paired, signed `GET /api/v1/features/{id}/eligibility`, response `eligible_to_install.state=ALLOWED`, plus `install_adapter` that is not `NONE`.
3. **Release metadata:** paired, signed `GET /api/v1/features/{id}/release?platform={os}-{arch}`, returning `tag`, `asset`, `size`, `sha256`, `decision_id`. Processor validates all fields, platform and size bound.
4. **User approval:** display exact tag, asset, digest, size, limitations. User explicitly confirms. On execution the release is fetched **again** and the same tag must match.
5. **Download:** paired stream `GET /api/v1/features/{id}/release/download?platform=…&tag=…`, with tag and asset headers checked, bounded body, byte count and SHA-256 check against signed metadata.
6. **Local persistence:** `<Processor profile>/features/{id}/{tag}/{asset}`, version-local `install.json` and atomic `current.json`. The receipt includes `contract: "feature-install/1-alpha"`, feature/tag/asset/hash/size, Identity installation ID and decision ID, install timestamp and `publisher_authenticity: "NOT_INDEPENDENTLY_SIGNED"`.
7. **Restart and verification:** Processor reads local receipt independently of Identity. It reports `INSTALLED_UNVERIFIED` until the operator requests full re-hash `VERIFIED`; missing/mismatched bytes are never called verified.
8. **Runtime separation:** generic install receipt says **nothing** about Engine process startup, registration, feature grants, per-identity enablement, source scopes or cognition. These need separately verified Identity/feature agreements.

## Status and error model

| Case | P0 outcome |
| --- | --- |
| Identity unavailable, pairing refused or wrong service | Do not install |
| Eligibility absent/denied/stale after reconnect | Do not install |
| No Identity install adapter | `NO_INSTALL_ADAPTER` (not a special-case Engine fallback) |
| No approved published release | `BLOCKED` / returned Identity error |
| Concurrent install | `BUSY`; one transaction per Processor profile |
| Tag changed between review and approval | Refuse; ask for new review |
| Wrong platform/asset/tag/digest/size | Refuse, leave `current.json` untouched |
| Interrupted or truncated download | Delete temporary staging file; preserve current version |
| Hash mismatch | Refuse, preserve prior valid installation |
| New version verified | Commit version metadata then atomically update `current.json` |
| Restart | Show recorded installation and require actual file verification before `VERIFIED` |

## Provenance limit

All release metadata is authenticated *to the paired Identity installation*, but `SHA256SUMS` is from the same GitHub release as the binary. The current adapter uses GitHub's access controls as **prototype acquisition eligibility**, not independent publisher-signed release attestation. A production publisher authenticity policy, signed manifest or approved trust anchor is required before asserting that publisher provenance is cryptographically verified.

The only built-in Identity provider adapter currently targets Thought Engine; the generic Processor-side interface has been tested using an unrelated **sample-feature fixture**, not a second real approved release. This distinction is essential.

## Privacy boundary

This contract requires server-side private feature catalog filtering at Identity or publisher. Processor hides unavailable private cards in the ordinary UI, but the current embedded Identity API can reveal feature names in a paired catalog response. Thus **private feature confidentiality remains BLOCKED** for public/tester release. Do not mistake frontend hiding for an authorization boundary.

## Compatible Identity code

- `Arameair/identity` `alpha-candidate-20261026`, version `0.4.0-alpha`, API `identity/1-alpha`.
- GitHub Actions real-process smoke pins Identity at full SHA `b4203e03d5004e1de9f79dcc9054394f53056af4`. This tests pairing, canonical identity, restart and outage only, **not** real GitHub or release download.
- All Processor files and tests remain in the isolated experimental branch; neither repository's owner-controlled release lanes were changed.

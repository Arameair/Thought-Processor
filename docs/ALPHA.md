# TASS alpha on Windows / WSL

This alpha combines Identity a582f8e, Processor workstation 571418a, TASS ae33d33 (0.2.0), and Thought 772200b. It adds launcher paths for alpha and local OpenAI-compatible model configuration in the workstation. This is an engineering alpha, not a production or security certification.

## Fresh installation

Use sibling clean alpha checkouts in `~/p1c/alpha/{identity,Thought-Processor,TASS,Thought}`. Run the launcher at `~/p1c/alpha/Thought-Processor/tools/p0-launcher/Start-Workbench.cmd` from Windows, or `python3 ~/p1c/alpha/Thought-Processor/tools/p0-launcher/secure_launch.py start` from WSL, then the same command with `open`.

The launcher creates a separate profile at `~/p1c/p1-tass-alpha`, Identity port 7798 and Processor port 7868. It preserves all earlier profiles. The launcher keeps private opening codes in memory and opens Edge InPrivate with a single-use ticket. Use `status`, `pairing`, `versions`, `check`, `open`, and `stop` with this same launcher. Closing the menu does not stop the services.

Pair with Identity using the private local dialog. Complete real GitHub authorization, establish your Admin in Identity, and approve TASS installation and each requested permission in Identity. No original installation credential is imported. Fresh means a new Identity-authorized installation with no inherited grants or conversation.

## IBM Granite through llama.cpp

Leave your loopback llama.cpp server on `127.0.0.1:8080`. When the workstation asks for a model, choose **llama.cpp / OpenAI-compatible**. It fills the local address and `ibm-research/granite-vision-3.2-2b-GGUF:Q4_K_M`. Choose **Use this model** after Identity approves model configuration. This still goes through `tass.model.set`, and TASS enforces its local-address policy. A provider API key is not needed for this local server.

## Private original key

The original installation credential backup belongs outside all repositories. It is not part of this alpha, cannot authorize a fresh Identity, and must never be committed, uploaded, pasted into chat, or shown in screenshots. Preserve the complete original profiles for recovery; the key backup alone is not a profile migration.

## Promotion scope and validation

Included: pinned-package checks, live Identity authorization, selected-file read-only workstation, secret-file and link protections, launcher recovery and one-time browser reopening. Automatic updates remain disabled. The separate universal-updater and installation-upgrade experiments are not included in this fresh-install alpha; they add authority and migration paths that are unnecessary for a fresh installation and need separate review.

Validation uses Go tests, TASS and Thought unit tests, launcher tests, and disposable real-process acceptance with explicitly labelled GitHub/model stand-ins. That acceptance does not prove real GitHub sign-in, Windows rendering, or Granite conversation success. Those are owner acceptance steps.

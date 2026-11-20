"""Previously completed synthetic candidate for a separate audit workstream."""
def public_status(value):
    return value if value in {"READY", "BLOCKED"} else "UNKNOWN"

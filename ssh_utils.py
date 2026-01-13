"""Helper functions for SSH key management.

This app supports multiple key files; when testing SSH connectivity we pass an
explicit identity file to avoid relying on ssh-agent or SSH config.
"""

import subprocess
from pathlib import Path
from typing import Literal

KeyAlgorithm = Literal["ed25519", "rsa", "ecdsa"]


def generate_key(
    algorithm: KeyAlgorithm = "ed25519",
    key_name: str = "id_ed25519",
    comment: str | None = None,
    force: bool = False,
) -> str:
    """Generate an SSH key with specified algorithm.

    Args:
        algorithm: Key algorithm (ed25519, rsa, ecdsa)
        key_name: Name for the key file (without .ssh/ prefix)
        comment: Optional comment to add to the key
        force: If True, overwrite existing key without prompting

    Returns:
        Status message
    """
    ssh_dir = Path.home() / ".ssh"
    ssh_dir.mkdir(exist_ok=True)
    key_path = ssh_dir / key_name

    if force and key_path.exists():
        key_path.unlink(missing_ok=True)
        Path(f"{key_path}.pub").unlink(missing_ok=True)

    cmd = ["ssh-keygen", "-t", algorithm, "-f", str(key_path), "-N", ""]
    
    # Add key size for RSA
    if algorithm == "rsa":
        cmd.extend(["-b", "4096"])
    
    if comment:
        cmd.extend(["-C", comment])

    result = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if result.returncode == 0:
        return f"Key '{key_name}' generated successfully."
    return f"Key generation failed: {result.stderr.strip()}"


def list_ssh_keys() -> list[dict[str, str]]:
    """List all SSH keys in .ssh directory.

    Returns:
        List of dicts with 'name', 'path', and 'pub_path'
    """
    ssh_dir = Path.home() / ".ssh"
    if not ssh_dir.exists():
        return []
    
    keys = []
    for pub_file in ssh_dir.glob("*.pub"):
        key_name = pub_file.stem
        key_path = ssh_dir / key_name
        if key_path.exists():
            keys.append({
                "name": key_name,
                "path": str(key_path),
                "pub_path": str(pub_file),
            })
    return keys


def load_public_key(key_name: str | None = None) -> str | None:
    """Load the public key content.

    Args:
        key_name: Name of the key (without .ssh/ prefix). If None, returns None.

    Returns:
        Public key content or None if not found
    """
    if not key_name:
        return None
    
    ssh_dir = Path.home() / ".ssh"
    pub_path = ssh_dir / f"{key_name}.pub"

    if not pub_path.exists():
        return None

    return pub_path.read_text(encoding="utf-8").strip()


def get_private_key_path(key_name: str) -> Path:
    """Return the private key path for a key name under ~/.ssh."""
    return (Path.home() / ".ssh" / key_name).expanduser()


def test_connections(key_name: str | None = None) -> dict[str, tuple[bool, str]]:
    """Test SSH authentication against supported Git hosts.

    Args:
        key_name: Optional key name (without .ssh/ prefix) to test with.
    """
    key_path: Path | None = None
    if key_name:
        candidate = get_private_key_path(key_name)
        if candidate.exists():
            key_path = candidate

    results: dict[str, tuple[bool, str]] = {}
    for host in ("github.com", "bitbucket.org"):
        results[host] = _test_ssh_host(host, key_path)
    return results


def _test_ssh_host(host: str, key_path: Path | None = None) -> tuple[bool, str]:
    """Test authentication to a Git SSH host."""
    base = ["ssh", "-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"]
    if key_path is not None:
        base.extend(["-i", str(key_path), "-o", "IdentitiesOnly=yes"])

    def run_attempt(extra: list[str], label: str) -> tuple[int, str, str]:
        cmd = base + extra + [f"git@{host}"]
        cp = subprocess.run(cmd, capture_output=True, text=True, check=False)
        combined = f"{cp.stderr}\n{cp.stdout}".strip()
        return cp.returncode, (combined or "No output"), label

    attempts: list[tuple[int, str, str]] = []
    attempts.append(run_attempt([], "port 22"))

    code, text, label = attempts[0]
    low = text.lower()

    if "successfully authenticated" in low or (
        "authenticated" in low and "shell access" in low
    ):
        return True, f"OK ({label})\n{text}"

    # If port 22 is blocked, try the vendor-supported port 443 endpoints.
    network_blocked = any(
        s in low
        for s in (
            "connection timed out",
            "operation timed out",
            "no route to host",
            "connection refused",
            "could not resolve hostname",
            "network is unreachable",
            "connect to host",
        )
    )

    if network_blocked:
        if host == "github.com":
            # GitHub: ssh.github.com:443
            attempts.append(run_attempt(["-p", "443", "-o", "Hostname=ssh.github.com"], "port 443 (ssh.github.com)"))
        if host == "bitbucket.org":
            # Bitbucket: altssh.bitbucket.org:443
            attempts.append(run_attempt(["-p", "443", "-o", "Hostname=altssh.bitbucket.org"], "port 443 (altssh.bitbucket.org)"))

    # Evaluate fallbacks, if any.
    for code, text, label in attempts[1:]:
        low = text.lower()
        if "successfully authenticated" in low or (
            "authenticated" in low and "shell access" in low
        ):
            return True, f"OK ({label})\n{text}"

    # Return the first attempt plus fallback outputs for debugging.
    combined = "\n\n".join(f"--- {lbl} ---\n{txt}" for _, txt, lbl in attempts)
    return False, combined

# Sudo/PAM authentication scenarios

`sudo` does not authenticate a password by itself. It asks PAM to authenticate the user. If PAM is configured with fingerprint support, the same sudo prompt can succeed because of a password or because of a fingerprint scan.

This means a TUI cannot honestly label the flow as "password check" unless it explicitly bypasses fingerprint PAM, which we should not do for the user's real sudo policy. The UI should present it as sudo authentication and describe password and fingerprint as possible inputs.

## Terminal behavior

With fingerprint PAM enabled, a sudo command can print prompts like:

```text
Place your finger on the fingerprint reader
```

or:

```text
Place your finger on the reader again
Failed to match fingerprint
```

If stdin is closed or no password is supplied, sudo can eventually report:

```text
sudo: no password was provided
sudo: 1 incorrect password attempt
```

Those messages can appear even when no password prompt was visibly shown, because PAM tried fingerprint first.

Observed local trace with `sudo -k && sudo -v`:

```text
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
[sudo] password for erikfrish:
Sorry, try again.
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
Place your finger on the fingerprint reader
Failed to match fingerprint
[sudo] password for erikfrish:
Sorry, try again.
Place your finger on the fingerprint reader
```

Current PAM ordering is therefore:

1. Fingerprint prompt.
2. Up to three failed fingerprint scans.
3. Password prompt.
4. On wrong password, another fingerprint cycle starts.
5. After the next failed fingerprint cycle, password is requested again.

## Scenarios to support

1. Password requested immediately, password entered correctly: action should continue.
2. Password requested immediately, password entered incorrectly, password entered correctly: first attempt should fail or retry, second should continue.
3. Fingerprint requested first, scan succeeds: action should continue without requiring password.
4. Fingerprint requested first, scan fails, then password is entered correctly: action should continue.
5. Fingerprint requested first, scan fails repeatedly, no password is entered: action should fail/cancel cleanly without flooding the TUI.
6. Password entered incorrectly, then fingerprint succeeds: action should continue because PAM accepted fingerprint.
7. User cancels during auth: sudo process should be stopped and UI should return to the previous screen.
8. Sudo timestamp cache exists: privileged actions should use `sudo -k` so the UI does not silently skip authentication.
9. Local PAM ordering: three failed fingerprint attempts, password prompt, wrong password, three failed fingerprint attempts, password prompt.

## UI implications

1. Do not say "checking password" for the PAM result.
2. Say "sudo authentication" and mention both accepted paths: password and fingerprint.
3. Capture sudo stdout/stderr; do not let PAM prompts write directly over the TUI.
4. Show only the latest meaningful auth status, not every repeated PAM line.
5. Allow cancel while waiting for sudo/PAM.
6. Treat success source as unknown unless output unambiguously says password or fingerprint.

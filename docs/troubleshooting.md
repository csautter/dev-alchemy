# Troubleshooting

This page collects rare, host-specific issues that do not affect most Dev Alchemy setups.

## Windows: Cygwin Ansible Shadowed by a Host Installation

On Windows with Cygwin, the Ansible installation inside Cygwin can be shadowed by another Python or Ansible installation from the Windows host.

If that happens:

- Do not install Ansible directly on the Windows host.
- Remove conflicting host-level Ansible installations.
- Install and run Ansible through the Cygwin Python environment instead.

## macOS: Ansible Fork Safety Crash

Running Ansible on macOS can trigger a fork-safety crash similar to this:

```bash
TASK [Gathering Facts] ***************************************************************************************************************************************
objc[9473]: +[NSNumber initialize] may have been in progress in another thread when fork() was called.
objc[9473]: +[NSNumber initialize] may have been in progress in another thread when fork() was called. We cannot safely call it or ignore it in the fork() child process. Crashing instead. Set a breakpoint on objc_initializeAfterForkError to debug.
ERROR! A worker was found in a dead state
```

Work around it by setting this environment variable before running Ansible:

```bash
export OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES
```

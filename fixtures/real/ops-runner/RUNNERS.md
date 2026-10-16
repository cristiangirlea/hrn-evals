# Runners

This repository is private: its CI runs on our own runners, never on GitHub's
hosted ones.

- A Linux job runs on the runner the repository variable `CI_RUNNER` names, and
  only when it is set: `if: ${{ vars.CI_RUNNER != '' }}` and
  `runs-on: ${{ vars.CI_RUNNER }}`.
- A Windows job runs on the runner `CI_RUNNER_WINDOWS` names, only when it is
  set, and never for a pull request:
  `if: ${{ vars.CI_RUNNER_WINDOWS != '' && github.event_name != 'pull_request' }}`
  and `runs-on: ${{ vars.CI_RUNNER_WINDOWS }}`.
- Never `ubuntu-latest`, `windows-latest` or any other hosted runner, and no
  fallback to one when a variable is empty.

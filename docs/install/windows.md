---
last_updated: 2017-10-09
---

# Install GitLab Runner on Windows

To install and run GitLab Runner on Windows you need:

- Git installed.
- A password for your user account, if you want to run it under your user
  account rather than the Built-in System Account.

## Installation

CAUTION: **Important:**
With GitLab Runner 10, the executable was renamed to `gitlab-runner`. If you
want to install a version prior to GitLab Runner 10, [visit the old docs](old.md).

1. Create a folder somewhere in your system, ex.: `C:\GitLab-Runner`.
1. Download the binary for [x86][] or [amd64][] and put it into the folder you
   created. Rename the binary to `gitlab-runner.exe`.
   You can download a binary for every available version as described in
   [Bleeding Edge - download any other tagged
   release](bleeding-edge.md#download-any-other-tagged-release).
1. Run an [elevated command prompt](https://docs.microsoft.com/en-us/powershell/scripting/getting-started/starting-windows-powershell?view=powershell-6#with-administrative-privileges-run-as-administrator):
1. [Register the Runner](../register/index.md).
1. Install the Runner as a service and start it. You can either run the service
   using the Built-in System Account (recommended) or using a user account.

   **Run service using Built-in System Account** (under directory created in step 1. from above, ex.: `C:\GitLab-Runner`)

   ```powershell
   cd C:\GitLab-Runner
   ./gitlab-runner.exe install
   ./gitlab-runner.exe start
   ```

   **Run service using user account** (under directory created in step 1. from above, ex.: `C:\GitLab-Runner`)

   You have to enter a valid password for the current user account, because
   it's required to start the service by Windows:

   ```powershell
   cd C:\GitLab-Runner
   ./gitlab-runner.exe install --user ENTER-YOUR-USERNAME --password ENTER-YOUR-PASSWORD
   ./gitlab-runner.exe start
   ```

   See the [troubleshooting section](#troubleshooting) if you encounter any
   errors during the Runner installation.

1. (Optional) Update Runners `concurrent` value in `C:\GitLab-Runner\config.toml`
   to allow multiple concurrent jobs as detailed in [advanced configuration details](../configuration/advanced-configuration.md).
   Additionally you can use the advanced configuration details to update your
   shell executor to use Bash or PowerShell rather than Batch.

Voila! Runner is installed, running, and will start again after each system reboot.
Logs are stored in Windows Event Log.

## Update

1. Stop the service (you need an [elevated command prompt](https://docs.microsoft.com/en-us/powershell/scripting/getting-started/starting-windows-powershell?view=powershell-6#with-administrative-privileges-run-as-administrator) as before):

   ```powershell
   cd C:\GitLab-Runner
   ./gitlab-runner.exe stop
   ```

1. Download the binary for [x86][] or [amd64][] and replace runner's executable.
   You can download a binary for every available version as described in
   [Bleeding Edge - download any other tagged release](bleeding-edge.md#download-any-other-tagged-release).

1. Start the service:

   ```powershell
   ./gitlab-runner.exe start
   ```

## Uninstall

From an [elevated command prompt](https://docs.microsoft.com/en-us/powershell/scripting/getting-started/starting-windows-powershell?view=powershell-6#with-administrative-privileges-run-as-administrator):

```powershell
cd C:\GitLab-Runner
./gitlab-runner.exe stop
./gitlab-runner.exe uninstall
cd ..
rmdir /s GitLab-Runner
```

## Troubleshooting

Make sure that you read the [FAQ](../faq/README.md) section which describes
some of the most common problems with GitLab Runner.

If you encounter an error like _The account name is invalid_ try to add `.\` before the username:

```powershell
./gitlab-runner.exe install --user ".\ENTER-YOUR-USERNAME" --password "ENTER-YOUR-PASSWORD"
```

If you encounter a _The service did not start due to a logon failure_ error
while starting the service, please [look in the FAQ](../faq/README.md#the-service-did-not-start-due-to-a-logon-failure-error-when-starting-service) to check how to resolve the problem.

If you don't have a Windows Password, Runner's service won't start but you can
use the Built-in System Account.

If you have issues with the Built-in System Account, please read
[How to Configure the Service to Start Up with the Built-in System Account](https://support.microsoft.com/en-us/help/327545/how-to-troubleshoot-service-startup-permissions-in-windows-server-2003#6)
on Microsoft's support website.

[x86]: https://gitlab-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-runner-windows-386.exe
[amd64]: https://gitlab-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-runner-windows-amd64.exe

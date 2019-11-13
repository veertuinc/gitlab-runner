# Shells supported by GitLab Runner

GitLab Runner implements a few shell script generators that allow to execute
builds on different systems.

## Overview

The shell scripts contain commands to execute all steps of the build:

1. `git clone`
1. Restore the build cache
1. Build commands
1. Update the build cache
1. Generate and upload the build artifacts

The shells don't have any configuration options. The build steps are received
from the commands defined in the [`script` directive in `.gitlab-ci.yml`][script].

The currently supported shells are:

| Shell         | Status             |  Description |
| --------------| ------------------ |  ----------- |
| `bash`        | Fully Supported    | Bash (Bourne-shell) shell. All commands executed in Bash context (default for all Unix systems) |
| `sh`          | Fully Supported    | Sh (Bourne-shell) shell. All commands executed in Sh context (fallback for `bash` for all Unix systems) |
| `cmd`         | Deprecated         | Windows Batch script. All commands are executed in Batch context. Default when no [`shell`](../configuration/advanced-configuration.md#the-runners-section) is specified. Due for removal on Jun 22, 2020 in favor of PowerShell. |
| `powershell`  | Fully Supported    | Windows PowerShell script. All commands are executed in PowerShell context. Default when registering a new Runner in version 12.0 or newer. |

If you want to select a particular shell to use other than the default, you will need to [specify the shell](../executors/shell.md#selecting-your-shell) in your `config.toml` file.

## Sh/Bash shells

This is the default shell used on all Unix based systems. The bash script used
in `.gitlab-ci.yml` is executed by piping the shell script to one of the
following commands:

```bash
# This command is used if the build should be executed in context
# of another user (the shell executor)
cat generated-bash-script | su --shell /bin/bash --login user

# This command is used if the build should be executed using
# the current user, but in a login environment
cat generated-bash-script | /bin/bash --login

# This command is used if the build should be executed in
# a Docker environment
cat generated-bash-script | /bin/bash
```

## Windows Batch

NOTE: **Note:**
In GitLab 11.11, the Windows Batch executor for the
GitLab Runner was deprecated in favor of the [PowerShell](#powershell)
executor. Support for Windows Batch will be removed in GitLab 13.0 (Jun
22, 2020).

TIP: **Tip:**
You can execute batch scripts from PowerShell using `Start-Process
"cmd.exe" "/c C:\Path\file.bat"` for old batch scripts not ported to
PowerShell.

Windows Batch is the default shell used on Windows when
[`shell`](../configuration/advanced-configuration.md#the-runners-section) is not
specified.

It doesn't support executing the build in context of another user.

The generated Batch script is executed by saving its content to file and
passing the file name to the following command:

```bash
cmd /Q /C generated-windows-batch.cmd
```

This is how an example batch script looks like:

```bash
@echo off
setlocal enableextensions
setlocal enableDelayedExpansion
set nl=^


echo Running on %COMPUTERNAME%...

call :prescript
IF !errorlevel! NEQ 0 exit /b !errorlevel!

call :buildscript
IF !errorlevel! NEQ 0 exit /b !errorlevel!

call :postscript
IF !errorlevel! NEQ 0 exit /b !errorlevel!

goto :EOF
:prescript
SET CI=true
SET CI_COMMIT_SHA=db45ad9af9d7af5e61b829442fd893d96e31250c
SET CI_COMMIT_BEFORE_SHA=d63117656af6ff57d99e50cc270f854691f335ad
SET CI_COMMIT_REF_NAME=master
SET CI_JOB_ID=1
SET CI_REPOSITORY_URL=http://gitlab.example.com/group/project.git
SET CI_PROJECT_ID=1
SET CI_PROJECT_DIR=Z:\Gitlab\tests\test\builds\0\project-1
SET CI_SERVER=yes
SET CI_SERVER_NAME=GitLab CI
SET CI_SERVER_VERSION=
SET CI_SERVER_REVISION=
SET GITLAB_CI=true
md "C:\\GitLab-Runner\\builds\\0\\project-1.tmp" 2>NUL 1>NUL
echo multiline!nl!tls!nl!chain > C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO
SET GIT_SSL_CAINFO=C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO
md "C:\\GitLab-Runner\\builds\\0\\project-1.tmp" 2>NUL 1>NUL
echo multiline!nl!tls!nl!chain > C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE
SET CI_SERVER_TLS_CA_FILE=C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE
echo Cloning repository...
rd /s /q "C:\GitLab-Runner\builds\0\project-1" 2>NUL 1>NUL
"git" "clone" "http://gitlab.example.com/group/project.git" "Z:\Gitlab\tests\test\builds\0\project-1"
IF !errorlevel! NEQ 0 exit /b !errorlevel!

cd /D "C:\GitLab-Runner\builds\0\project-1"
IF !errorlevel! NEQ 0 exit /b !errorlevel!

echo Checking out db45ad9a as master...
"git" "checkout" "db45ad9af9d7af5e61b829442fd893d96e31250c"
IF !errorlevel! NEQ 0 exit /b !errorlevel!

IF EXIST "..\..\..\cache\project-1\pages\master\cache.tgz" (
  echo Restoring cache...
  "gitlab-runner-windows-amd64.exe" "extract" "--file" "..\..\..\cache\project-1\pages\master\cache.tgz"
  IF !errorlevel! NEQ 0 exit /b !errorlevel!

) ELSE (
  IF EXIST "..\..\..\cache\project-1\pages\master\cache.tgz" (
    echo Restoring cache...
    "gitlab-runner-windows-amd64.exe" "extract" "--file" "..\..\..\cache\project-1\pages\master\cache.tgz"
    IF !errorlevel! NEQ 0 exit /b !errorlevel!

  )
)
goto :EOF

:buildscript
SET CI=true
SET CI_COMMIT_SHA=db45ad9af9d7af5e61b829442fd893d96e31250c
SET CI_COMMIT_BEFORE_SHA=d63117656af6ff57d99e50cc270f854691f335ad
SET CI_COMMIT_REF_NAME=master
SET CI_JOB_ID=1
SET CI_REPOSITORY_URL=Z:\Gitlab\tests\test
SET CI_PROJECT_ID=1
SET CI_PROJECT_DIR=Z:\Gitlab\tests\test\builds\0\project-1
SET CI_SERVER=yes
SET CI_SERVER_NAME=GitLab CI
SET CI_SERVER_VERSION=
SET CI_SERVER_REVISION=
SET GITLAB_CI=true
md "C:\\GitLab-Runner\\builds\\0\\project-1.tmp" 2>NUL 1>NUL
echo multiline!nl!tls!nl!chain > C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO
SET GIT_SSL_CAINFO=C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO
md "C:\\GitLab-Runner\\builds\\0\\project-1.tmp" 2>NUL 1>NUL
echo multiline!nl!tls!nl!chain > C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE
SET CI_SERVER_TLS_CA_FILE=C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE
cd /D "C:\GitLab-Runner\builds\0\project-1"
IF !errorlevel! NEQ 0 exit /b !errorlevel!

echo $ echo true
echo true
goto :EOF

:postscript
SET CI=true
SET CI_COMMIT_SHA=db45ad9af9d7af5e61b829442fd893d96e31250c
SET CI_COMMIT_BEFORE_SHA=d63117656af6ff57d99e50cc270f854691f335ad
SET CI_COMMIT_REF_NAME=master
SET CI_JOB_ID=1
SET CI_REPOSITORY_URL=Z:\Gitlab\tests\test
SET CI_PROJECT_ID=1
SET CI_PROJECT_DIR=Z:\Gitlab\tests\test\builds\0\project-1
SET CI_SERVER=yes
SET CI_SERVER_NAME=GitLab CI
SET CI_SERVER_VERSION=
SET CI_SERVER_REVISION=
SET GITLAB_CI=true
md "C:\\GitLab-Runner\\builds\\0\\project-1.tmp" 2>NUL 1>NUL
echo multiline!nl!tls!nl!chain > C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO
SET GIT_SSL_CAINFO=C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO
md "C:\\GitLab-Runner\\builds\\0\\project-1.tmp" 2>NUL 1>NUL
echo multiline!nl!tls!nl!chain > C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE
SET CI_SERVER_TLS_CA_FILE=C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE
cd /D "C:\GitLab-Runner\builds\0\project-1"
IF !errorlevel! NEQ 0 exit /b !errorlevel!

echo Archiving cache...
"gitlab-runner-windows-amd64.exe" "archive" "--file" "..\..\..\cache\project-1\pages\master\cache.tgz" "--path" "vendor"
IF !errorlevel! NEQ 0 exit /b !errorlevel!

goto :EOF
```

## PowerShell

The default shell when a new Runner is registed using GitLab Runner
12.0 or newer.

PowerShell doesn't support executing the build in context of another user.

The generated PowerShell script is executed by saving it content to a file and
passing the file name to the following command:

```bash
powershell -noprofile -noninteractive -executionpolicy Bypass -command generated-windows-powershell.ps1
```

This is how an example PowerShell script looks like:

```bash
$ErrorActionPreference = "Continue"

echo "Running on $env:computername..."

& {
  $CI="true"
  $env:CI=$CI
  $CI_COMMIT_SHA="db45ad9af9d7af5e61b829442fd893d96e31250c"
  $env:CI_COMMIT_SHA=$CI_COMMIT_SHA
  $CI_COMMIT_BEFORE_SHA="d63117656af6ff57d99e50cc270f854691f335ad"
  $env:CI_COMMIT_BEFORE_SHA=$CI_COMMIT_BEFORE_SHA
  $CI_COMMIT_REF_NAME="master"
  $env:CI_COMMIT_REF_NAME=$CI_COMMIT_REF_NAME
  $CI_JOB_ID="1"
  $env:CI_JOB_ID=$CI_JOB_ID
  $CI_REPOSITORY_URL="Z:\Gitlab\tests\test"
  $env:CI_REPOSITORY_URL=$CI_REPOSITORY_URL
  $CI_PROJECT_ID="1"
  $env:CI_PROJECT_ID=$CI_PROJECT_ID
  $CI_PROJECT_DIR="Z:\Gitlab\tests\test\builds\0\project-1"
  $env:CI_PROJECT_DIR=$CI_PROJECT_DIR
  $CI_SERVER="yes"
  $env:CI_SERVER=$CI_SERVER
  $CI_SERVER_NAME="GitLab CI"
  $env:CI_SERVER_NAME=$CI_SERVER_NAME
  $CI_SERVER_VERSION=""
  $env:CI_SERVER_VERSION=$CI_SERVER_VERSION
  $CI_SERVER_REVISION=""
  $env:CI_SERVER_REVISION=$CI_SERVER_REVISION
  $GITLAB_CI="true"
  $env:GITLAB_CI=$GITLAB_CI
  $GIT_SSL_CAINFO=""
  md "C:\GitLab-Runner\builds\0\project-1.tmp" -Force | out-null
  $GIT_SSL_CAINFO | Out-File "C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO"
  $GIT_SSL_CAINFO="C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO"
  $env:GIT_SSL_CAINFO=$GIT_SSL_CAINFO
  $CI_SERVER_TLS_CA_FILE=""
  md "C:\GitLab-Runner\builds\0\project-1.tmp" -Force | out-null
  $CI_SERVER_TLS_CA_FILE | Out-File "C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE"
  $CI_SERVER_TLS_CA_FILE="C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE"
  $env:CI_SERVER_TLS_CA_FILE=$CI_SERVER_TLS_CA_FILE
  echo "Cloning repository..."
  if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path "C:\GitLab-Runner\builds\0\project-1" -PathType Container) ) {
    Remove-Item2 -Force -Recurse "C:\GitLab-Runner\builds\0\project-1"
  } elseif(Test-Path "C:\GitLab-Runner\builds\0\project-1") {
    Remove-Item -Force -Recurse "C:\GitLab-Runner\builds\0\project-1"
  }

  & "git" "clone" "https://gitlab.com/group/project.git" "Z:\Gitlab\tests\test\builds\0\project-1"
  if(!$?) { Exit $LASTEXITCODE }

  cd "C:\GitLab-Runner\builds\0\project-1"
  if(!$?) { Exit $LASTEXITCODE }

  echo "Checking out db45ad9a as master..."
  & "git" "checkout" "db45ad9af9d7af5e61b829442fd893d96e31250c"
  if(!$?) { Exit $LASTEXITCODE }

  if(Test-Path "..\..\..\cache\project-1\pages\master\cache.tgz" -PathType Leaf) {
    echo "Restoring cache..."
    & "gitlab-runner-windows-amd64.exe" "extract" "--file" "..\..\..\cache\project-1\pages\master\cache.tgz"
    if(!$?) { Exit $LASTEXITCODE }

  } else {
    if(Test-Path "..\..\..\cache\project-1\pages\master\cache.tgz" -PathType Leaf) {
      echo "Restoring cache..."
      & "gitlab-runner-windows-amd64.exe" "extract" "--file" "..\..\..\cache\project-1\pages\master\cache.tgz"
      if(!$?) { Exit $LASTEXITCODE }

    }
  }
}
if(!$?) { Exit $LASTEXITCODE }

& {
  $CI="true"
  $env:CI=$CI
  $CI_COMMIT_SHA="db45ad9af9d7af5e61b829442fd893d96e31250c"
  $env:CI_COMMIT_SHA=$CI_COMMIT_SHA
  $CI_COMMIT_BEFORE_SHA="d63117656af6ff57d99e50cc270f854691f335ad"
  $env:CI_COMMIT_BEFORE_SHA=$CI_COMMIT_BEFORE_SHA
  $CI_COMMIT_REF_NAME="master"
  $env:CI_COMMIT_REF_NAME=$CI_COMMIT_REF_NAME
  $CI_JOB_ID="1"
  $env:CI_JOB_ID=$CI_JOB_ID
  $CI_REPOSITORY_URL="Z:\Gitlab\tests\test"
  $env:CI_REPOSITORY_URL=$CI_REPOSITORY_URL
  $CI_PROJECT_ID="1"
  $env:CI_PROJECT_ID=$CI_PROJECT_ID
  $CI_PROJECT_DIR="Z:\Gitlab\tests\test\builds\0\project-1"
  $env:CI_PROJECT_DIR=$CI_PROJECT_DIR
  $CI_SERVER="yes"
  $env:CI_SERVER=$CI_SERVER
  $CI_SERVER_NAME="GitLab CI"
  $env:CI_SERVER_NAME=$CI_SERVER_NAME
  $CI_SERVER_VERSION=""
  $env:CI_SERVER_VERSION=$CI_SERVER_VERSION
  $CI_SERVER_REVISION=""
  $env:CI_SERVER_REVISION=$CI_SERVER_REVISION
  $GITLAB_CI="true"
  $env:GITLAB_CI=$GITLAB_CI
  $GIT_SSL_CAINFO=""
  md "C:\GitLab-Runner\builds\0\project-1.tmp" -Force | out-null
  $GIT_SSL_CAINFO | Out-File "C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO"
  $GIT_SSL_CAINFO="C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO"
  $env:GIT_SSL_CAINFO=$GIT_SSL_CAINFO
  $CI_SERVER_TLS_CA_FILE=""
  md "C:\GitLab-Runner\builds\0\project-1.tmp" -Force | out-null
  $CI_SERVER_TLS_CA_FILE | Out-File "C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE"
  $CI_SERVER_TLS_CA_FILE="C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE"
  $env:CI_SERVER_TLS_CA_FILE=$CI_SERVER_TLS_CA_FILE
  cd "C:\GitLab-Runner\builds\0\project-1"
  if(!$?) { Exit $LASTEXITCODE }

  echo "`$ echo true"
  echo true
}
if(!$?) { Exit $LASTEXITCODE }

& {
  $CI="true"
  $env:CI=$CI
  $CI_COMMIT_SHA="db45ad9af9d7af5e61b829442fd893d96e31250c"
  $env:CI_COMMIT_SHA=$CI_COMMIT_SHA
  $CI_COMMIT_BEFORE_SHA="d63117656af6ff57d99e50cc270f854691f335ad"
  $env:CI_COMMIT_BEFORE_SHA=$CI_COMMIT_BEFORE_SHA
  $CI_COMMIT_REF_NAME="master"
  $env:CI_COMMIT_REF_NAME=$CI_COMMIT_REF_NAME
  $CI_JOB_ID="1"
  $env:CI_JOB_ID=$CI_JOB_ID
  $CI_REPOSITORY_URL="Z:\Gitlab\tests\test"
  $env:CI_REPOSITORY_URL=$CI_REPOSITORY_URL
  $CI_PROJECT_ID="1"
  $env:CI_PROJECT_ID=$CI_PROJECT_ID
  $CI_PROJECT_DIR="Z:\Gitlab\tests\test\builds\0\project-1"
  $env:CI_PROJECT_DIR=$CI_PROJECT_DIR
  $CI_SERVER="yes"
  $env:CI_SERVER=$CI_SERVER
  $CI_SERVER_NAME="GitLab CI"
  $env:CI_SERVER_NAME=$CI_SERVER_NAME
  $CI_SERVER_VERSION=""
  $env:CI_SERVER_VERSION=$CI_SERVER_VERSION
  $CI_SERVER_REVISION=""
  $env:CI_SERVER_REVISION=$CI_SERVER_REVISION
  $GITLAB_CI="true"
  $env:GITLAB_CI=$GITLAB_CI
  $GIT_SSL_CAINFO=""
  md "C:\GitLab-Runner\builds\0\project-1.tmp" -Force | out-null
  $GIT_SSL_CAINFO | Out-File "C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO"
  $GIT_SSL_CAINFO="C:\GitLab-Runner\builds\0\project-1.tmp\GIT_SSL_CAINFO"
  $env:GIT_SSL_CAINFO=$GIT_SSL_CAINFO
  $CI_SERVER_TLS_CA_FILE=""
  md "C:\GitLab-Runner\builds\0\project-1.tmp" -Force | out-null
  $CI_SERVER_TLS_CA_FILE | Out-File "C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE"
  $CI_SERVER_TLS_CA_FILE="C:\GitLab-Runner\builds\0\project-1.tmp\CI_SERVER_TLS_CA_FILE"
  $env:CI_SERVER_TLS_CA_FILE=$CI_SERVER_TLS_CA_FILE
  cd "C:\GitLab-Runner\builds\0\project-1"
  if(!$?) { Exit $LASTEXITCODE }

  echo "Archiving cache..."
  & "gitlab-runner-windows-amd64.exe" "archive" "--file" "..\..\..\cache\project-1\pages\master\cache.tgz" "--path" "vendor"
  if(!$?) { Exit $LASTEXITCODE }

}
if(!$?) { Exit $LASTEXITCODE }
```

[script]: http://doc.gitlab.com/ce/ci/yaml/README.html#script

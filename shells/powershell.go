package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
)

const dockerWindowsExecutor = "docker-windows"

type PowerShell struct {
	AbstractShell
}

type PsWriter struct {
	bytes.Buffer
	TemporaryPath string
	indent        int
}

func psQuote(text string) string {
	// taken from: http://www.robvanderwoude.com/escapechars.php
	text = strings.Replace(text, "`", "``", -1)
	// text = strings.Replace(text, "\0", "`0", -1)
	text = strings.Replace(text, "\a", "`a", -1)
	text = strings.Replace(text, "\b", "`b", -1)
	text = strings.Replace(text, "\f", "^f", -1)
	text = strings.Replace(text, "\r", "`r", -1)
	text = strings.Replace(text, "\n", "`n", -1)
	text = strings.Replace(text, "\t", "^t", -1)
	text = strings.Replace(text, "\v", "^v", -1)
	text = strings.Replace(text, "#", "`#", -1)
	text = strings.Replace(text, "'", "`'", -1)
	text = strings.Replace(text, "\"", "`\"", -1)
	return "\"" + text + "\""
}

func psQuoteVariable(text string) string {
	text = psQuote(text)
	text = strings.Replace(text, "$", "`$", -1)
	return text
}

func (b *PsWriter) GetTemporaryPath() string {
	return b.TemporaryPath
}

func (b *PsWriter) Line(text string) {
	b.WriteString(strings.Repeat("  ", b.indent) + text + "\r\n")
}

func (b *PsWriter) CheckForErrors() {
	b.checkErrorLevel()
}

func (b *PsWriter) Indent() {
	b.indent++
}

func (b *PsWriter) Unindent() {
	b.indent--
}

func (b *PsWriter) checkErrorLevel() {
	b.Line("if(!$?) { Exit &{if($LASTEXITCODE) {$LASTEXITCODE} else {1}} }")
	b.Line("")
}

func (b *PsWriter) Command(command string, arguments ...string) {
	b.Line(b.buildCommand(command, arguments...))
	b.checkErrorLevel()
}

func (b *PsWriter) buildCommand(command string, arguments ...string) string {
	list := []string{
		psQuote(command),
	}

	for _, argument := range arguments {
		list = append(list, psQuote(argument))
	}

	return "& " + strings.Join(list, " ")
}

func (b *PsWriter) TmpFile(name string) string {
	filePath := b.Absolute(path.Join(b.TemporaryPath, name))
	return helpers.ToBackslash(filePath)
}

func (b *PsWriter) EnvVariableKey(name string) string {
	return fmt.Sprintf("$%s", name)
}

func (b *PsWriter) Variable(variable common.JobVariable) {
	if variable.File {
		variableFile := b.TmpFile(variable.Key)
		b.Line(fmt.Sprintf("New-Item -ItemType directory -Force -Path %s | out-null", psQuote(helpers.ToBackslash(b.TemporaryPath))))
		b.Line(fmt.Sprintf("Set-Content %s -Value %s -Encoding UTF8 -Force", psQuote(variableFile), psQuoteVariable(variable.Value)))
		b.Line("$" + variable.Key + "=" + psQuote(variableFile))
	} else {
		b.Line("$" + variable.Key + "=" + psQuoteVariable(variable.Value))
	}

	b.Line("$env:" + variable.Key + "=$" + variable.Key)
}

func (b *PsWriter) IfDirectory(path string) {
	b.Line("if(Test-Path " + psQuote(helpers.ToBackslash(path)) + " -PathType Container) {")
	b.Indent()
}

func (b *PsWriter) IfFile(path string) {
	b.Line("if(Test-Path " + psQuote(helpers.ToBackslash(path)) + " -PathType Leaf) {")
	b.Indent()
}

func (b *PsWriter) IfCmd(cmd string, arguments ...string) {
	b.ifInTryCatch(b.buildCommand(cmd, arguments...) + " 2>$null")
}

func (b *PsWriter) IfCmdWithOutput(cmd string, arguments ...string) {
	b.ifInTryCatch(b.buildCommand(cmd, arguments...))
}

func (b *PsWriter) ifInTryCatch(cmd string) {
	b.Line("Set-Variable -Name cmdErr -Value $false")
	b.Line("Try {")
	b.Indent()
	b.Line(cmd)
	b.Line("if(!$?) { throw &{if($LASTEXITCODE) {$LASTEXITCODE} else {1}} }")
	b.Unindent()
	b.Line("} Catch {")
	b.Indent()
	b.Line("Set-Variable -Name cmdErr -Value $true")
	b.Unindent()
	b.Line("}")
	b.Line("if(!$cmdErr) {")
	b.Indent()
}

func (b *PsWriter) Else() {
	b.Unindent()
	b.Line("} else {")
	b.Indent()
}

func (b *PsWriter) EndIf() {
	b.Unindent()
	b.Line("}")
}

func (b *PsWriter) Cd(path string) {
	b.Line("cd " + psQuote(helpers.ToBackslash(path)))
	b.checkErrorLevel()
}

func (b *PsWriter) MkDir(path string) {
	b.Line(fmt.Sprintf("New-Item -ItemType directory -Force -Path %s | out-null", psQuote(helpers.ToBackslash(path))))
}

func (b *PsWriter) MkTmpDir(name string) string {
	path := helpers.ToBackslash(path.Join(b.TemporaryPath, name))
	b.MkDir(path)

	return path
}

func (b *PsWriter) RmDir(path string) {
	path = psQuote(helpers.ToBackslash(path))
	b.Line("if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path " + path + " -PathType Container) ) {")
	b.Indent()
	b.Line("Remove-Item2 -Force -Recurse " + path)
	b.Unindent()
	b.Line("} elseif(Test-Path " + path + ") {")
	b.Indent()
	b.Line("Remove-Item -Force -Recurse " + path)
	b.Unindent()
	b.Line("}")
	b.Line("")
}

func (b *PsWriter) RmFile(path string) {
	path = psQuote(helpers.ToBackslash(path))
	b.Line("if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path " + path + " -PathType Leaf) ) {")
	b.Indent()
	b.Line("Remove-Item2 -Force " + path)
	b.Unindent()
	b.Line("} elseif(Test-Path " + path + ") {")
	b.Indent()
	b.Line("Remove-Item -Force " + path)
	b.Unindent()
	b.Line("}")
	b.Line("")
}

func (b *PsWriter) Print(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_RESET + fmt.Sprintf(format, arguments...)
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) Notice(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_BOLD_GREEN + fmt.Sprintf(format, arguments...) + helpers.ANSI_RESET
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) Warning(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_YELLOW + fmt.Sprintf(format, arguments...) + helpers.ANSI_RESET
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) Error(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_BOLD_RED + fmt.Sprintf(format, arguments...) + helpers.ANSI_RESET
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) EmptyLine() {
	b.Line("echo \"\"")
}

func (b *PsWriter) Absolute(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}

	b.Line("$CurrentDirectory = (Resolve-Path .\\).Path")
	return filepath.Join("$CurrentDirectory", dir)
}

func (b *PsWriter) Finish(trace bool) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	// write BOM
	io.WriteString(w, "\xef\xbb\xbf")
	if trace {
		io.WriteString(w, "Set-PSDebug -Trace 2\r\n")
	}

	// add empty line to close code-block when it is piped to STDIN
	b.Line("")
	io.WriteString(w, b.String())
	w.Flush()
	return buffer.String()
}

func (b *PowerShell) GetName() string {
	return "powershell"
}

func (b *PowerShell) GetConfiguration(info common.ShellScriptInfo) (script *common.ShellConfiguration, err error) {
	script = &common.ShellConfiguration{
		Command:       "powershell",
		Arguments:     []string{"-noprofile", "-noninteractive", "-executionpolicy", "Bypass", "-command"},
		PassFile:      info.Build.Runner.Executor != dockerWindowsExecutor,
		Extension:     "ps1",
		DockerCommand: []string{"PowerShell", "-NoProfile", "-NoLogo", "-InputFormat", "text", "-OutputFormat", "text", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "-"},
	}
	return
}

func (b *PowerShell) GenerateScript(buildStage common.BuildStage, info common.ShellScriptInfo) (script string, err error) {
	w := &PsWriter{
		TemporaryPath: info.Build.TmpProjectDir(),
	}

	if buildStage == common.BuildStagePrepare {
		if len(info.Build.Hostname) != 0 {
			w.Line("echo \"Running on $env:computername via " + psQuoteVariable(info.Build.Hostname) + "...\"")
		} else {
			w.Line("echo \"Running on $env:computername...\"")
		}
	}

	err = b.writeScript(w, buildStage, info)

	// No need to set up BOM or tracing since no script was generated.
	if w.Buffer.Len() > 0 {
		script = w.Finish(info.Build.IsDebugTraceEnabled())
	}

	return
}

func (b *PowerShell) IsDefault() bool {
	return false
}

func init() {
	common.RegisterShell(&PowerShell{})
}

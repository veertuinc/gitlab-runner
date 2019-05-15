v11.10.0 (2019-04-22)

**Deprecations:**

All deprecations, with a detailed description, are listed at
https://about.gitlab.com/2019/04/22/gitlab-11-10-released/#release-deprecations

1. With version 11.10 we're deprecating the feature flag
[FF_USE_LEGACY_GIT_CLEAN_STRATEGY](https://docs.gitlab.com/runner/configuration/feature-flags.html#available-feature-flags).

**Release changes:**

- Fix git lfs not getting submodule objects !1298
- Refactor slightly ./shells/shellstest !1237
- Fix CI_PROJECT_DIR handling !1241
- Log time took preparing executors !1196
- Restore availability of pprof in the debug server !1242
- Move variables defining .gopath to a shared place for all Windows jobs !1245
- Docs: clarify runner api registration process !1244
- add lfs support to ubuntu docker runner !1192
- Add information about Kaniko for Kubernetes executor !1161
- Enable the docs CI job !1251
- Rename test to be more descriptive !1249
- Create the reviewers guide base document !1233
- Update codeclimate version !1252
- Add retryable err type !1215
- Get windows tag for helper image !1239
- Remove unnecessary log alias for logrus inport !1256
- Make gitlab-runner:alpine more specific, Add link to Dockerfiles sources,... !1259
- Docs: Fix broken anchor in docker.md !1264
- Replace the current k8s manual installation with the Helm chart !1250
- Create cache for `/builds` dir !1265
- Expose `CI_CONCURRENT_(PROJECT)_ID` !1268
- DOC: note on case-sensitive proxy variables and the need for upper and lower case versions !1248
- Add new links checker !1271
- Update log messages for listen & session address !1275
- Use delayed variable expansion for error check in cmd !1260
- Unexport common.RepoRemoteURL !1276
- Update index.md - added sudo when registering the service on macos (without... !1272
- Add new lines around lists for renderer !1278
- Fix color output on Windows !1208
- Make it again possible to disable Git LFS pull !1273
- Add cross references to Runners API !1284
- Improve support for `git clean` !1281
- Make Kubernetes executor to clone into /builds !1282
- Add option to specify clone path !1267
- Allow to disable debug tracing !1286
- Add Route Map for runner docs !1285
- Do not print remote addition failure message !1287
- Add true to the run-untagged subcommand !1288
- Cleanup k8s cleanup test !1280
- Change helper image to servercore !1290
- Add note about git-lfs !1294

v11.9.0 (2019-03-22)

**Deprecations:**

All deprecations, with a detailed description, are listed at
https://about.gitlab.com/2019/03/22/gitlab-11-9-released/#release-deprecations

1. With version 11.9 we're deprecating the support for Docker Executor on CentOS 6

2. With version 11.9 we've implemented a new method for cloning/fetching repositories.
   Currently GitLab Runner still respects the old configuration sent from GitLab, but with
   12.0 old methods will be removed and GitLab Runner will require at least GitLab 11.9
   to work properly.

3. With version 11.0 we've changed how the metrics server is configured for GitLab Runner.
   `metrics_server` was replaced with `listen_address`. With version 12.0 the old configuration
   option will be removed.

4. With version 11.3 we've implemented support for different remote cache providers, which
   required a change in how the cache is configured. With version 12.0 support for old
   configuration structure will be removed.

5. With version 11.4 we've fixed the way how `entrypoint:` and `command:` options of
   Extended Docker configuration (https://docs.gitlab.com/ee/ci/docker/using_docker_images.html#extended-docker-configuration-options)
   are being handled by Kubernetes Executor. The previous implementation was wrong and
   was making the configuration unusable in most cases. However some users could relay
   on this wrong behavior. Because of that we've added a feature flag `FF_K8S_USE_ENTRYPOINT_OVER_COMMAND`
   which, when set to `false`, could bring back the old behavior. With version 12.0 the
   feature flag as well as the old behavior will be removed.

6. Some Linux distributions for which GitLab Runner is providing DEB and RPM packages
   have reached their End of Life. With version 12.0 we'll remove support for all
   EoL distributions at the moment of 12.0 release.

7. With version 11.9 we've prepared a go-based replacement for Runner Helper commands
   executed within Docker executor inside of the Helper Image. With version 12.0
   we will remove support for old commands basing on bash scripts. This change will
   affect only the users that are configuring their custom Helper Image (the image
   will require an update to align with new requirements)

**Release changes:**

- fix(parallels): use the newer sntp command to time sync !1145
- Update docker API verion !1187
- Update alpine images to alpine 3.9 !1197
- Fix a typo in the description of the configuration option !1205
- Document creation of Docker volumes passed with docker exec --docker-volumes !1120
- Correct spelling of timed out in literals !1121
- Fix spelling and other minor improvements !1207
- Migrate service wait script to Go !1195
- Docs update: Run runner on kubernetes !1185
- Increase test timeout for shell executor !1214
- Follow style convention for documentation !1213
- Add test for runner build limit !1186
- Migrate cache bash script to Go for helper image !1201
- Document OS deprecations for 12.0 !1210
- Fix anchors in Runner documentation !1216
- Add `build_simple` to `help` make target !1212
- Split `make docker` for GitLab Runner Helper !1188
- Add windows Dockerfiles for gitlab-runner-helper !1167
- Make Runner tests working on Windows with our CI Pipeline !1219
- Fetch code from provided refspecs !1203
- Check either ntpdate command exists or not before trying to execute it !1189
- Deprecate helper image commands !1218
- Add script for building windows helper image !1178
- Fix ShellWriter.RmFile(string) for cmd shell !1226
- Mask log trace !1204
- Add note about pod annotations for more clarity !1220
- Resolve memory allocation failure when cloning repos with LFS objects bigger than available RAM !1200
- Release also on gitlab releases page !1232
- Restore availability of pprof in the debug server !1242

v11.8.0 (2019-02-22)
- Kubernetes executor: add support for Node tolerations !941
- Update logrus version to v1.3.0 !1137
- Docs - Clarify Docker Runner Documentation !1097
- Update github.com/stretchr/testify dependency !1141
- Update LICENSE file !1132
- Update example of cache config !1140
- Update documentation for autoscaling on AWS !1142
- Remove unnecessary dep constraint !1147
- readme: make author block render md !999
- Corrected note when using a config container to mount custom data volume. !1126
- Fix typo in documentation of k8s executor. !1118
- Make new runner tokens compatible with docker-machine executor !1144
- docs: Use `sudo tee` for apt pinning. !1047
- docs: fix indendation !1081
- Updated hint on running Windows 10 shell as administrator !1136
- Fixed typo in logged information !1074
- Update registry_and_cache_servers.md !1098
- Update golang.org/x/sys !1149
- Refactor frontpage for grammar and style !1151
- Update github.com/Azure/go-ansiterm dependency !1152
- Testing on windows with vagrant !1003
- Add fix for race condition in windows cache extraction !863
- Consolidate docker API version definition !1154
- Prevent Executors from modifying Runner configuration !1134
- Update ExecutorProvider interface signature !1159
- Update logging for processing multi runner !1160
- Update kubernetes.md - fix typo for bearer_token !1162
- Update github.com/prometheus/client_golang dep !1150
- Remove ContainerWait from docker client !1155
- Update advanced-configuration.md: Fix blockquote not reaching the entire note !1163
- Fix docs review app URL !1169
- docs: Add a helpful command to reload config !1106
- Update AWS autoscale documentation !1166
- Refactor dockerfiles !1068
- Add link to AWS driver about default values !1171
- Add support for fedora/29 packages !1082
- Add windows server 2019 as default for windows development !1165
- Docs: Fix bad anchor links in runner docs !1177
- Improve documentation concerning proxy setting in the case of docker-in-docker-executor !1090
- Add few fixes to Release Checklist template !1135
- Set table to not display under TOC !1168
- Update Docker client SDK !1148
- docs: add GitLab Runner Helm Chart link !945

v11.7.0 (2019-01-22)
- Docs: Cleaning up the executors doc !1114
- Update to testify v1.2.2 !1119
- Fix a typo in VirtualBox Executor docs !1124
- Use the term `macOS` instead of `OS X` or `OSX` !1125
- Update github.com/sirupsen/logrus dependency !1129
- Docs update release checklist !1131
- Kill session when build is cancelled !1058
- Fix path separator for CI_PROJECT_DIR in Windows !1128
- Make new runner tokens compatible with docker-machine executor !1144

v11.6.0 (2018-12-22)
- Make compatibility chart super clear and remove old entries !1078
- Add slack notification option for 'dep status' check failures !1072
- Docker executor: use DNS, DNSSearch and ExtraHosts settings from configuration !1075
- Fix some invalid links in documentation !1085
- Fix SC2155 where shellcheck warns about errors !1063
- Change parallel tests configuration ENV names !1095
- Improve documentation of IAM instance profile usage for caching !1071
- Remove duplicate builds_dir definition from docs !952
- Make k8s object names DNS-1123 compatible !1105
- Docs: working example of helper image with CI_RUNNER_REVISION !1032
- Docs: omit ImagePullPolicy !1107
- Disable the docs lint job for now !1112
- Docs: comment about how listen_address works !1076
- Fix the indented bullet points of the features list in documentation !1093
- Add note on the branch naming for documentation changes !1113
- Docs: add session-server link to advanced list in index !1108

v11.5.0 (2018-11-22)
- Support RAW artifacts !1057
- Docs: changing secret variable to just variable in advanced-configuration.md !1055
- Docs: Fixing some bad links in Runner docs. !1056
- Docs: Updating Docs links from /ce to /ee !1061
- Docs: Fixing Substrakt Health URL !1064
- Add failure reason for execution timeout !1051

v 11.4.0 (2018-10-22)
- Do not create apk cache !1017
- Handle untracked files with Unicode characters in filenames. !913
- Add metrics with concurrent and limit values !1019
- Add a gitlab_runner_jobs_total metric !1018
- Add a job duration histogram metric !1025
- Filter content of X-Amz-Credential from logs !1028
- Disable escaping project bucket in cache operations !1029
- Fix example for session_server and added the note about where this section should be placed !1035
- Fix job duration counting !1033
- Log duration on job finishing log line !1034
- Allow disabling docker entrypoint overwrite !965
- Fix command and args assignment when creating containers with K8S executor !1010
- Support json logging !1020
- Change image for docs link checking !1043
- Fix command that prepares the definitions of tests !1044
- Add OomKillDisable option to Docker executor !1042
- Add docker support for interactive web terminal !1008
- Add support docker machine web terminal support !1046

v 11.3.0 (2018-09-22)
- Fix logrus secrets cleanup !990
- Fix test failure detection !993
- Fix wrongly generated `Content-Range` header for `PATCH /api/v4/jobs/:id/trace` request !906
- Improve and fix release checklist !940
- Add ~"git operations" label to CONTRIBUTING guide !943
- Disable few jobs for docs-/-docs branches !996
- Update release checklist issue template !995
- Fix HTTPS validation problem when ssh executor is used !962
- Reduce complexity of reported methods !997
- Update docker images to alpine:3.8 !984
- Fail build in case of code_quality errors !986
- Add initial support for CI Web Terminal !934
- Make session and metrics server initialization logging consistent !994
- Make prepare-changelog-entries.rb script compatible with GitLab APIv4 !927
- Save compilation time always in UTC timezone !1000
- Extend debug logging for k8s executor !949
- Introduce GCS adapter for remote cache !968
- Make configuration of helper image more dynamic !1005
- Logrus upgrade - fix data race in helpers.MakeFatalToPanic() !1011
- Add few TODOs to mark things that should be cleaned in 12.0 !1013
- Update debug jobs list output !992
- Remove duplicate build_dir setting !1015
- Add step for updating Runner helm chart !1009
- Clenup env, cli-options and deprecations of cache settings !1012

v 11.2.0 (2018-08-22)
- Fix support for Unicode variable values when Windows+PowerShell are used !960
- Update docs/executors/kubernetes.md !957
- Fix missing code_quality widget !972
- Add `artifact` format !923
- Improve some k8s executor tests !980
- Set useragent in Kubernetes API calls !977
- Clarifying the tls-ca-file option is in the [[runners]] section !973
- Update mocks !983
- Add building to development heading !919
- Add coverage report for unit tests !928
- Add /etc/nsswitch.conf to helper on docker executor to read /etc/hosts when upload artifacts !951
- Add busybox shell !900
- Fix support for features for shells !989
- Fix logrus secrets cleanup !990
- Fix test failure detection !993

v 11.1.0 (2018-07-22)
- Fix support for Unicode variable values when Windows+PowerShell are used !960
- Unify receivers used for 'executor' struct in ./executors/docker/ !926
- Update Release Checklist template !898
- Cache the connectivity of live Docker Machine instances !909
- Update kubernetes vendor to 1.10 !877
- Upgrade helper image alpine 3.7 !917
- Detect possible misplaced boolean on command line !932
- Log 'metrics_server' deprecation not only when the setting is used !939
- Speed-up ./executor/docker/executor_docker_command_test.go tests !937
- Remove go-bindata !831
- Fix the release of helper images script !946
- Sign RPM and DEB packages !922
- Improve docker timeouts !963
- Wrap all docker errors !964

v 11.0.0 (2018-06-22)
- Resolve "Invalid OffPeakPeriods value, no such file or directory." !897
- Add --paused option to register command !896
- Start rename of "metrics server" config !838
- Update virtualbox.md temporary fix for #2981 !889
- Fix panic on PatchTrace execution !905
- Do not send first PUT !908
- Rename CI_COMMIT_REF to CI_COMMIT_SHA !911
- Fix test file archiver tests !915
- Document how check_interval works !903
- Add link to development guide in readme !918
- Explain gitlab-runner workflow labels !921
- Change Prometheus metrics names !912

v 10.8.0 (2018-05-22)
- Resolve "Invalid OffPeakPeriods value, no such file or directory." !897
- Fix type in Substrakt Health company name !875
- Rename libre to core !879
- Correct hanging parenthesis in index.md !882
- Update interfaces mocks !871
- Rename keyword in kubernetes executor documentation !880
- Temporary add 'retry: 2' for 'unit tests (no race)' job !885
- Update docs/executors/README.md !881
- Add support for fedora/27 and fedora/28 packages !883
- Update supported distribution releases !887
- Automatize release checklist issue creation !870
- Change docs license to CC BY-SA 4.0 !893
- Update Docker installation method docs !890
- Add new metrics related to jobs requesting and API usage !886

v 10.7.0 (2018-04-22)
- Rename Sirupsen/logrus library !843
- Refer to gitlab versions as libre, starter, premium, and ultimate !851
- Fix assert.Equal parameter order !854
- Upgrade docker-machine to v0.14.0 !850
- Refactor autoscale docs !733
- Add possibility to specify memory in Docker containers !847
- Upgrade helper image to alpine 3.6 !859
- Update docker images bases to alpine:3.7 and ubuntu:16:04 !860
- Verify git-lfs checksum !796
- Improve services health check !867
- Add proxy documentation !623
- Downgrade go to 1.8.7 !869
- Add support for max_job_timeout parameter in registration !846

v 10.6.0 (2018-03-22)
- Upgrade docker-machine to v0.14.0 !850
- Upgrade helper image to alpine 3.6 !859
- Add CI_RUNNER_VERSION, CI_RUNNER_REVISION, and CI_RUNNER_EXECUTABLE_ARCH job environment variables !788
- Always prefer creating new containers when running with Docker Executor !818
- Use IAM instance profile credentials for S3 caching !646
- exec command is no longer deprecated !834
- Print a notice when skipping cache operation due to empty cache key !842
- Switch to Go 1.9.4 !827
- Move dependencies to dep !813
- Improve output of /debug/jobs/list !826
- Fix panic running docker package tests !828
- Fixed typo in console output !845

v 10.5.0 (2018-02-22)
- Always prefer creating new containers when running with Docker Executor !818
- Improve output of /debug/jobs/list !826
- Fix panic running docker package tests !828
- Fix git 1.7.1 compatibility in executors/shell package tests !791
- Do not add /cache volume if already provided by the user during gitlab-runner register !807
- Change confusing Built value for development version !821
- docs: explain valid values for check_interval !801
- docs: Fix OffPeak variables list !806
- docs: Add note about gitlab-runner on the SSH host being used for uploads !817

v 10.4.0 (2018-01-22)
- Always load OS certificate pool when evaluating TLS connections !804
- Add (overwritable) pod annotations for the kubernetes executor !666
- docker.allowed_images can use glob syntax in config.toml !721
- Added docker runtime support !764
- Send `failure_reason` when updating job statues (GitLab API endpoint) !675
- Do not use `git config --local` as it's not available in git v1.7.1 !790
- Use local GOPATH in Makefile !779
- Move Bleeding Edge release from ubuntu/yakkety to ububut/artful !797
- Fix data race in commands package unit tests !787
- Fix data race in function common.(*Trace).Write() !784
- Fix data races in executor/docker package !800
- Fix data races in network package !775

v 10.3.1 (2018-01-22)
- Always load OS certificate pool when evaluating TLS connections !804

v 10.3.0 (2017-12-22)
- Do not use `git config --local` as it's not available in git v1.7.1 !790
- new RC naming schema !780
- Stop Docker Machine before removing it !718
- add `--checkout --force` options to `git submodule update --init` !704
- Fix trailing "<nil>" in syslog logging !734
- Fix kubernetes executor job overwritten variables behavior !739
- Add zip archive for windows release files !760
- Add kubernetes executor connection with service account, bearer token can also be overwritten !744
- Fix SIGSEGV in kubernetes executor Cleanup !769

v 10.2.1 (2018-01-22)
- Do not use `git config --local` as it's not available in git v1.7.1 !790
- Always load OS certificate pool when evaluating TLS connections !804

v 10.2.0 (2017-11-22)
- Update supported platforms !712
- Fix typo in Kubernetes runner docs !714
- Add info on upgrading to Runner 10 !709
- Add some documentation for disable_cache configuration option !713
- Remove .git/HEAD.lock before git fetch !722
- Add helper_image option to docker executor config !723
- Add notes about gitlab-runner inside the VM being used for uploads !719
- Fix panic when global flags are passed as command flags !726
- Update minio go library to v3.0.3 !707
- Label ci_runner_builds metric with runner short token !729

v 10.1.1 (2018-01-22)
- Do not use `git config --local` as it's not available in git v1.7.1 !790
- Always load OS certificate pool when evaluating TLS connections !804

v 10.1.0 (2017-10-22)
- Allow customizing go test flags with TESTFLAGS variable !688
- Clarify that cloning a runner could be considered an attack vector !658
- Remove disable_verbose from docs !692
- Add info about pre 10.0 releases !691
- Update BurntSushi/toml for MIT-license !695
- Expose if running in a disposable environment !690
- Adds EmptyDir support for k8s volumes !660
- Update git-lfs to 2.3.1 !703
- Collect metrics on build stages !689
- Construct git remote URL based on configuration !698
- Set git SSL information only for gitlab host !687

v 10.0.2 (2017-10-04)
- Hide tokens from URLs printed in job's trace !708

v 10.0.1 (2017-09-27)
- Remove deprecation message from service management commands !699

v 10.0.0 (2017-09-22)

> **Note:** With 10.0, we've moved repository from https://gitlab.com/gitlab-org/gitlab-ci-multi-runner
to https://gitlab.com/gitlab-org/gitlab-runner. Please update your Bookmarks!

> **Note:** Starting with 10.0, we're marking the `exec` and service-related commands as **deprecated**. They will
be removed in one of the upcoming releases.

> **Note:** Starting with 10.0, we're marking the `docker-ssh` and `docker-ssh+machine` executors as **deprecated**.
They will be removed in one of the upcoming releases.

> **Note:** Starting with 10.0, behavior of `register` command was slightly changed. Please look into
https://gitlab.com/gitlab-org/gitlab-runner/merge_requests/657 for more details.

- Lock runners to project by default on registration !657
- Update cli library !656
- Fix RunSingleCommand race condition in waitForInterrupts !594
- Add handling of non-existing images for Docker >= 17.07 !664
- Document how to define default image to run using Kubernetes executor !668
- Specify an explicit length for git rev-parse --short to avoid conflicts when run !672
- Add link to Kubernetes executor details !670
- Add install VirtualBox step & improve VM setup details !676
- Rename repository from gitlab-ci-multi-runner to gitlab-runner !661
- Fix variable file permission !655
- Add Release Checklist template !677
- Fix randomly failing test from commands/single_test.go !684
- Mark docker-ssh and docker-ssh+machine executors as DEPRECATED !681
- Mark exec and service-management commands as DEPRECATED !679
- Fix support for `tmpfs` in docker executor config !680

v 9.5.1 (2017-10-04)
- Hide tokens from URLs printed in job's trace !708
- Add handling of non-existing images for Docker >= 17.07 !664

v 9.5.0 (2017-08-22)
- Fix allowed_images behavior !635
- Cleanup formatting on windows upgrade details !637
- Names must meet the DNS name requirements (no upper case) !636
- Execute steps for build as-is, without joining and splitting them !626
- Fix typo on killall command !638
- Fix usage of one image for multiple services in one job !639
- Update Docker Machine to 0.12.2 and add checksum checking for Docker Machine and dumb-init for official Docker images !640
- Fix services usage when service name is using variable !641
- Remove confusing compatibility check !642
- Add sysctl support for Docker executor !541
- Reduce binary size with removing debugging symbols !643
- Add support for credentials store !501
- Fix I am not sure section link !650
- Add tzdata by default to official Docker images to avoid OffPeakPeriods timezone error !649
- Fix read error from upload artifacts execution !645
- Add support for tmpfs on the job container !654
- Include note about volume path on OSX !648
- Start using 'toc' in yaml frontmatter to explicitly disable it !644

v 9.4.3 (2017-10-04)
- Hide tokens from URLs printed in job's trace !708
- Add handling of non-existing images for Docker >= 17.07 !664

v 9.4.2 (2017-08-02)
- Fix usage of one image for multiple services in one job !639
- Fix services usage when service name is using variable !641

v 9.4.1 (2017-07-25)
- Fix allowed_images behavior !635

v 9.4.0 (2017-07-22)
- Use Go 1.8 for CI !620
- Warn on archiving git directory !591
- Add CacheClient with timeout configuration for cache operations !608
- Remove '.git/hooks/post-checkout' hooks when using fetch strategy !603
- Fix VirtualBox and Parallels executors registration bugs !589
- Support Kubernetes PVCs !606
- Support cache policies in .gitlab-ci.yml !621
- Improve kubernetes volumes support !625
- Adds an option `--all` to unregister command !622
- Add the technical description of version release !631
- Update documentation on building docker images inside of a kubernetes cluster. !628
- Support for extended docker configuration in gitlab-ci.yml !596
- Add ServicesTmpfs options to Docker runner configuration. !605
- Fix network timeouts !634

v 9.3.0 (2017-06-22)
- Make GitLab Runner metrics HTTP endpoint default to :9252 !584
- Add handling for GIT_CHECKOUT variable to skip checkout !585
- Use HTTP status code constants from net/http library !569
- Remove tls-skip-verify from advanced-configuration.md !590
- Improve docker machine removal !582
- Add support for Docker '--cpus' option !586
- Add requests backoff mechanism !570
- Fixed doc typo, change `--service-name` to `--service` !592
- Slight fix to build/ path in multi runner documentation !598
- Move docs on private Registry to GitLab docs !597
- Install Git LFS in Helper image for X86_64 !588
- Docker entrypoint: use exec !581
- Create gitlab-runner user on alpine !593
- Move registering Runners info in a separate document !599
- Add basic support for Kubernetes volumes !516
- Add required runners.docker section to example config. !604
- Add userns support for Docker executor !553
- Fix another regression on docker-machine credentials usage !610
- Added ref of Docker app installation !612
- Update linux-repository.md !615

v 9.2.2 (2017-07-04)
- Fix VirtualBox and Parallels executors registration bugs !589

v 9.2.1 (2017-06-17)
- Fix regression introduced in the way how `exec` parses `.gitlab-ci.yml` !535
- Fix another regression on docker-machine credentials usage !610

v 9.2.0 (2017-05-22)

This release introduces a change in the ordering of artifacts and cache restoring!

It may happen that someone, by mistake or by purpose, uses the same path in
`.gitlab-ci.yml` for both cache and artifacts keywords, and this could cause that
a stale cache might inadvertently override artifacts that are used across the
pipeline.

Starting with this release, artifacts are always restored after the cache to ensure
that even in edge cases you can always rely on them.

- Improve Windows runner details !514
- Add support for TLS client authentication !157
- Fix apt-get syntax to install a specific version. !563
- Add link to Using Docker Build CI docs !561
- Document the `coordinator` and make the FAQ list unordered !567
- Add links to additional kubernetes details !566
- Add '/debug/jobs/list' endpoint that lists all handled jobs !564
- Remove .godir !568
- Add PodLabels field to Kubernetes config structure !558
- Remove the build container after execution has completed !571
- Print proper message when cache upload operation failed !556
- Remove redundant ToC from autoscale docs and add intro paragraph !574
- Make possible to compile Runner under Openbsd2 !511
- Improve docker configuration docs !576
- Use contexes everywhere !559
- Add support for kubernetes service account and override on gitlab-ci.yaml !554
- Restore cache before artifacts !577
- Fix link to the LICENSE file. !579

v 9.1.3 (2017-07-04)
- Fix VirtualBox and Parallels executors registration bugs !589

v 9.1.2 (2017-06-17)
- Print proper message when cache upload operation fails !556
- Fix regression introduced in the way how `exec` parses `.gitlab-ci.yml` !535

v 9.1.1 (2017-05-02)
- Fix apt-get syntax to install a specific version. !563
- Remove the build container after execution has completed !571

v 9.1.0 (2017-04-22)
- Don't install docs for the fpm Gem !526
- Mention tagged S3 sources in installation documentation !513
- Extend documentation about accessing docker services !527
- Replace b.CurrentStage with b.CurrentState where it was misused !530
- Docker provider metrics cleanups and renaming !531
- Replace godep with govendor !505
- Add histogram metrics for docker machine creation !533
- Fix cache containers dicsovering regression !534
- Add urls to environments created with CI release jobs !537
- Remove unmanaged docker images sources !538
- Speed up CI pipeline !536
- Add job for checking the internal docs links !542
- Mention Runner -> GitLab compatibility concerns after 9.0 release !544
- Log error if API v4 is not present (GitLab CE/EE is older than 9.0) !528
- Cleanup variables set on GitLab already !523
- Add faq entry describing how to handle missing zoneinfo.zip problem !543
- Add documentation on how Runner uses Minio library !419
- Update docker.md - typo in runners documentation link !546
- Add log_level option to config.toml !524
- Support private registries with Kubernetes !551
- Cleanup Kubernetes typos and wording !550
- Fix runner crashing on builds helper collect !529
- Config docs: Fix syntax in example TOML for Kubernetes !552
- Docker: Allow to configure shared memory size !468
- Return error for cache-extractor command when S3 cache source returns 404 !429
- Add executor stage to ci_runner_builds metric's labels !548
- Don't show image's ID when it's the same as image's name !557
- Extended verify command with runner selector !532
- Changed information line logged by Runner while unregistering !540
- Properly configure connection timeouts and keep-alives !560
- Log fatal error when concurrent is less than 1 !549

v 9.0.4 (2017-05-02)
- Fix apt-get syntax to install a specific version. !563
- Remove the build container after execution has completed !571

v 9.0.3 (2017-04-21)
- Fix runner crashing on builds helper collect !529
- Properly configure connection timeouts and keep-alives !560

v 9.0.2 (2017-04-06)
- Speed up CI pipeline !536

v 9.0.1 (2017-04-05)
- Don't install docs for the fpm Gem !526
- Mention tagged S3 sources in installation documentation !513
- Replace b.CurrentStage with b.CurrentState where it was misused !530
- Replace godep with govendor !505
- Fix cache containers dicsovering regression !534
- Add urls to environments created with CI release jobs !537
- Mention Runner -> GitLab compatibility concerns after 9.0 release !544
- Log error if API v4 is not present (GitLab CE/EE is older than 9.0) !528

v 9.0.0
- Change dependency from `github.com/fsouza/go-dockerclient` to `github.com/docker/docker/client`" !301
- Update docker-machine version to fix coreos provision !500
- Cleanup windows install docs !497
- Replace io.Copy with stdcopy.StdCopy for docker output handling !503
- Fixes typo: current to concurrent. !508
- Modifies autoscale algorithm example !509
- Force-terminate VirtualBox and Parallels VMs so snapshot restore works properly !313
- Fix indentation of 'image_pull_secrets' in kubernetes configuration example !512
- Show Docker image ID in job's log !507
- Fix word consistency in autoscaling docs !519
- Rename the binary on download to use gitlab-runner as command !510
- Improve details around limits !502
- Switch from CI API v1 to API v4 !517
- Make it easier to run tests locally !506
- Kubernetes private credentials !520
- Limit number of concurrent requests to builds/register.json !518
- Remove deprecated kubernetes executor configuration fields !521
- Drop Kubernetes executor 'experimental' notice !525

v 1.11.5 (2017-07-04)
- Fix VirtualBox and Parallels executors registration bugs !589

v 1.11.4 (2017-04-28)
- Fixes test that was failing 1.11.3 release

v 1.11.3 (2017-04-28)
- Add urls to environments created with CI release jobs !537
- Speed up CI pipeline !536
- Fix runner crashing on builds helper collect !529

v 1.11.2
- Force-terminate VirtualBox and Parallels VMs so snapshot restore works properly !313
- Don't install docs for the fpm Gem !526
- Mention tagged S3 sources in installation documentation !513
- Limit number of concurrent requests to builds/register.json !518
- Replace b.CurrentStage with b.CurrentState where it was misused !530

v 1.11.1
- Update docker-machine version to fix coreos provision !500

v 1.11.0
- Fix S3 and packagecloud uploads step in release process !455
- Add ubuntu/yakkety to packages generation list !458
- Reduce size of gitlab-runner-helper images !456
- Fix crash on machine creation !461
- Rename 'Build (succeeded|failed)' to 'Job (succeeded|failed)' !459
- Fix race in helpers/prometheus/log_hook.go: Fire() method !463
- Fix missing VERSION on Mac build !465
- Added post_build_script to call scripts after user-defined build scripts !460
- Fix offense reported by vet. Add vet to 'code style' job. !477
- Add the runner name to the first line of log output, after the version !473
- Make CI_DEBUG_TRACE working on Windows CMD !483
- Update packages targets !485
- Update Makefile (fix permissions on /usr/share/gitlab-runner/) !487
- Add timezone support for OffPeak intervals !479
- Set GIT_SUBMODULE_STRATEGY=SubmoduleNone when GIT_STRATEGY=GitNone !480
- Update maintainers information !489

v 1.10.8
- Force-terminate VirtualBox and Parallels VMs so snapshot restore works properly !313
- Don't install docs for the fpm Gem !526
- Mention tagged S3 sources in installation documentation !513
- Limit number of concurrent requests to builds/register.json !518
- Replace b.CurrentStage with b.CurrentState where it was misused !530

v 1.10.7
- Update docker-machine version to fix coreos provision !500

v 1.10.6
- Update Makefile (fix permissions on /usr/share/gitlab-runner/) !487

v 1.10.5
- Update packages targets !485

v 1.10.4
- Fix race in helpers/prometheus/log_hook.go: Fire() method !463

v 1.10.3
- Fix crash on machine creation !461

v 1.10.2
- Add ubuntu/yakkety to packages generation list !458
- Reduce size of gitlab-runner-helper images !456

v 1.10.1
- Fix S3 and packagecloud uploads step in release process !455

v 1.10.0
- Make /usr/share/gitlab-runner/clear-docker-cache script /bin/sh compatible !427
- Handle Content-Type header with charset information !430
- Don't raise error if machines directory is missing on machines listing !433
- Change digital ocean autoscale to use stable coreos channel !434
- Fix package's scripts permissions !440
- Use -q flag instead of --format. !442
- Kubernetes termination grace period !383
- Check if directory exists before recreating it with Windows CMD !435
- Add '--run-tagged-only' cli option for runners !438
- Add armv6l to the ARM replacements list for docker executor helper image !446
- Add configuration options for Kubernetss resource requests !391
- Add poll interval and timeout parameters for Kubernetes executor !384
- Add support for GIT_SUBMODULE_STRATEGY !443
- Create index file for S3 downloads !452
- Add Prometheus metric that counts number of catched errors !439
- Exclude unused options from AbstractExecutor.Build.Options !445
- Update Docker Machine in official Runner images to v0.9.0 !454
- Pass ImagePullSecrets for Kubernetes executor !449
- Add Namespace overwrite possibility for Kubernetes executor !444

v 1.9.10
- Force-terminate VirtualBox and Parallels VMs so snapshot restore works properly !313

v 1.9.9
- Update docker-machine version to fix coreos provision !500

v 1.9.8
- Update Makefile (fix permissions on /usr/share/gitlab-runner/) !487

v 1.9.7
- Update packages targets !485

v 1.9.6
- Add ubuntu/yakkety to packages generation list !458

v 1.9.5
- Update Docker Machine in official Runner images to v0.9.0 !454

v 1.9.4
- Add armv6l to the ARM replacements list for docker executor helper image !446

v 1.9.3
- Fix package's scripts permissions !440
- Check if directory exists before recreating it with Windows CMD !435

v 1.9.2
- Handle Content-Type header with charset information !430
- Don't raise error if machines directory is missing on machines listing !433

v 1.9.1
- Make /usr/share/gitlab-runner/clear-docker-cache script /bin/sh compatible !427

v 1.9.0
- Add pprof HTTP endpoints to metrics server !398
- Add a multiple prometheus metrics: !401
- Split prepare stage to be: prepare, git_clone, restore_cache, download_artifacts !406
- Update CONTRIBUTING.md to refer to go 1.7.1 !409
- Introduce docker.Client timeouts !411
- Allow network-sourced variables to specify that they should be files !413
- Add a retry mechanism to prevent failed clones in builds !399
- Remove shallow.lock before fetching !407
- Colorize log entries for cmd and powershell !400
- Add section describing docker usage do Kubernetes executor docs !394
- FreeBSD runner installation docs update !387
- Update prompts for register command !377
- Add volume_driver Docker configuration file option !365
- Fix bug permission denied on ci build with external cache !347
- Fix entrypoint for alpine image !346
- Add windows vm checklist for virtualbox documentation !348
- Clarification around authentication with the Kubernetes executor !296
- Fix docker hanging for docker-engine 1.12.4 !415
- Use lib machine to fetch a list of docker-machines !418
- Cleanup docker cache clear script !388
- Allow the --limit option to control the number of jobs a single runner will run !369
- Store and send last_update value with API calls against GitLab !410
- Add graceful shutdown documentation !421
- Add Kubernete Node Selector !328
- Push prebuilt images to dockerhub !420
- Add path and share cache settings for S3 cache !423
- Remove unnecessary warning about using image with the same ID as provided !424
- Add a link where one can download the packages directly !292
- Kubernetes executor - use pre-build container !425

v 1.8.8
- Update Makefile (fix permissions on /usr/share/gitlab-runner/) !487

v 1.8.7
- Update packages targets !485

v 1.8.6
- Add ubuntu/yakkety to packages generation list !458

v 1.8.5
- Update Docker Machine in official Runner images to v0.9.0 !454

v 1.8.4
- Add armv6l to the ARM replacements list for docker executor helper image !446

v 1.8.3
- Fix package's scripts permissions !440
- Check if directory exists before recreating it with Windows CMD !435

v 1.8.2
- Handle Content-Type header with charset information !430

v 1.8.1
- Rrefactor the private container registry docs !392
- Make pull policies usage clear !393

v 1.8.0
- Fix {Bash,Cmd,Ps}Writer.IfCmd to escape its arguments !364
- Fix path to runners-ssh page !368
- Add initial Prometheus metrics server to runner manager !358
- Add a global index.md for docs !371
- Ensure that all builds are executed on tagged runners !374
- Fix broken documentation links !382
- Bug Fix: use a regex to pull out the service and version in the splitServiceAndVersion method !376
- Add FAQ entry about handling the service logon failure on Windows !385
- Fix "unit tests" random failures !370
- Use correct constant for kubernetes ressource limits. !367
- Unplug stalled endpoints !390
- Add PullPolicy config option for kubernetes !335
- Handle received 'failed' build state while patching the trace !366
- Add support for using private docker registries !386

v 1.7.5
- Update Docker Machine in official Runner images to v0.9.0 !454

v 1.7.4
- Add armv6l to the ARM replacements list for docker executor helper image !446

v 1.7.3
- Fix package's scripts permissions !440
- Check if directory exists before recreating it with Windows CMD !435

v 1.7.2
- Handle Content-Type header with charset information !430

v 1.7.1
- Fix {Bash,Cmd,Ps}Writer.IfCmd to escape its arguments !364

v 1.7.0
- Improve description of --s3-bucket-location option !325
- Use Go 1.7 !323
- Add changelog entries generation script !322
- Add docker_images release step to CI pipeline !333
- Refactor shell executor tests !334
- Introduce GIT_STRATEGY=none !332
- Introduce a variable to enable shell tracing on bash, cmd.exe and powershell.exe !339
- Try to load the InCluster config first, if that fails load kubectl config !327
- Squash the "No TLS connection state" warning !343
- Add a benchmark for helpers.ShellEscape and optimise it !351
- Godep: update github.com/Sirupsen/logrus to v0.10.0 !344
- Use git clone --no-checkout and git checkout --force !341
- Change machine.machineDetails to machine.Details !353
- Make runner name lowercase to work with GCE restrictions !297
- Add per job before_script handling for exec command !355
- Add OffPeak support for autoscaling !345
- Prevent caching failures from marking a build as failed !359
- Add missed "server" command for minio in autoscaled S3 cache tutorial !361
- Add a section for Godep in CONTRIBUTING.md !302
- Add a link to all install documentation files describing how to obtain a registration token !362
- Improve registration behavior !356
- Add the release process description !176
- Fix documentation typo in docs/configuration/advanced-configuration.md !354
- Fix data races around runner health and build stats !352

v 1.6.1
- Add changelog entries generation script !322
- Add docker_images release step to CI pipeline !333

v 1.6.0
- Remove an unused method from the Docker executor !280
- Add note about certificate concatenation !278
- Restore 755 mode for gitlab-runner-service script !283
- Remove git-lfs from docker helper images !288
- Improve Kubernetes support !277
- docs: update troubleshooting section in development. !286
- Windows installation, added a precision on the install command (issue related #1265) !223
- Autodetect "/ci" in URL !289
- Defer removing failed containers until Cleanup() !281
- fix typo in tls-self-signed.md !294
- Improve CI tests !276
- Generate a BuildError when Docker/Kubernetes image is missing !295
- cmd.exe: Caret-escape parentheses when not inside double quotes !284
- Fixed some spelling/grammar mistakes. !291
- Update Go instructions in README !175
- Add APT pinning configuration for debian in installation docs !303
- Remove yaml v1 !307
- Add options to runner configuration to specify commands executed before code clone and build !106
- Add RC tag support and fix version discovering !312
- Pass all configured CA certificates to builds !299
- Use git-init templates (clone) and git config without --global (fetch) to disable recurseSubmodules !314
- Improve docker machine logging !234
- Add possibility to specify a list of volumes to inherit from another container !236
- Fix range mismatch handling error while patch tracing !319
- Add docker+machine and kubernetes executors to "I'm not sure" part of executors README.md !320
- Remove ./git/index.lock before fetching !316

v 1.5.3
- Fix Caret-escape parentheses when not inside double quotes for Windows cmd
- Remove LFS from prebuilt images

v 1.5.2
(no changes)

v 1.5.1
- Fix file mode of gitlab-runner-service script !283

v 1.5.0
- Update vendored toml !258
- Release armel instead arm for Debian packages !264
- Improve concurrency of docker+machine executor !254
- Use .xz for prebuilt docker images to reduce binary size and provisioning speed of Docker Engines !249
- Remove vendored test files !271
- Update gitlab-runner-service to return 1 when no Host or PORT is defined !253
- Log caching URL address
- Retry executor preparation to reduce system failures !244
- Fix missing entrypoint script in alpine Dockerfile !248
- Suppress all but the first warning of a given type when extracting a ZIP file !261
- Mount /builds folder to all services when used with Docker Executor !272
- Cache docker client instances to avoid a file descriptor leak !260
- Support bind mount of `/builds` folder !193

v 1.4.3
- Fix Caret-escape parentheses when not inside double quotes for Windows cmd
- Remove LFS from prebuilt images

v 1.4.2
- Fix abort mechanism when patching trace

v 1.4.1
- Fix panic while artifacts handling errors

v 1.4.0
- Add sentry support
- Add support for cloning VirtualBox VM snapshots as linked clones
- Add support for `security_opt` docker configuration parameter in docker executor
- Add first integration tests for executors
- Add many logging improvements (add more details to some logs, move some logs to Debug level, refactorize logger etc.)
- Make final build trace upload be done before cleanup
- Extend support for caching and artifacts to all executors
- Improve support for Docker Machine
- Improve build aborting
- Refactor common/version
- Use `environment` feature in `.gitlab-ci.yml` to track latest versions for Bleeding Edge and Stable
- Fix Absolute method for absolute path discovering for bash
- Fix zombie issues by using dumb-init instead of github.com/ramr/go-reaper

v 1.3.5
- Fix Caret-escape parentheses when not inside double quotes for Windows cmd

v 1.3.4
- Fix panic while artifacts handling errors

v 1.3.3
- Fix zombie issue by using dumb-init

v 1.3.2
- Fix architecture detection bug introduced in 1.3.1

v 1.3.1
- Detect architecture if not given by Docker Engine (versions before 1.9.0)

v 1.3.0
- Add incremental build trace update
- Add possibility to specify CpusetCpus, Dns and DnsSearch for docker containers created by runners
- Add a custom `User-Agent` header with version number and runtime information (go version, platform, os)
- Add artifacts expiration handling
- Add artifacts handling for failed builds
- Add customizable `check_interval` to set how often to check GitLab for a new builds
- Add docker Machine IP address logging
- Make Docker Executor ARM compatible
- Refactor script generation to make it fully on-demand
- Refactor runnsers Acquire method to improve performance
- Fix branch name setting at compile time
- Fix panic when generating log message if provision of node fails
- Fix docker host logging
- Prevent leaking of goroutines when aborting builds
- Restore valid version info in --help message
- [Experimental] Add `GIT_STRATEGY` handling - clone/fetch strategy configurable per job
- [Experimental] Add `GIT_DEPTH` handling - `--depth` parameter for `git fetch` and `git clone`

v 1.2.0
- Use Go 1.6
- Add `timeout` option for the `exec` command
- Add runtime platform information to debug log
- Add `docker-machine` binary to Runner's official docker images
- Add `build_current` target to Makefile - to build only a binary for used architecture
- Add support for `after_script`
- Extend version information when using `--version` flag
- Extend artifacts download/upload logs with more response data
- Extend unregister command to accept runner name
- Update shell detection mechanism
- Update the github.com/ayufan/golag-kardianos-service dependency
- Replace ANSI_BOLD_YELLOW with ANSI_YELLOW color for logging
- Reconcile VirtualBox status constants with VBoxManage output values
- Make checkout quiet
- Make variables to work at job level in exec mode
- Remove "user mode" warning when running in a system mode
- Create `gitlab-runner` user as a system account
- Properly create `/etc/gitlab-runner/certs` in Runner's official docker images
- Disable recursive submodule fetchin on fetching changes
- Fix nil casting issue on docker client creation
- Fix used build platforms for `gox`
- Fix a limit problems when trying to remove a non-existing machines
- Fix S3 caching issues
- Fix logging messages on artifacts dowloading
- Fix binary panic while using VirtualBox executor with no `vboxmanage` binary available

v 1.1.4
- Create /etc/gitlab-runner/certs
- Exclude architectures from GOX, rather then including
- Update mimio-go to a newest version
- Regression: Implement CancelRequest to fix S3 caching support
- Fix: Skip removal of machine that doesn't exist (autoscaling)

v 1.1.3
- Regression: On Linux use `sh -s /bin/bash user -c` instead of `sh user -c`. This fixes non-login for user.
- Regression: Fix user mode warning
- Fix: vet installation
- Fix: nil casting issue on docker client creation
- Fix: docker client download issue

v 1.1.2
- Regression: revert shell detection mechanism and limit it only to Docker

v 1.1.1
- Fix: use different shell detection mechanism
- Regression: support for `gitlab-runner exec`
- Regression: support for login/non-login shell for Bash

v 1.1.0
- Use Go 1.5
- Change license to MIT
- Add docker-machine based auto-scaling for docker executor
- Add support for external cache server
- Add support for `sh`, allowing to run builds on images without the `bash`
- Add support for passing the artifacts between stages
- Add `docker-pull-policy`, it removes the `docker-image-ttl`
- Add `docker-network-mode`
- Add `git` to gitlab-runner:alpine
- Add support for `CapAdd`, `CapDrop` and `Devices` by docker executor
- Add support for passing the name of artifacts archive (`artifacts:name`)
- Add support for running runner as system service on OSX
- Refactor: The build trace is now implemented by `network` module
- Refactor: Remove CGO dependency on Windows
- Fix: Create alternative aliases for docker services (uses `-`)
- Fix: VirtualBox port race condition
- Fix: Create cache for all builds, including tags
- Fix: Make the shell executor more verbose when the process cannot be started
- Fix: Pass gitlab-ci.yml variables to build container created by docker executor
- Fix: Don't restore cache if not defined in gitlab-ci.yml
- Fix: Always use `json-file` when starting docker containers
- Fix: Error level checking for Windows Batch and PowerShell

v 1.0.4
- Fix support for Windows PowerShell

v 1.0.3
- Fix support for Windows Batch
- Remove git index lock file: this solves problem with git checkout being terminated
- Hijack docker.Client to use keep-alives and to close extra connections

v 1.0.2
- Fix bad warning about not found untracked files
- Don't print error about existing file when restoring the cache
- When creating ZIP archive always use forward-slashes and don't permit encoding absolute paths
- Prefer to use `path` instead of `filepath` which is platform specific: solves the docker executor on Windows

v 1.0.1
- Use nice log formatting for command line tools
- Don't ask for services during registration (we prefer the .gitlab-ci.yml)
- Create all directories when extracting the file

v 1.0.0
- Add `gitlab-runner exec` command to easy running builds
- Add `gitlab-runner status` command to easy check the status of the service
- Add `gitlab-runner list` command to list all runners from config file
- Allow to specify `ImageTTL` for configuration the frequency of docker image re-pulling (see advanced-configuration)
- Inject TLS certificate chain for `git clone` in build container, the gitlab-runner SSL certificates are used
- Remove TLSSkipVerify since this is unsafe option
- Add go-reaper to make gitlab-runner to act as init 1 process fixing zombie issue when running docker container
- Create and send artifacts as zip files
- Add internal commands for creating and extracting archives without the system dependencies
- Add internal command for uploading artifacts without the system dependencies
- Use umask in docker build containers to fix running jobs as specific user
- Fix problem with `cache` paths never being archived
- Add support for [`cache:key`](http://doc.gitlab.com/ce/ci/yaml/README.html#cachekey)
- Add warnings about using runner in `user-mode`
- Push packages to all upcoming distributions (Debian/Ubuntu/Fedora)
- Rewrite the shell support adding all features to all shells (makes possible to use artifacts and caching on Windows)
- Complain about missing caching and artifacts on some executors
- Added VirtualBox executor
- Embed prebuilt docker build images in runner binary and load them if needed
- Make possible to cache absolute paths (unsafe on shell executor)

v 0.7.2
- Adjust `umask` for build image
- Use absolute path when executing archive command
- Fix regression when variables were not passed to service container
- Fix duplicate files in cache or artifacts archive

v 0.7.1
- Fix caching support
- Suppress tar verbose output

v 0.7.0
- Refactor code structure
- Refactor bash script adding pre-build and post-build steps
- Add support for build artifacts
- Add support for caching build directories
- Add command to generate archive with cached folders or artifacts
- Use separate containers to run pre-build (git cloning), build (user scripts) and post-build (uploading artifacts)
- Expand variables, allowing to use $CI_BUILD_TAG in image names, or in other variables
- Make shell executor to use absolute path for project dir
- Be strict about code formatting
- Move network related code to separate package
- Automatically load TLS certificates stored in /etc/gitlab-runner/certs/<hostname>.crt
- Allow to specify tls-ca-file during registration
- Allow to disable tls verification during registration

v 0.6.2
- Fix PowerShell support
- Make more descriptive pulling message
- Add version check to Makefile

v 0.6.1
- Revert: Fix tags handling when using git fetch: fetch all tags and prune the old ones

v 0.6.0
- Fetch docker auth from ~/.docker/config.json or ~/.dockercfg
- Added support for NTFSSecurity PowerShell module to address problems with long paths on Windows
- Make the service startup more readable in case of failure: print a nice warning message
- Command line interface for register and run-single accepts all possible config parameters now
- Ask about tags and fix prompt to point to gitlab.com/ci
- Pin to specific Docker API version
- Fix docker volume removal issue
- Add :latest to imageName if missing
- Pull docker images every minute
- Added support for SIGQUIT to allow to gracefully finish runner: runner will not accept new jobs, will stop once all current jobs are finished.
- Implicitly allow images added as services
- Evaluate script command in subcontext, making it to close stdin (this change since 0.5.x where the separate file was created)
- Pass container labels to docker
- Force to use go:1.4 for building packages
- Fix tags handling when using git fetch: fetch all tags and prune the old ones
- Remove docker socket from gitlab/gitlab-runner images
- Pull (update) images and services every minute
- Ignore options from Coordinator that are null
- Provide FreeBSD binary
- Use -ldflags for versioning
- Update go packages
- Fix segfault on service checker container
- WARNING: By default allow to override image and services

v 0.5.5
- Fix cache_dir handling

v 0.5.4
- Update go-dockerclient to fix problems with creating docker containers

v 0.5.3
- Pin to specific Docker API version
- Fix docker volume removal issue

v 0.5.2
- Fixed CentOS6 service script
- Fixed documentation
- Added development documentation
- Log service messages always to syslog

v 0.5.1
- Update link for Docker configuration

v 0.5.0
- Allow to override image and services for Docker executor from Coordinator
- Added support for additional options passed from coordinator
- Added support for receiving and defining allowed images and services from the Coordinator
- Rename gitlab_ci_multi_runner to gitlab-runner
- Don't require config file to exist in order to run runner
- Change where config file is stored: /etc/gitlab-runner/config.toml (*nix, root), ~/.gitlab-runner/config.toml (*nix, user)
- Create config on service install
- Require root to control service on Linux
- Require to specify user when installing service
- Run service as root, but impersonate as --user when executing shell scripts
- Migrate config.toml from user directory to /etc/gitlab-runner/
- Simplify service installation and upgrade
- Add --provides and --replaces to package builder
- Powershell: check exit code in writeCommandChecked
- Added installation tests
- Add runner alpine-based image
- Send executor features with RunnerInfo
- Verbose mode by using `echo` instead of `set -v`
- Colorize bash output
- Set environment variables from bash script: this fixes problem with su
- Don't cache Dockerfile VOLUMEs
- Pass (public) environment variables received from Coordinator to service containers

v 0.4.2
- Force GC cycle after processing build
- Use log-level set to info, but also make `Checking for builds: nothing` being print as debug
- Fix memory leak - don't track references to builds

v 0.4.1
- Fixed service reregistration for RedHat systems

v 0.4.0
- Added CI=true and GITLAB_CI=true to environment variables
- Added output_limit (in kilobytes) to runner config which allows to enlarge default build log size
- Added support for custom variables received from CI
- Added support for SSH identity file
- Optimize build path to make it shorter, more readable and allowing to fix shebang issue
- Make the debug log human readable
- Make default build log limit set to 4096 (4MB)
- Make default concurrent set to 1
- Make default limit for runner set to 1 during registration
- Updated kardianos service to fix OSX service installation
- Updated logrus to make console output readable on Windows
- Change default log level to warning
- Make selection of forward or back slashes dependent by shell not by system
- Prevent runner to be stealth if we reach the MaxTraceOutputSize
- Fixed Windows Batch script when builds are located on different drive
- Fixed Windows runner
- Fixed installation scripts path
- Fixed wrong architecture for i386 debian packages
- Fixed problem allowing commands to consume build script making the build to succeed even if not all commands were executed

v 0.3.4
- Create path before clone to fix Windows issue
- Added CI=true and GITLAB_CI=true
- Fixed wrong architecture for i386 debian packages

v 0.3.3
- Push package to ubuntu/vivid and ol/6 and ol/7

v 0.3.2
- Fixed Windows batch script generator

v 0.3.1
- Remove clean_environment (it was working only for shell scripts)
- Run bash with --login (fixes missing .profile environment)

v 0.3.0
- Added repo slug to build path
- Build path includes repository hostname
- Support TLS connection with Docker
- Default concurrent limit is set to number of CPUs
- Make most of the config options optional
- Rename setup/delete to register/unregister
- Checkout as detached HEAD (fixes compatibility with older git versions)
- Update documentation

v 0.2.0
- Added delete and verify commands
- Limit build trace size (1MB currently)
- Validate build log to contain only valid UTF-8 sequences
- Store build log in memory
- Integrate with ci.gitlab.com
- Make packages for ARM and CentOS 6 and provide beta version
- Store Docker cache in separate containers
- Support host-based volumes for Docker executor
- Don't send build trace if nothing changed
- Refactor build class

v 0.1.17
- Fixed high file descriptor usage that could lead to error: too many open files

v 0.1.16
- Fixed systemd service script

v 0.1.15
- Fix order of executor commands
- Fixed service creation options
- Fixed service installation on OSX

v 0.1.14
- Use custom kardianos/service with enhanced service scripts
- Remove all system specific packages and use universal for package manager

v 0.1.13
- Added abstraction over shells
- Moved all bash specific stuff to shells/bash.go
- Select default shell for OS (bash for Unix, batch for Windows)
- Added Windows Cmd support
- Added Windows PowerShell support
- Added the kardianos/service which allows to easily run gitlab-ci-multi-runner as service on different platforms
- Unregister Parallels VMs which are invalid
- Delete Parallels VM if it doesn't contain snapshots
- Fixed concurrency issue when assigning unique names

v 0.1.12
- Abort all jobs if interrupt or SIGTERM is received
- Runner now handles HUP and reloads config on-demand
- Refactored runner setup allowing to non-interactive configuration of all questioned parameters
- Added CI_PROJECT_DIR environment variable
- Make golint happy (in most cases)

v 0.1.11
- Package as .deb and .rpm and push it to packagecloud.io (for now)

v 0.1.10
- Wait for docker service to come up (Loïc Guitaut)
- Send build log as early as possible

v 0.1.9
- Fixed problem with resetting ruby environment

v 0.1.8
- Allow to use prefixed services
- Allow to run on Heroku
- Inherit environment variables by default for shell scripts
- Mute git messages during checkout
- Remove some unused internal messages from build log

v 0.1.7
- Fixed git checkout

v 0.1.6
- Remove Docker containers before starting job

v 0.1.5
- Added Parallels executor which can use snapshots for fast revert (only OSX supported)
- Refactored sources

v 0.1.4
- Remove Job and merge it into Build
- Introduce simple API server
- Ask for services during setup

v 0.1.3
- Optimize setup
- Optimize multi-runner setup - making it more concurrent
- Send description instead of hostname during registration
- Don't ask for tags

v 0.1.2
- Make it work on Windows

v 0.1.1
- Added Docker services

v 0.1.0
- Initial public release

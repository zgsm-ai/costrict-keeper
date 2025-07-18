#!/usr/bin/env python3
# -*- coding: UTF-8 -*-

import os, time, subprocess, sys
# --debug
# --install
# --software
# --protocol
opt_debug = False
opt_install = False
opt_software = "1.0.20250603"
opt_protocol = "1.0.20250603"
opt_app = "costrict-host"

def run_cmd(cmd):
    p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout = p.communicate()[0].decode('utf-8').strip()
    return stdout

# Get last tag.
def last_tag():
    return run_cmd('git rev-parse --abbrev-ref HEAD')

# Get last git commit id.
def last_commit_id():
    return run_cmd('git log --pretty=format:"%h" -1')

# Assemble build command.
def build_cmd():
    build_flags = []

    build_flags.append("-X 'code.sangfor.com/shenma/{0}/cmd.SoftwareVer={1}'".format(opt_app, opt_software))
    build_flags.append("-X 'code.sangfor.com/shenma/{0}/cmd.ProtocolVer={1}'".format(opt_app, opt_protocol))
    last_git_tag = last_tag()
    if last_git_tag != "":
        build_flags.append("-X 'code.sangfor.com/shenma/{0}/cmd.BuildTag={1}'".format(opt_app, last_git_tag))

    commit_id = last_commit_id()
    if commit_id != "":
        build_flags.append("-X 'code.sangfor.com/shenma/{0}/cmd.BuildCommitId={1}'".format(opt_app, commit_id))

    # current time
    build_flags.append("-X 'code.sangfor.com/shenma/{0}/cmd.BuildTime={1}'".format(opt_app, 
        time.strftime("%Y-%m-%d %H:%M:%S")))

    debug_flag = ""
    if opt_debug:
        debug_flag = '-gcflags=all="-N -l"'

    if opt_install:
        return 'go install {0} -ldflags "{1}"'.format(debug_flag, " ".join(build_flags))
    else:
        return 'go build {0} -ldflags "{1}"'.format(debug_flag, " ".join(build_flags))

def parse_opts():
    global opt_debug
    global opt_install
    global opt_protocol
    global opt_software
    global opt_app
    argc = len(sys.argv)
    if argc == 1:
        return True
    i = 1
    while i < argc:
        arg = sys.argv[i]
        if arg == '-h':
            print("build.py [--debug] [--install] [--software VER] [--protocol VER] [--app APPNAME]")
            print("  -d,--debug        编译调试版本")
            print("  -i,--install      把程序拷贝到安装目录")
            print("  -s,--software VER 指定软件版本,VER格式:x.x.x,如: 1.1.1210")
            print("  -p,--protocol VER RESTful API的版本,VER格式:x.x.x,如: 1.1.1210")
            print("  -a,--app APPNAME  当前构建的程序名字")
            return False
        elif arg == '-d' or arg == '--debug':
            opt_debug = True
        elif arg == '-i' or arg == '--install':
            opt_install = True
        elif arg == '-a' or arg == '--app':
            i += 1
            if i == argc:
                raise Exception("--app/-a missing parameter")
            opt_app = sys.argv[i]
        elif arg == '-s' or arg == '--software':
            i += 1
            if i == argc:
                raise Exception("--software/-s missing parameter")
            opt_software = sys.argv[i]
        elif arg == '-p' or arg == '--protocol':
            i += 1
            if i == argc:
                raise Exception("--protocol/-p missing parameter")
            opt_protocol = sys.argv[i]
        i += 1
    return True

# main
if not parse_opts():
    exit(0)
cmdline = build_cmd()
if subprocess.call(cmdline, shell=True) == 0:
    print("build ok: {0}".format(cmdline))
    exit(0)
else:
    print("build failed: {0}".format(cmdline))
    exit(1)

# 如果版本发生变化需要修改这里的版本号，以及build.py中的版本号
VER    := 1.0.20250603
OS     := $(shell go env GOOS)
ARCH   := $(shell go env GOARCH)
EXEEXT ?= 
ifeq (windows,$(OS))
EXEEXT := .exe
endif
APP    := costrict-host$(EXEEXT)

build:
	python ./build.py --software $(VER)

install:
	python ./build.py --software $(VER) --install

.PHONY: build install

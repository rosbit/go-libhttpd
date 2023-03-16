SHELL=/bin/bash

build:
	@if [ "$t" == "" ]; then \
		echo "Usage: make build t=TARGET"; \
	else \
		if [ "$o" == "macos" ]; then \
			CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 GOPATH=`pwd` go build -ldflags "-X 'main.buildTime=`date '+%F %T'`' -X 'main.osInfo=`uname -sr`' -X 'main.goInfo=`go version`' -extldflags -static" $t; \
		else \
            GOPATH=`pwd` go build -ldflags "-X 'main.buildTime=`date '+%F %T'`' -X 'main.osInfo=`uname -sr`' -X 'main.goInfo=`go version`' -linkmode external -extldflags -static" $t; \
		fi; \
	fi

get:
	@if [ "$t" == "" ]; then \
		echo "Usage: make get t=TARGET"; \
	else \
		GOPATH=`pwd` go get $t; \
	fi

so:
	@if [ "$t" == "" ]; then \
		echo "Usage: make so t=TARGET"; \
	else \
		go build -buildmode=c-shared -tags timetzdata -o $t; \
	fi

plugin:
	@if [ "$t" == "" ]; then \
		echo "Usage: make plugin t=TARGET"; \
	else \
		GOPATH=`pwd` go build -buildmode=plugin $t; \
	fi

help:
	@echo "   make t=<target>           build <target> to executable"
	@echo "   make get t=<target>       git pull source of <target>"
	@echo "   make so t=<target>        build <target> to C shared lib"
	@echo "   make plugin t=<target>    build <target> to golang plugin"

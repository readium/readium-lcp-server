
ROOT_DIR=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

BUILD_DIR=$(ROOT_DIR)/build

UNAME_S= $(shell uname -s)

BUILD_PROD=false
LIBUSERKEY_PATH=/
USERKEYH_PATH=/
USERKEYGO_PATH=/

export READIUM_LCPSERVER_CONFIG := $(BUILD_DIR)/config.yaml
export READIUM_LSDSERVER_CONFIG := $(BUILD_DIR)/config.yaml
export READIUM_FRONTEND_CONFIG := $(BUILD_DIR)/config.yaml
export GOPATH= $(BUILD_DIR)

lcpencrypt=lcpencrypt
lcpserver=lcpserver
lsdserver=lsdserver
frontend=frontend
frontend_manage=frontend/manage

# NODE_VERSION=18.12.1
ifeq ($(UNAME_S), Linux)
# NODE_URL="https://nodejs.org/dist/v$(NODE_VERSION)/node-v$(NODE_VERSION)-linux-arm64.tar.xz"
	SED_I=sed -i
else
# NODE_URL="https://nodejs.org/dist/v$(NODE_VERSION)/node-v$(NODE_VERSION)-darwin-arm64.tar.xz"
	SED_I=sed -i ''
endif

#LDFLAGS=-extldflags=-static
LDFLAGS=

#amd64
CC=GOARCH=arm64 go install -x -ldflags '-linkmode external -w -extldflags "-static"'
 #-ldflags="$(LDFLAGS)"

#node
.PHONY: all run prepare clean

all: $(lcpencrypt) $(lcpserver) $(lsdserver) $(frontend) $(frontend_manage)

clean:
	@if [ "$(BUILD_PROD)" = "true" ]; then\
		echo "RM $(LIBUSERKEY_PATH)"; 																							\
		rm -f license/libuserkey.a; 																		\
		echo "RM $(USERKEYH_PATH)"; 																								\
		rm -f license/userkey.h;																				\
		echo "RM $(USERKEYGO_PATH)";																								\
		mv $(BUILD_DIR)/user_key.go.backup license/user_key.go;											\
	fi

	@rm -rf $(BUILD_DIR) 2>/dev/null || true
	@rm -rf $(ROOT_DIR)/$(frontend_manage)/node_modules
	@rm -rf $(ROOT_DIR)/$(frontend_manage)/dist

# node:
# 	open $(NODE_URL)

prepare:
	mkdir -p $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)/cert
	mkdir -p $(BUILD_DIR)/db
	mkdir -p $(BUILD_DIR)/files
	mkdir -p $(BUILD_DIR)/files/storage
	cp $(ROOT_DIR)/test/cert/cert-edrlab-test.pem $(BUILD_DIR)/cert/.
	cp $(ROOT_DIR)/test/cert/privkey-edrlab-test.pem $(BUILD_DIR)/cert/.
	mkdir -p $(BUILD_DIR)/log
	mkdir -p $(BUILD_DIR)/frontend/manage
	sed 's~<LCP_HOME>~$(BUILD_DIR)~g' < $(ROOT_DIR)/test/config.yaml > $(BUILD_DIR)/config.yaml
	echo "adm_username:\$$apr1\$$bxwn8jim\$$kbfYFRgbBlKDWpAvd2tHW." > $(BUILD_DIR)/htpasswd
	@if [ "$(BUILD_PROD)" = "true" ]; then\
		echo "COPY $(LIBUSERKEY_PATH)"; 																			\
		cp $(LIBUSERKEY_PATH) license/.; 																			\
		echo "COPY $(USERKEYH_PATH)"; 																				\
		cp $(USERKEYH_PATH) license/.;																				\
		echo "COPY $(USERKEYGO_PATH)"; 																				\
		cp license/user_key.go $(BUILD_DIR)/user_key.go.backup; 	\
		cp $(USERKEYGO_PATH) license/.;													\
	fi

$(lcpencrypt): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(lcpserver): prepare
	GOPATH=$(GOPATH) CGO_ENABLED=1 $(CC) ./$@

$(lsdserver): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(frontend): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(frontend_manage): prepare
		cd ./$@ \
		&& cp package.json package.json.backup \
		&& $(SED_I) '/\"lite-server\"\:/d' package.json \
		&& $(SED_I) 's/git\:/https\:/g' package.json \
		&& npm install --legacy-peer-deps \
		&& npm update \
		&& npm run clean \
		&& npm run build-css \
		&& npm run copy-templates \
		&& $(SED_I) '/es2015/d' node_modules/@types/node/index.d.ts \
		&& node_modules/.bin/tsc \
		&& mv package.json.backup package.json \
		&& cp -r . $(BUILD_DIR)/frontend/manage/.
		

run:
	rm -f $(BUILD_DIR)/run.sh
	echo "#! /usr/bin/env sh" >> $(BUILD_DIR)/run.sh
	echo "READIUM_LCPSERVER_CONFIG=$(READIUM_LCPSERVER_CONFIG) $(BUILD_DIR)/bin/$(lcpserver) > $(BUILD_DIR)/log/$(lcpserver).log &" >> $(BUILD_DIR)/run.sh
	echo "READIUM_LSDSERVER_CONFIG=$(READIUM_LSDSERVER_CONFIG) $(BUILD_DIR)/bin/$(lsdserver) > $(BUILD_DIR)/log/$(lsdserver).log &" >> $(BUILD_DIR)/run.sh
	echo "READIUM_FRONTEND_CONFIG=$(READIUM_FRONTEND_CONFIG) $(BUILD_DIR)/bin/$(frontend) > $(BUILD_DIR)/log/$(frontend).logi &" >> $(BUILD_DIR)/run.sh
	echo "wait" >> $(BUILD_DIR)/run.sh
	chmod +x $(BUILD_DIR)/run.sh
	@open http://127.0.0.1:8991/
	sh $(BUILD_DIR)/run.sh



ROOT_DIR=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

BUILD_DIR=$(ROOT_DIR)/build

export READIUM_LCPSERVER_CONFIG := $(BUILD_DIR)/config.yaml
export READIUM_LSDSERVER_CONFIG := $(BUILD_DIR)/config.yaml
export READIUM_FRONTEND_CONFIG := $(BUILD_DIR)/config.yaml
export GOPATH= $(BUILD_DIR)

lcpencrypt=lcpencrypt
lcpserver=lcpserver
lsdserver=lsdserver
frontend=frontend
frontend_manage=frontend/manage


CC=go install 

.PHONY: all

all: $(lcpencrypt) $(lcpserver) $(lsdserver) $(frontend) #$(frontend_manage)

clean:
	@rm -rf $(BUILD_DIR) 2>/dev/null || true

prepare:
	mkdir -p $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)/cert
	mkdir -p $(BUILD_DIR)/db
	mkdir -p $(BUILD_DIR)/files
	mkdir -p $(BUILD_DIR)/files/storage
	cp $(ROOT_DIR)/test/cert/cert-edrlab-test.pem $(BUILD_DIR)/cert/.
	cp $(ROOT_DIR)/test/cert/privkey-edrlab-test.pem $(BUILD_DIR)/cert/.
	mkdir -p $(BUILD_DIR)/log
	sed 's~<LCP_HOME>~$(BUILD_DIR)~g' < $(ROOT_DIR)/test/config.yaml > $(BUILD_DIR)/config.yaml
	echo "adm_username:$$apr1$$bxwn8jim$$kbfYFRgbBlKDWpAvd2tHW." > $(BUILD_DIR)/htpasswd

$(lcpencrypt): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(lcpserver): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(lsdserver): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(frontend): prepare
	GOPATH=$(GOPATH) $(CC) ./$@

$(frontend_manage): prepare
	cd ./$@ && npm install

run:
	READIUM_LCPSERVER_CONFIG=$(READIUM_LCPSERVER_CONFIG) $(BUILD_DIR)/bin/$(lcpserver) > $(BUILD_DIR)/log/$(lcpserver).log &
	READIUM_LSDSERVER_CONFIG=$(READIUM_LSDSERVER_CONFIG) $(BUILD_DIR)/bin/$(lsdserver) > $(BUILD_DIR)/log/$(lsdserver).log &
	READIUM_FRONTEND_CONFIG=$(READIUM_FRONTEND_CONFIG) $(BUILD_DIR)/bin/$(frontend) > $(BUILD_DIR)/log/$(frontend).log &


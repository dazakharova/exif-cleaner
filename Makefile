webui:
	$(MAKE) -C services/webui $(TARGET)

# Run a target inside stripper
stripper:
	$(MAKE) -C services/stripper $(TARGET)

tidy:
	TARGET=tidy $(MAKE) webui
	TARGET=tidy $(MAKE) stripper

verify:
	TARGET=verify $(MAKE) webui
	TARGET=verify $(MAKE) stripper

vet:
	TARGET=vet $(MAKE) webui
	TARGET=vet $(MAKE) stripper

staticcheck:
	TARGET=staticcheck $(MAKE) webui
	TARGET=staticcheck $(MAKE) stripper

test:
	TARGET=test $(MAKE) webui
	TARGET=test $(MAKE) stripper

check:
	TARGET=check $(MAKE) webui
	TARGET=check $(MAKE) stripper

build:
	TARGET=build $(MAKE) webui
	TARGET=build $(MAKE) stripper
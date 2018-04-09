VERSION=1.70
PACKAGE=amahi-anywhere
RPMBUILDDIR=$(HOME)/rpmbuild

# PLATFORM is a build-time check for platform -- fedora or ubuntu or forced to other (e.g. darwin)
# compile for production release
PLATFORM=$(shell head -1 /etc/system-release | cut -f1 -d" " | tr "[:upper:]" "[:lower:]")

export GOPATH=$(shell pwd)

all: build-production

PLATFORM=$(shell head -1 /etc/system-release | cut -f1 -d" " | tr "[:upper:]" "[:lower:]")

build-production:
	go get -tags "$(PLATFORM) linux" fs
	@rm -f bin/amahi-anywhere
	@ln bin/fs bin/amahi-anywhere

build-development:
	@echo "********************************************************"
	@echo "************** BUILDING FOR DEVELOPMENT/DEBUG RELEASE"
	@echo "********************************************************"
	go get -tags "development $(PLATFORM) linux" fs
	@rm -f bin/amahi-anywhere
	@ln bin/fs bin/amahi-anywhere

mac:
	go get -tags "development darwin" fs
	@rm -f bin/amahi-anywhere
	@ln bin/fs bin/amahi-anywhere

race:
	go build -race fs
	mkdir -p bin/
	mv -f fs bin/

clean:
	go clean -i -x fs
	rm -rf pkg bin

dist: update-header
	(mkdir -p release && cd release && mkdir -p $(PACKAGE)-$(VERSION))
	rsync -a src $(PACKAGE).spec Makefile $(PACKAGE).service debian release/$(PACKAGE)-$(VERSION)/
	(cd release && tar -czvf $(PACKAGE)-$(VERSION).tar.gz $(PACKAGE)-$(VERSION))
	(cd release && rm -rf $(PACKAGE)-$(VERSION))

update-header:
	sed -i -e "s/Version:.*/Version:\t$(VERSION)/" $(PACKAGE).spec
	sed -i -e "s/const VERSION\s*=.*/const VERSION = \"$(VERSION)\"/" src/fs/fs.go

# build the rpm for production
rpm: dist
	(cd release && rpmbuild $(SIGN) -D 'BUILD_TYPE production' -ta $(PACKAGE)-$(VERSION).tar.gz)
	mv $(RPMBUILDDIR)/RPMS/*/$(PACKAGE)-$(VERSION)-*.rpm release/
	mv $(RPMBUILDDIR)/SRPMS/$(PACKAGE)-$(VERSION)-*.src.rpm release/

# build the rpm for development
rpm-dev: dist
	@echo "********************************************************"
	@echo "************** BUILDING FOR DEVELOPMENT/DEBUG RELEASE"
	@echo "********************************************************"
	(cd release && rpmbuild $(SIGN) -D 'BUILD_TYPE development' -ta $(PACKAGE)-$(VERSION).tar.gz)
	mv $(RPMBUILDDIR)/RPMS/*/$(PACKAGE)-$(VERSION)-*.rpm release/
	mv $(RPMBUILDDIR)/SRPMS/$(PACKAGE)-$(VERSION)-*.src.rpm release/

update-lib:
	-git rm --cached src/golang.org/x/net
	git submodule init
	git submodule update --merge
	(cd src/github.com/amahi/go-metadata && git pull origin master && git checkout master)
	(cd src/github.com/amahi/go-themoviedb && git pull origin master && git checkout master)
	(cd src/github.com/amahi/go-tvrage && git pull origin master && git checkout master)
	(cd src/github.com/go-sql-driver/mysql && git pull origin master && git checkout master)
	(cd src/github.com/mattn/go-sqlite3 && git pull origin master && git checkout master)


deb-dev: dist
	@echo "****************************************************"
	@echo "************** BUILDING FOR DEVELOPMENT RELEASE"
	@echo "****************************************************"
	(cd release && ln -sf $(PACKAGE)-$(VERSION).tar.gz $(PACKAGE)_$(VERSION).orig.tar.gz)
	(cd release && tar -zxf $(PACKAGE)_$(VERSION).orig.tar.gz)
	(cd release/$(PACKAGE)-$(VERSION)/debian && PLATFORM=ubuntu BUILD_TYPE=production debuild -uc -us)

deb: dist
	(cd release && ln -sf $(PACKAGE)-$(VERSION).tar.gz $(PACKAGE)_$(VERSION).orig.tar.gz)
	(cd release && tar -zxf $(PACKAGE)_$(VERSION).orig.tar.gz)
	(cd release/$(PACKAGE)-$(VERSION)/debian && PLATFORM=ubuntu BUILD_TYPE=development debuild -uc -us)

install: build
	mkdir -p $(DESTDIR)/usr/bin
	install bin/fs $(DESTDIR)/usr/bin/amahi-anywhere

install-rpm-deps:
	sudo yum -y install golang

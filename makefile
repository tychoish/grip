VERSION?=main
.PHONY:upgrade-fun
upgrade-fun:
	go get github.com/tychoish/fun@$(VERSION)
	for i in $(shell find . -name "go.mod"); do pushd $$(dirname $$i); echo $(dirname $i); go get github.com/tychoish/fun@$(VERSION); go mod tidy; popd; done

upgrade-grip:
	git push --tags
	for i in $(shell find . -name "go.mod"); do pushd $$(dirname $$i); echo $(dirname $i); go get github.com/tychoish/grip@$(VERSION); go mod tidy; popd; done
	git add ./x
	git commit -m "post-bump: update x/ deps"

VERSION?=main
.PHONY:upgrade-fun go-mod-tidy upgrade-grip

upgrade-fun:
	go get github.com/tychoish/fun@$(VERSION)
	for i in $$(find . -name "go.mod"); do pushd $$(dirname $$i); go get github.com/tychoish/fun@$(VERSION); go mod tidy; go build ./... ; popd; done

upgrade-grip:
	git push --tags
	for i in $$(find . -name "go.mod"); do pushd $$(dirname $$i); go get github.com/tychoish/grip@$(VERSION); go mod tidy; go build ./...; popd; done
	git add ./x
	git commit -m "post-bump: update x/ deps"

go-mod-tidy:
	for i in $$(find . -name "go.mod"); do pushd $$(dirname $$i);  go mod tidy; popd; done

test-compile:
	for i in $$(find . -name "go.mod"); do pushd $$(dirname $$i); go test ./... -run=compileCheck; popd; done

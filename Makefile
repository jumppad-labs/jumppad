git_commit = $(shell git log -1 --pretty=format:"%H")

test_unit:
	go test -v -race ./...

test_functional:
	go run main.go purge
	go run main.go test ./examples/container
	
	go run main.go purge
	go run main.go test ./examples/build

	go run main.go purge
	go run main.go test ./examples/docs

	go run main.go purge
	go run main.go test ./examples/nomad

	go run main.go purge
	go run main.go test ./examples/single_k3s_cluster
	
	go run main.go purge
	go run main.go test ./examples/multiple_k3s_clusters
	
	go run main.go purge
	go run main.go test ./examples/exec
	
	go run main.go purge
	go run main.go test ./examples/certificates
	
	go run main.go purge
	go run main.go test ./examples/terraform
	
	go run main.go purge
	go run main.go test ./examples/registiries

test_e2e_cmd: install_local
	jumppad up --no-browser ./examples/single_k3s_cluster
	jumppad down

dagger_build:
	dagger call -m "./.dagger" all \
		--output=./all_archive \
		--src=. \
		--notorize-cert=${QUILL_SIGN_P12} \
		--notorize-cert-password=QUILL_SIGN_PASSWORD \
		--notorize-key=${QUILL_NOTARY_KEY} \
		--notorize-id=${QUILL_NOTARY_KEY_ID} \
		--notorize-issuer=${QUILL_NOTARY_ISSUER}

dagger_release:
	dagger call -m "./.dagger" release \
		--github-token=GITHUB_TOKEN \
		--archives=./all_archive \
		--src=.  

generate_mocks:
	go install github.com/vektra/mockery/v2@v2.20.0
	go generate ./...

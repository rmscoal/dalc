.PHONY: download-kind
download-kind:
	chmod +x install-kind.sh
	./install-kind.sh

.PHONY: create-cluster
create-cluster: download-kind
	kind create cluster --config kind-cluster.yaml

.PHONY: deploy
deploy:
	kubectl apply -f k8s

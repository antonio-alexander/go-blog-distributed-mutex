## ----------------------------------------------------------------------
## This Makefile contains multiple commands used for local dev ops 
## ----------------------------------------------------------------------

docker_args=-l error #default args, supresses warnings

.PHONY: help dep run stop clean

# REFERENCE: https://stackoverflow.com/questions/16931770/makefile4-missing-separator-stop
help: ## - Show this help.
	@sed -ne '/@sed/!s/## //p' $(MAKEFILE_LIST)

dep: ## run all dependencies
	@docker ${docker_args} compose up --detach --wait

run: dep ## run all dependencies
	@go run ./cmd/main.go

stop: ## stop all dependencies and services
	@docker ${docker_args} compose down

clean: ## stop all dependencies and services and clear volumes
	@docker ${docker_args} compose down --volumes --remove-orphans

default:
    @just --list

# All tests here run in the host environment, not on Windows.
check:
    go test ./... -count=1
    go vet ./...
    GOOS=windows GOARCH=amd64 go vet ./...
    GOOS=windows GOARCH=arm64 go vet ./...

build:
    go tool mygo build -platform windows/amd64,windows/arm64

# Optional Native UI layout previews; these do not test desktop embedding.
preview:
    COUNTDOWNER_PREVIEW_DIR=build/previews go test . -run TestLanguageOrderAndNativeViews -count=1

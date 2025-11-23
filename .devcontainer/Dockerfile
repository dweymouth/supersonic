FROM mcr.microsoft.com/devcontainers/go:2-1.24-bookworm

RUN apt-get update && apt install -y libmpv-dev gcc libegl1-mesa-dev xorg-dev

USER vscode
RUN go install fyne.io/fyne/v2/cmd/fyne@latest

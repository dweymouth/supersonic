FROM mcr.microsoft.com/devcontainers/go:2-1.24-bookworm

RUN apt update && apt install -y libmpv-dev gcc libegl1-mesa-dev xorg-dev acl
RUN usermod -aG audio vscode

USER vscode
RUN go install fyne.io/fyne/v2/cmd/fyne@latest

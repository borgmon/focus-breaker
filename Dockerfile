FROM golang:1.25-trixie

RUN apt update && apt install -y gcc libgl1-mesa-dev xorg-dev libasound2-dev xz-utils
RUN go install fyne.io/tools/cmd/fyne@latest

COPY . .

RUN fyne package -os linux -icon assets/Icon.png -release
# ENTRYPOINT ["tail", "-f", "/dev/null"]

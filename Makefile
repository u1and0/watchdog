# Raspberry pi zero 2用にビルド
build:
	GOOS=linux GOARCH=arm GOARM=6 go build -o watchdog.arm .

# デフォルトターゲット
all: build

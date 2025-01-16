`watchdog` is a continuous health monitoring CLI tool

Usage:
  health-monitor [flags]

Flags:
  -e, --endpoint string         Target endpoint to monitor
  -h, --help                    help for health-monitor
  -M, --max-interval duration   Maximum check interval (default 1h0m0s)
  -m, --min-interval duration   Minimum check interval (default 1m0s)
  -u, --user string             Slack user ID to mention in error messages
  -w, --webhook string          Slack webhook URL

```
# minimal option
$ watchdog -e http://localhost:8080 -w $SLACK_WEBHOOK
```


```
# full option
$ watchdog -e http://localhost:8080 -w $SLACK_WEBHOOK -m 2s -M 16s -u $SLACK_USER
```

上記オプションでは、最小間隔2秒、最大間隔16秒ごとにendpointに指定したURLへアクセスを試みます。
エラー発生時には、webhookに指定したチャンネルへ、userで指定したユーザー向けにリプライを飛ばします。
エラー発生時は指数バックオフの考えを使って、リトライ間隔を最大間隔まで伸ばします。

# セットアップ

## デーモン化
バックグラウンドプロセスとして常駐してもらうためにデーモン化を行います。


```sh
$ /etc/systemd/system/watchdog.service
```

## 環境変数

```.env
SLACK_WEBHOOK="https://hooks.slack.com/services/XXXXXXXXX/YYYYYYYYYYY/ZZZZZZZZZZZZZZZZZZZZZZZZ"
SLACK_USER="XXXXXXXXX"
```

## インストール

作成したバイナリ、サービスファイル、環境変数ファイルをサーバーへアップロードします。

```sh
scp ./watchdog.service myuser@myhost:/etc/systemd/system/
scp ./.env myuser@myhost:/home/myuser/
scp ./watchdog myuser@myhost:/usr/local/bin/
```

## デプロイ

デーモンを有効化します。

```sh
$ sudo systemctl daemon-reload
$ sudo systemctl enable watchdog
$ sudo systemctl start watchdog
```
